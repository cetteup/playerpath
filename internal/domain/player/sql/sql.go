package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"

	"github.com/cetteup/playerpath/internal/domain/player"
)

const (
	playerTable = "players"

	columnPID      = "pid"
	columnNick     = "nick"
	columnProvider = "provider"
	columnImported = "imported"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) UpsertMany(ctx context.Context, players []player.Player) (int, error) {
	query := sq.
		Insert(playerTable).
		Columns(
			columnPID,
			columnNick,
			columnProvider,
			columnImported,
		).
		// Provider is part of the primary key, meaning there's no way to trigger an update with a different provider
		// Which is why the provider column not included in the upsert columns
		Suffix(fmt.Sprintf("ON DUPLICATE KEY UPDATE %s", strings.Join([]string{
			fmt.Sprintf("%[1]s = VALUES(%[1]s)", columnNick),
		}, ", ")))

	for _, p := range players {
		query = query.Values(
			p.PID,
			p.Nick,
			p.Provider,
			p.Imported,
		)
	}

	result, err := query.RunWith(r.db).ExecContext(ctx)
	if err != nil {
		return 0, err
	}

	modified, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(modified), nil
}

func (r *Repository) FindByPID(ctx context.Context, pid int) (player.Player, error) {
	query := sq.
		Select(
			columnPID,
			columnNick,
			columnProvider,
			columnImported,
		).
		From(playerTable).
		Where(sq.And{
			sq.Eq{columnPID: pid},
		}).
		OrderBy(
			fmt.Sprintf("%s ASC", columnProvider),
		)

	rows, err := query.RunWith(r.db).QueryContext(ctx)
	if err != nil {
		return player.Player{}, err
	}

	// Load all results, as we need to ensure we only find exactly one player
	players := make([]player.Player, 0)
	for rows.Next() {
		var p player.Player
		if err = rows.Scan(
			&p.PID,
			&p.Nick,
			&p.Provider,
			&p.Imported,
		); err != nil {
			return player.Player{}, err
		}

		players = append(players, p)
	}

	if err = rows.Err(); err != nil {
		return player.Player{}, err
	}

	if len(players) == 0 {
		return player.Player{}, player.ErrPlayerNotFound
	}
	if len(players) > 1 {
		return player.Player{}, player.ErrMultiplePlayersFound
	}

	return players[0], nil
}
