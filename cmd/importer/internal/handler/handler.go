package handler

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cetteup/playerpath/internal/domain/player"
	"github.com/cetteup/playerpath/internal/domain/provider"
	"github.com/cetteup/playerpath/internal/pkg/registry"
)

type Client interface {
	GetPlayers(ctx context.Context, filters ...registry.FilterFunc) (registry.PageIterator, error)
}

type Handler struct {
	client     Client
	repository player.Repository

	providers []provider.Provider
	batchSize int

	markers map[provider.Provider]string
}

func NewHandler(client Client, repository player.Repository, providers []provider.Provider, batchSize int) *Handler {
	markers := make(map[provider.Provider]string, len(providers))
	for _, p := range providers {
		markers[p] = ""
	}

	return &Handler{
		client:     client,
		repository: repository,
		providers:  providers,
		batchSize:  batchSize,
		markers:    markers,
	}
}

func (s *Handler) ImportPlayers(ctx context.Context) error {
	for _, pv := range s.providers {
		marker, err := s.importPlayers(ctx, pv, s.markers[pv])
		if err != nil {
			return err
		}
		s.markers[pv] = marker
	}

	return nil
}

func (s *Handler) importPlayers(ctx context.Context, pv provider.Provider, marker string) (string, error) {
	log.Debug().Msgf("Importing players from %s", pv)

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// Load batches asynchronously, allowing us to collect the next batch while upserting the current one
	players := make(chan player.Player, s.batchSize)
	go func() {
		defer close(players)

		var err error
		if marker, err = s.load(ctx, pv, marker, players); err != nil {
			cancel(err)
		}
	}()

	stats := struct {
		processed int
		imported  int
	}{}

	batch := make([]player.Player, 0, s.batchSize)
	for p := range players {
		stats.processed++
		batch = append(batch, player.Player{
			PID:      p.PID,
			Nick:     p.Nick,
			Provider: pv,
			Imported: time.Now().UTC(),
		})

		if len(batch) == cap(batch) {
			modified, err := s.repository.UpsertMany(ctx, batch)
			if err != nil {
				return "", err
			}

			stats.imported += modified
			batch = batch[:0]
		}
	}

	if err := context.Cause(ctx); err != nil {
		return "", err
	}

	// Upsert any remaining, incomplete batch
	if len(batch) > 0 {
		modified, err := s.repository.UpsertMany(ctx, batch)
		if err != nil {
			return "", err
		}

		stats.imported += modified
	}

	log.Info().
		Int("processed", stats.processed).
		Msgf("Imported %d players from %s", stats.imported, pv)

	return marker, nil
}

func (s *Handler) load(
	ctx context.Context,
	pv provider.Provider,
	marker string,
	out chan<- player.Player,
) (string, error) {
	players, err := s.client.GetPlayers(ctx, registry.WithProviderFilter(strings.ToLower(pv.String())))
	if err != nil {
		return "", err
	}

	for p := range players.After(marker) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case out <- player.Player{
			PID:      p.PID,
			Nick:     p.Nick,
			Provider: pv,
			Imported: time.Now().UTC(),
		}:
			marker = p.ID
		}
	}

	if err = players.Err(); err != nil {
		return "", err
	}

	return marker, nil
}
