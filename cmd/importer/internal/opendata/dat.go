package opendata

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type Loader struct {
	basePath string
}

func NewLoader(basePath string) *Loader {
	return &Loader{
		basePath: basePath,
	}
}

func (l *Loader) GetPlayers(ctx context.Context, provider string, cb func(ctx context.Context, pid int, nick string) error) error {
	file, err := os.Open(filepath.Join(l.basePath, fmt.Sprintf("v_%s.dat", provider)))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if err2 := ctx.Err(); err2 != nil {
			return err2
		}

		var player Player
		err2 := player.UnmarshalText(scanner.Bytes())
		if err2 != nil {
			return err2
		}

		if err2 = cb(ctx, player.PID, player.Nick); err2 != nil {
			return err2
		}
	}

	return nil
}
