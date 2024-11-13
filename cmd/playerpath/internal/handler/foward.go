package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/cetteup/playerpath/cmd/playerpath/internal/asp"
	"github.com/cetteup/playerpath/internal/domain/player"
	"github.com/cetteup/playerpath/internal/domain/provider"
	"github.com/cetteup/playerpath/internal/trace"
)

type UpstreamResponse struct {
	StatusCode int
	Header     map[string][]string
	Body       []byte
}

func (h *Handler) HandleDynamicForward(c echo.Context) error {
	p := struct {
		PID int `query:"pid"`
	}{}
	if err := c.Bind(&p); err != nil {
		return c.String(http.StatusOK, asp.NewSyntaxErrorResponse().Serialize())
	}

	pv, err := h.determineProvider(c.Request().Context(), p.PID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err)
	}

	return h.handleForward(c, pv)
}

func (h *Handler) HandleStaticForward(c echo.Context) error {
	return h.handleForward(c, h.provider)
}

func (h *Handler) handleForward(c echo.Context, pv provider.Provider) error {
	u, err := url.Parse(pv.BaseURL())
	if err != nil {
		return err
	}
	u = u.JoinPath(c.Request().URL.Path)
	u.RawQuery = c.Request().URL.RawQuery

	req, err := http.NewRequestWithContext(c.Request().Context(), c.Request().Method, u.String(), c.Request().Body)
	if err != nil {
		return err
	}

	// Use GameSpy host value only if provider requires it
	if pv.RequiresGameSpyHost() {
		switch u.Path {
		// Yes, BF2Hub really *requires* different host headers for these endpoints
		case "/ASP/getrankstatus.aspx":
			req.Host = "battlefield2.gamestats.gamespy.com"
		case "/ASP/sendsnapshot.aspx":
			req.Host = "gamestats.gamespy.com"
		default:
			req.Host = "BF2Web.gamespy.com"
		}
	}

	// Copy downstream user agent to ensure compatibility
	req.Header.Set("User-Agent", c.Request().Header.Get("User-Agent"))
	req.Header.Set("X-Forwarded-Proto", c.Request().Proto)
	req.Header.Set("X-Forwarded-For", c.RealIP())
	req.Header.Set("X-Real-IP", c.RealIP())

	log.Debug().
		Stringer(trace.LogProvider, pv).
		Str("URI", c.Request().RequestURI).
		Msg("Forwarding request")

	res, err := h.client.Do(req)
	if err != nil {
		return err
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = res.Body.Close()
	if err != nil {
		return err
	}

	return c.String(res.StatusCode, string(bytes))
}

func (h *Handler) determineProvider(ctx context.Context, pid int) (provider.Provider, error) {
	p, err := h.repository.FindByPID(ctx, pid)
	if err != nil {
		if errors.Is(err, player.ErrPlayerNotFound) {
			log.Warn().
				Int(trace.LogPlayerPID, pid).
				Msg("Player not found, falling back to default provider")
			return h.provider, nil
		}
		if errors.Is(err, player.ErrMultiplePlayersFound) {
			log.Warn().
				Int(trace.LogPlayerPID, pid).
				Msg("Found multiple players, falling back to default provider")
			return h.provider, nil
		}
		return 0, err
	}

	return p.Provider, nil
}
