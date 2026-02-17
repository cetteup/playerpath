package modify

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/cetteup/playerpath/cmd/playerpath/internal/asp"
	"github.com/cetteup/playerpath/internal/domain/provider"
)

const (
	dummyPID = 0
)

type VerificationResponseModifier struct{}

func (m VerificationResponseModifier) Type() ModifierType {
	return ModifierTypeResponse
}

func (m VerificationResponseModifier) Modify(pv provider.Provider, res *http.Response) error {
	if m.skip(pv, res) {
		return nil
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	q := parseVerifyPlayersQuery(res.Request.URL.RawQuery)
	pid := q.Get("pid")
	nick := q.Get("SoldierNick")

	if pid == "" || nick == "" {
		res.Body = io.NopCloser(strings.NewReader(asp.NewSyntaxErrorResponse().Serialize()))
		return nil
	}

	if pv == provider.BF2Hub {
		resp, err2 := transformBF2HubPlayerVerificationResult(pid, nick, string(body))
		if err2 != nil {
			return err2
		}

		res.Body = io.NopCloser(strings.NewReader(resp.Serialize()))
	}

	return nil
}

func (m VerificationResponseModifier) skip(pv provider.Provider, res *http.Response) bool {
	// Skip modify responses from provider which support standard player verification
	// Results from such providers need to be passed through as is
	if provider.SupportsStandardPlayerVerification(pv) {
		return true
	}

	// Skip unrelated responses
	if res.Request.URL.Path != "/ASP/VerifyPlayer.aspx" {
		return true
	}

	// Skip non-ok response (ASP should *always* respond 200/OK, even for errors)
	if res.StatusCode != http.StatusOK {
		return true
	}

	return false
}

// parseVerifyPlayersQuery Query string parsing specifically for expected parameters of VerifyPlayers.aspx
func parseVerifyPlayersQuery(query string) url.Values {
	q := make(url.Values)

	// Battlefield 2 does not query encode any characters whatsoever, not even ones that can be mistaken for syntax
	// To still be able to parse the query parameters, we look for the specific parameters for this request (in order)

	query, found := strings.CutPrefix(query, "auth=")
	if !found {
		return nil
	}

	auth, query, found := strings.Cut(query, "&SoldierNick=")
	if !found {
		return nil
	}

	nick, pid, found := strings.Cut(query, "&pid=")
	if !found {
		return nil
	}

	// pid may only contain decimal digits ([0-9])
	if strings.ContainsFunc(pid, func(r rune) bool {
		return r < '0' || r > '9'
	}) {
		return nil
	}

	q.Set("auth", auth)
	q.Set("SoldierNick", nick)
	q.Set("pid", pid)

	return q
}

func transformBF2HubPlayerVerificationResult(pid, nick, result string) (*asp.Response, error) {
	var valid bool
	switch result {
	case "E\t996\n$\t4\t$":
		valid = true
	case "E\t997\n$\t4\t$":
		valid = false
	case "E\t999\n$\t4\t$":
		return asp.NewSyntaxErrorResponse(), nil
	default:
		return nil, fmt.Errorf("unknown player verification response code: %s", result)
	}

	resp := asp.NewOKResponse().
		WriteHeader("pid", "nick", "spid", "asof")

	if valid {
		resp.
			WriteData(pid, nick, pid, asp.Timestamp()).
			WriteHeader("result").
			WriteData("Ok")
	} else {
		// We cannot know whether the auth was invalid or the pid/nick did not match
		// The result is simply used to indicate "there is no session with this combination of pid, nick and auth"
		resp.
			WriteData(strconv.Itoa(dummyPID), addInvalidPrefix(nick), pid, asp.Timestamp()).
			WriteHeader("result").
			WriteData("InvalidAuthProfileID")
	}

	return resp, nil
}

func addInvalidPrefix(nick string) string {
	// `[prefix] nick` usually get cut off after 23 characters in the game's client-server protocols
	// While the limit appears to not be applied to values returned by the validation,
	// it's probably best to follow that convention/limit
	prefixed := "INVALID " + nick
	return prefixed[:min(len(prefixed), 23)]
}
