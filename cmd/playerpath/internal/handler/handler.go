package handler

import (
	"context"
	"net/http"

	"github.com/cetteup/playerpath/internal/domain/player"
	"github.com/cetteup/playerpath/internal/domain/provider"
)

type repository interface {
	FindByPID(ctx context.Context, pid int) (player.Player, error)
}

type Handler struct {
	repository repository
	provider   provider.Provider

	client *http.Client
}

func NewHandler(repository repository, provider provider.Provider) *Handler {
	return &Handler{
		repository: repository,
		provider:   provider,
		client: &http.Client{
			// Don't follow redirects, just return first response (mimic proxy behaviour)
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}
