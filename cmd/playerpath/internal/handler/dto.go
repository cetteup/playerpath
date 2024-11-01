package handler

import (
	"fmt"

	"github.com/cetteup/playerpath/internal"
	"github.com/cetteup/playerpath/internal/domain/player"
	"github.com/cetteup/playerpath/internal/domain/provider"
)

type PlayerDTO struct {
	PID        int     `json:"pid"`
	Nick       string  `json:"nick"`
	Provider   string  `json:"provider"`
	ProfileURL *string `json:"profileUrl"`
}

func EncodePlayer(p player.Player) PlayerDTO {
	return PlayerDTO{
		PID:        p.PID,
		Nick:       p.Nick,
		Provider:   p.Provider.String(),
		ProfileURL: buildProfileURL(p),
	}
}

func buildProfileURL(p player.Player) *string {
	switch p.Provider {
	case provider.ProviderBF2Hub:
		return internal.ToPointer(fmt.Sprintf("https://www.bf2hub.com/stats/%d", p.PID))
	case provider.ProviderPlayBF2:
		return internal.ToPointer(fmt.Sprintf("http://bf2.tgamer.ru/stats/?pid=%d", p.PID))
	case provider.ProviderB2BF2:
		return internal.ToPointer(fmt.Sprintf("https://b2bf2.net/bfhq?id=%d", p.PID))
	default:
		return nil
	}
}
