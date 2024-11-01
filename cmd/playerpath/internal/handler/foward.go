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
	if err2 := c.Bind(&p); err2 != nil {
		return c.String(http.StatusOK, asp.NewSyntaxErrorResponse().Serialize())
	}

	pv, err2 := h.determineProvider(c.Request().Context(), p.PID)
	if err2 != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err2)
	}

	log.Debug().
		Stringer(trace.LogProvider, pv).
		Str("URI", c.Request().RequestURI).
		Msg("Forwarding request")

	res, err2 := h.forwardRequest(c.Request().Context(), pv, c.Request(), c.RealIP())
	if err2 != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err2)
	}

	// Copy all upstream header to ensure response can be handled correctly downstream
	for key, values := range res.Header {
		for _, value := range values {
			c.Response().Header().Add(key, value)
		}
	}

	return c.String(res.StatusCode, string(res.Body))
}

func (h *Handler) HandleStaticForward(c echo.Context) error {
	res, err2 := h.forwardRequest(c.Request().Context(), h.provider, c.Request(), c.RealIP())
	if err2 != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err2)
	}

	// Copy all upstream header to ensure response can be handled correctly downstream
	for key, values := range res.Header {
		for _, value := range values {
			c.Response().Header().Add(key, value)
		}
	}

	return c.String(res.StatusCode, string(res.Body))
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

func (h *Handler) forwardRequest(ctx context.Context, pv provider.Provider, incoming *http.Request, realIP string) (*UpstreamResponse, error) {
	u, err := url.Parse(pv.BaseURL())
	if err != nil {
		return nil, err
	}
	u = u.JoinPath(incoming.URL.Path)
	u.RawQuery = incoming.URL.RawQuery

	req, err := http.NewRequestWithContext(ctx, incoming.Method, u.String(), incoming.Body)
	if err != nil {
		return nil, err
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
	req.Header.Set("User-Agent", incoming.Header.Get("User-Agent"))
	req.Header.Set("X-Forwarded-Proto", incoming.Proto)
	req.Header.Set("X-Forwarded-For", realIP)
	req.Header.Set("X-Real-IP", realIP)

	res, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return &UpstreamResponse{
		StatusCode: res.StatusCode,
		Header:     res.Header,
		Body:       bytes,
	}, nil
}
