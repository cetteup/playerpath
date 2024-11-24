package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/cetteup/playerpath/cmd/playerpath/internal/asp"
	"github.com/cetteup/playerpath/internal/domain/player"
	"github.com/cetteup/playerpath/internal/domain/provider"
	"github.com/cetteup/playerpath/internal/trace"
)

const (
	dummyPID = 0
)

func (h *Handler) HandleGetVerifyPlayer(c echo.Context) error {
	params := struct {
		PID  int    `query:"pid"`
		Nick string `query:"SoldierNick"`
	}{}
	if err := c.Bind(&params); err != nil {
		return c.String(http.StatusOK, asp.NewSyntaxErrorResponse().Serialize())
	}

	p, err := h.repository.FindByPID(c.Request().Context(), params.PID)
	if err != nil {
		if errors.Is(err, player.ErrPlayerNotFound) {
			log.Warn().
				Int(trace.LogPlayerPID, params.PID).
				Msg("Player not found, treating as unverified")
			// Treating not found as unverified here to ensure you cannot bypass verification simply by using
			// an unknown PID when using a non-verifying default provider such as BF2Hub
			return c.String(http.StatusOK, buildResponse(
				addInvalidPrefix(params.Nick),
				params.Nick,
				dummyPID,
				params.PID,
			).Serialize())
		}
		if errors.Is(err, player.ErrMultiplePlayersFound) {
			log.Warn().
				Int(trace.LogPlayerPID, params.PID).
				Msg("Found multiple players, using default provider to verify player")
			// Using the default provider here isn't great either, but there is no clean solution to this conflict
			// If we treat the conflict as a verification failure, neither of the conflicting PID players will pass
			// By leaving the verification up to the default provider (which should be the provider used by the server),
			// the provider can (potentially) resolve the conflict based on the `auth` parameter
			return h.handleForward(c, h.getServerOrDefaultProvider(c.RealIP()))
		}
		return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(fmt.Errorf("failed to find player: %w", err))
	}

	switch p.Provider {
	case provider.ProviderBF2Hub, provider.ProviderB2BF2:
		// Neither BF2Hub nor B2BF2 currently offer a (compatible) VerifyPlayer.aspx endpoint
		// All we can do is validate that an account with the given pid and name exists
		return c.String(http.StatusOK, buildResponse(p.Nick, params.Nick, p.PID, params.PID).Serialize())
	default:
		return h.handleForward(c, p.Provider)
	}
}

// buildResponse Signature analog to default onPlayerNameValidated handler
func buildResponse(realNick, oldNick string, realPID, oldPID int) *asp.Response {
	resp := asp.NewOKResponse().
		WriteHeader("pid", "nick", "spid", "asof").
		WriteData(strconv.Itoa(realPID), realNick, strconv.Itoa(oldPID), asp.Timestamp()).
		WriteHeader("result")

	if realNick == oldNick && realPID == oldPID {
		resp.WriteData("Ok")
	} else if realNick != oldNick && realPID != oldPID {
		// We obviously cannot validate the auth param, but neither value matching would indicate
		// that the player was not found and this is the closest to "completely invalid" there is
		// (no player can be logged into a profile that does not exist)
		resp.WriteData("InvalidAuthProfileID")
	} else if realNick != oldNick {
		resp.WriteData("InvalidReportedNick")
	} else {
		// Currently unused as realNick differs from oldNick for any non-ok response
		// Primarily here for completeness-sake
		resp.WriteData("InvalidReportedProfileID")
	}

	return resp
}

func addInvalidPrefix(nick string) string {
	// `[prefix] nick` usually get cut off after 23 characters in the game's client-server protocols
	// While the limit appears to not be applied to values returned by the validation,
	// it's probably best to follow that convention/limit
	prefixed := "INVALID " + nick
	return prefixed[:min(len(prefixed), 23)]
}
