package player

import (
	"time"

	"github.com/cetteup/playerpath/internal/domain/provider"
)

type Player struct {
	PID      int
	Nick     string
	Provider provider.Provider
	Imported time.Time
}
