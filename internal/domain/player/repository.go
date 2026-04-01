package player

import (
	"context"
	"errors"
)

var (
	ErrPlayerNotFound       = errors.New("player not found")
	ErrMultiplePlayersFound = errors.New("found multiple players")
)

type Repository interface {
	UpsertMany(ctx context.Context, players []Player) (int, error)
	FindByPID(ctx context.Context, pid int) (Player, error)
}
