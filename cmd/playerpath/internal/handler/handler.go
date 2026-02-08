package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/cetteup/playerpath/cmd/playerpath/modify"
	"github.com/cetteup/playerpath/internal/domain/player"
	"github.com/cetteup/playerpath/internal/domain/provider"
	"github.com/cetteup/playerpath/internal/trace"
)

type repository interface {
	FindByPID(ctx context.Context, pid int) (player.Player, error)
}

type Handler struct {
	repository repository
	servers    map[string]provider.Provider
	provider   provider.Provider

	modifiers struct {
		request  []modify.RequestModifier
		response []modify.ResponseModifier
	}

	client *http.Client
}

func NewHandler(repository repository, servers map[string]provider.Provider, provider provider.Provider) *Handler {
	return &Handler{
		repository: repository,
		servers:    servers,
		provider:   provider,
		client: &http.Client{
			Transport: &http.Transport{
				DisableCompression: true,
			},
			// Don't follow redirects, just return first response (mimic proxy behaviour)
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (h *Handler) WithModifier(modifiers ...modify.Modifier) {
	for _, modifier := range modifiers {
		if modifier.Type() == modify.ModifierTypeRequest {
			m, ok := modifier.(modify.RequestModifier)
			if ok {
				h.modifiers.request = append(h.modifiers.request, m)
			}
		}

		if modifier.Type() == modify.ModifierTypeResponse {
			m, ok := modifier.(modify.ResponseModifier)
			if ok {
				h.modifiers.response = append(h.modifiers.response, m)
			}
		}
	}
}

func (h *Handler) determineProvider(ctx context.Context, pid int, serverIP string) (provider.Provider, error) {
	// Primarily determine provider based on player
	pv, err := h.getPlayerProvider(ctx, pid)
	if err != nil {
		return provider.ProviderUnknown, err
	} else if pv != provider.ProviderUnknown {
		return pv, nil
	}

	// Alternative use server's default provider
	pv = h.getServerProvider(serverIP)
	if pv != provider.ProviderUnknown {
		return pv, nil
	}

	// Finally fall back to overall default provider
	return h.provider, nil
}

func (h *Handler) getPlayerProvider(ctx context.Context, pid int) (provider.Provider, error) {
	p, err := h.repository.FindByPID(ctx, pid)
	if err != nil {
		if errors.Is(err, player.ErrPlayerNotFound) {
			log.Warn().
				Int(trace.LogPlayerPID, pid).
				Msg("Player not found, deferring provider selection")
			return provider.ProviderUnknown, nil
		}
		if errors.Is(err, player.ErrMultiplePlayersFound) {
			log.Warn().
				Int(trace.LogPlayerPID, pid).
				Msg("Found multiple players, deferring provider selection")
			return provider.ProviderUnknown, nil
		}
		return provider.ProviderUnknown, err
	}

	return p.Provider, nil
}

func (h *Handler) getServerProvider(ip string) provider.Provider {
	pv, ok := h.servers[ip]
	if !ok {
		if len(h.servers) > 0 {
			// Only log warning if any servers have been configured, which is totally optional
			// (simple use cases work fine with just a default provider)
			log.Warn().
				Str("ip", ip).
				Msg("Server not configured, deferring provider selection")
		}
		return provider.ProviderUnknown
	}

	return pv
}

func (h *Handler) getServerOrDefaultProvider(ip string) provider.Provider {
	pv := h.getServerProvider(ip)
	if pv != provider.ProviderUnknown {
		return pv
	}

	return h.provider
}
