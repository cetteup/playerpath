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
			event := log.Warn().
				Int(trace.LogPlayerPID, params.PID)

			// Defer verification to server/global default provider if possible
			if pv := h.getServerOrDefaultProvider(c.RealIP()); pv.SupportsPlayerVerification() {
				event.
					Stringer(trace.LogProvider, pv).
					Msg("Player not found, deferring verification to default provider")
				return h.handleForward(c, pv)
			}

			// Treating not found as unverified here to ensure you cannot bypass verification simply by using
			// an unknown PID when using a non-verifying default provider such as BF2Hub
			event.Msg("Player not found, treating as unverified")
			return c.String(http.StatusOK, buildResponse(
				addInvalidPrefix(params.Nick),
				params.Nick,
				dummyPID,
				params.PID,
			).Serialize())
		}
		if errors.Is(err, player.ErrMultiplePlayersFound) {
			event := log.Warn().
				Int(trace.LogPlayerPID, params.PID)

			// By leaving the verification up to the default provider (which should be the provider used by the server),
			// the provider can (potentially) resolve the conflict (e.g. based on the `auth` parameter)
			if pv := h.getServerOrDefaultProvider(c.RealIP()); pv.SupportsPlayerVerification() {
				event.
					Stringer(trace.LogProvider, pv).
					Msg("Found multiple players, using default provider to verify player")
				return h.handleForward(c, pv)
			}

			// Again: Treat as unverified to not enable verification bypass by using a conflicting PID
			// This does mean that players with conflicting PIDs will be unable to play on some servers,
			// but there is no clean solution to this conflict from the outside
			event.Msg("Found multiple players, treating as unverified")
			return c.String(http.StatusOK, buildResponse(
				addInvalidPrefix(params.Nick),
				params.Nick,
				dummyPID,
				params.PID,
			).Serialize())
		}
		return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(fmt.Errorf("failed to find player: %w", err))
	}

	// Let the player's provider verify if possible
	if p.Provider.SupportsPlayerVerification() {
		return h.handleForward(c, p.Provider)
	}

	// All we can do here is validate that an account with the given pid and name exists
	return c.String(http.StatusOK, buildResponse(p.Nick, params.Nick, p.PID, params.PID).Serialize())
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
