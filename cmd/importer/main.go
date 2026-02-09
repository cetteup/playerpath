package main

import (
	"cmp"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cetteup/playerpath/cmd/importer/internal/config"
	"github.com/cetteup/playerpath/cmd/importer/internal/opendata"
	"github.com/cetteup/playerpath/cmd/importer/internal/options"
	"github.com/cetteup/playerpath/cmd/importer/internal/registry"
	"github.com/cetteup/playerpath/internal/domain/player"
	"github.com/cetteup/playerpath/internal/domain/player/sql"
	"github.com/cetteup/playerpath/internal/domain/provider"
	"github.com/cetteup/playerpath/internal/sqlutil"
	"github.com/cetteup/playerpath/internal/trace"
)

var (
	buildVersion = "development"
	buildCommit  = "uncommitted"
	buildTime    = "unknown"
)

func main() {
	version := fmt.Sprintf("importer %s (%s) built at %s", buildVersion, buildCommit, buildTime)
	opts := options.Init()

	// Print version and exit
	if opts.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    !opts.ColorizeLogs,
		TimeFormat: time.RFC3339,
	})
	if opts.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("config", opts.ConfigPath).
			Msg("Failed to read config file")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db := sqlutil.Connect(
		cfg.Database.Hostname,
		cfg.Database.DatabaseName,
		cfg.Database.Username,
		cfg.Database.Password,
	)
	defer func() {
		err2 := db.Close()
		if err2 != nil {
			log.Error().
				Err(err2).
				Msg("Failed to close database connection")
		}
	}()

	repository := sql.NewRepository(db)

	var source Source
	if isUsableURL(opts.Source) {
		source = registry.NewClient(opts.Source, 10*time.Second)
	} else if isDirPath(opts.Source) {
		source = opendata.NewLoader(opts.Source)
	} else {
		log.Fatal().Msg("Source must be a valid URL or a directory path")
	}

	log.Info().Msgf("Importing players from %s", opts.Source)

	err = load(ctx, source, repository, opts.BatchSize)
	if err != nil {
		log.Fatal().
			Err(err).
			Msgf("Failed to import players from %s", opts.Source)
	}
}

type Source interface {
	GetPlayers(ctx context.Context, provider string, cb func(ctx context.Context, pid int, nick string) error) error
}

func load(ctx context.Context, source Source, repository player.Repository, batchSize int) error {
	providers := []provider.Provider{provider.BF2Hub, provider.PlayBF2, provider.OpenSpy, provider.B2BF2}
	for _, pv := range providers {
		stats := struct {
			processed int
			imported  int
			added     int
			updated   int
		}{}

		batch := make([]player.Player, 0, batchSize)
		err := source.GetPlayers(ctx, pv.String(), func(ctx context.Context, pid int, nick string) error {
			stats.processed++
			batch = append(batch, player.Player{
				PID:      pid,
				Nick:     nick,
				Provider: pv,
				Imported: time.Now().UTC(),
			})

			if len(batch) == cap(batch) {
				added, updated, err2 := upsert(ctx, repository, pv, batch)
				if err2 != nil {
					return err2
				}

				stats.imported += added + updated
				stats.added += added
				stats.updated += updated

				batch = batch[:0]
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to import players from %s: %w", pv, err)
		}

		// Upsert any remaining, incomplete batch
		if len(batch) > 0 {
			added, updated, err2 := upsert(ctx, repository, pv, batch)
			if err2 != nil {
				return err2
			}

			stats.imported += added + updated
			stats.added += added
			stats.updated += updated
		}

		log.Info().
			Int("processed", stats.processed).
			Int("added", stats.added).
			Int("updated", stats.updated).
			Msgf("Imported %d players from %s", stats.imported, pv)
	}

	return nil
}

func upsert(ctx context.Context, repository player.Repository, pv provider.Provider, players []player.Player) (int, int, error) {
	if len(players) == 0 {
		return 0, 0, nil
	}

	// Ensure players are sorted ascending by PID
	slices.SortFunc(players, func(a, b player.Player) int {
		return cmp.Compare(a.PID, b.PID)
	})

	existing, err := repository.FindByProviderBetweenPIDs(ctx, pv, players[0].PID, players[len(players)-1].PID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to find existing players: %w", err)
	}

	// Create map for consistently fast lookups
	catalog := make(map[int]string, len(existing))
	for _, p := range existing {
		catalog[p.PID] = p.Nick
	}

	insert := make([]player.Player, 0, len(players))
	update := make([]player.Player, 0)
	for _, p := range players {
		if nick, exists := catalog[p.PID]; !exists {
			insert = append(insert, p)
		} else if p.Nick != nick {
			update = append(update, p)
		}
	}

	if len(insert) != 0 {
		err2 := repository.InsertMany(ctx, insert)
		if err2 != nil {
			return 0, 0, fmt.Errorf("failed to insert new players: %w", err2)
		}

		for _, p := range insert {
			log.Debug().
				Int(trace.LogPlayerPID, p.PID).
				Str(trace.LogPlayerNick, p.Nick).
				Stringer(trace.LogProvider, pv).
				Msg("Added new player")
		}
	}

	if len(update) != 0 {
		err2 := repository.UpdateMany(ctx, update)
		if err2 != nil {
			return 0, 0, fmt.Errorf("failed to update existing players: %w", err2)
		}

		for _, p := range update {
			log.Debug().
				Int(trace.LogPlayerPID, p.PID).
				Str(trace.LogPlayerNick, p.Nick).
				Stringer(trace.LogProvider, pv).
				Msg("Updated existing player")
		}
	}

	return len(insert), len(update), nil
}

func isUsableURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func isDirPath(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
