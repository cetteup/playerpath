package main

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cetteup/playerpath/cmd/importer/internal/config"
	"github.com/cetteup/playerpath/cmd/importer/internal/handler"
	"github.com/cetteup/playerpath/cmd/importer/internal/options"
	"github.com/cetteup/playerpath/internal/domain/player/sql"
	"github.com/cetteup/playerpath/internal/domain/provider"
	"github.com/cetteup/playerpath/internal/pkg/registry"
	"github.com/cetteup/playerpath/internal/sqlutil"
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

	registryBaseURL := cmp.Or(cfg.RegistryBaseURL, registry.BaseURL)
	client := registry.NewClient(registryBaseURL, 10*time.Second)
	repository := sql.NewRepository(db)

	h := handler.NewHandler(
		client,
		repository,
		[]provider.Provider{
			provider.BF2Hub,
			provider.PlayBF2,
			provider.OpenSpy,
			provider.B2BF2,
			provider.Gameppy,
		},
		opts.BatchSize,
	)

	// Trigger import once on startup
	once := make(chan struct{}, 1)
	once <- struct{}{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-once:
		case <-time.After(opts.Interval):
		}

		log.Info().Msgf("Importing players via %s", registryBaseURL)

		if err = h.ImportPlayers(ctx); err != nil {
			log.Error().
				Err(err).
				Msgf("Failed to import players via %s", registryBaseURL)
		}
	}
}
