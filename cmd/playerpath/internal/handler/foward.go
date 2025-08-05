package handler

import (
	"io"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/cetteup/playerpath/cmd/playerpath/internal/asp"
	"github.com/cetteup/playerpath/internal/domain/provider"
	"github.com/cetteup/playerpath/internal/trace"
)

type UpstreamResponse struct {
	StatusCode int
	Header     map[string][]string
	Body       []byte
}

// HandleDynamicForward Handle requests that are forwarded on a per-player basis.
// Dynamic meaning that two requests from a single server for two different players may be forwarded to different providers.
func (h *Handler) HandleDynamicForward(c echo.Context) error {
	p := struct {
		PID int `query:"pid"`
	}{}
	if err := c.Bind(&p); err != nil {
		return c.String(http.StatusOK, asp.NewSyntaxErrorResponse().Serialize())
	}

	pv, err := h.determineProvider(c.Request().Context(), p.PID, c.RealIP())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err)
	}

	return h.handleForward(c, pv)
}

// HandleStaticForward Handle requests that are forwarded on a per-server basis.
// Static only in the sense that any request from a given server will be forwarded to the same provider.
func (h *Handler) HandleStaticForward(c echo.Context) error {
	return h.handleForward(c, h.getServerOrDefaultProvider(c.RealIP()))
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

	// Make any required provider-specific modifications to the outgoing request
	if pv.RequiresGameSpyHost() {
		HostModifier(req)
	}
	if pv.RequiresBFHQInfoQuery() {
		QueryModifier(req)
	}

	// Copy any relevant downstream headers
	for key, values := range c.Request().Header {
		if shouldCopyHeader(key) {
			for i, value := range values {
				if i == 0 {
					// Set rather than add first value to ensure we overwrite any default values
					req.Header.Set(key, value)
				} else {
					req.Header.Add(key, value)
				}
			}
		}
	}

	// Add proxy headers
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

func shouldCopyHeader(key string) bool {
	// Keys *must* use canonical header format
	switch key {
	case "User-Agent":
		// Copy downstream user agent to ensure compatibility
		return true
	case "X-Bf2hub-Tsdata":
		// Copy BF2Hub snapshot header (snapshots sent without are flagged and not processed)
		return true
	default:
		return false
	}
}
