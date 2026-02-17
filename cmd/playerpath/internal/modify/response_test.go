package modify_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cetteup/playerpath/cmd/playerpath/internal/asp"
	"github.com/cetteup/playerpath/cmd/playerpath/internal/modify"
	"github.com/cetteup/playerpath/internal/domain/provider"
)

func TestVerificationResponseModifier_Modify(t *testing.T) {
	tests := []struct {
		name            string
		provider        provider.Provider
		prepare         func(res *http.Response)
		wantBody        string
		wantErrContains string
	}{
		{
			name:     "modifies verification passed response from BF2Hub",
			provider: provider.BF2Hub,
			prepare:  func(res *http.Response) {},
			wantBody: "O\nH\tpid\tnick\tspid\tasof\nD\t1234567890\twalterwhite\t1234567890\t1771369200\nH\tresult\nD\tOk\n$\t69\t$",
		},
		{
			name:     "modifies verification failed response from BF2Hub",
			provider: provider.BF2Hub,
			prepare: func(res *http.Response) {
				res.Body = io.NopCloser(strings.NewReader("E\t997\n$\t4\t$"))
			},
			wantBody: "O\nH\tpid\tnick\tspid\tasof\nD\t0\tINVALID walterwhite\t1234567890\t1771369200\nH\tresult\nD\tInvalidAuthProfileID\n$\t86\t$",
		},
		{
			name:     "modifies invalid syntax response from BF2Hub",
			provider: provider.BF2Hub,
			prepare: func(res *http.Response) {
				res.Body = io.NopCloser(strings.NewReader("E\t999\n$\t4\t$"))
			},
			wantBody: "E\t107\nH\tasof\terr\nD\t1771369200\tInvalid Syntax!\n$\t38\t$",
		},
		{
			name:     "truncates nick after adding invalid prefix to maintain 23 character limit",
			provider: provider.BF2Hub,
			prepare: func(res *http.Response) {
				res.Body = io.NopCloser(strings.NewReader("E\t997\n$\t4\t$"))
				res.Request.URL.RawQuery = "auth=abcdefghijklmnopqrstuv__&SoldierNick=somelongnamethisis&pid=1234567890"
			},
			wantBody: "O\nH\tpid\tnick\tspid\tasof\nD\t0\tINVALID somelongnamethi\t1234567890\t1771369200\nH\tresult\nD\tInvalidAuthProfileID\n$\t90\t$",
		},
		{
			name:     "handles SoldierNick containing unescaped query parameter syntax characters",
			provider: provider.BF2Hub,
			prepare: func(res *http.Response) {
				res.Request.URL.RawQuery = "auth=abcdefghijklmnopqrstuv__&SoldierNick=it?s%20me&mario?&pid=1234567890"
			},
			wantBody: "O\nH\tpid\tnick\tspid\tasof\nD\t1234567890\tit?s%20me&mario?\t1234567890\t1771369200\nH\tresult\nD\tOk\n$\t74\t$",
		},
		{
			name:     "sets syntax error for query not starting with auth=",
			provider: provider.BF2Hub,
			prepare: func(res *http.Response) {
				res.Request.URL.RawQuery = "SoldierNick=walterwhite&pid=1234567890"
			},
			wantBody: "E\t107\nH\tasof\terr\nD\t1771369200\tInvalid Syntax!\n$\t38\t$",
		},
		{
			name:     "sets syntax error for query without SoldierNick",
			provider: provider.BF2Hub,
			prepare: func(res *http.Response) {
				res.Request.URL.RawQuery = "auth=abcdefghijklmnopqrstuv__&pid=1234567890"
			},
			wantBody: "E\t107\nH\tasof\terr\nD\t1771369200\tInvalid Syntax!\n$\t38\t$",
		},
		{
			name:     "sets syntax error for query without pid",
			provider: provider.BF2Hub,
			prepare: func(res *http.Response) {
				res.Request.URL.RawQuery = "auth=abcdefghijklmnopqrstuv__&SoldierNick=walterwhite"
			},
			wantBody: "E\t107\nH\tasof\terr\nD\t1771369200\tInvalid Syntax!\n$\t38\t$",
		},
		{
			name:     "sets syntax error for query with non-numeric pid",
			provider: provider.BF2Hub,
			prepare: func(res *http.Response) {
				res.Request.URL.RawQuery = "auth=abcdefghijklmnopqrstuv__&SoldierNick=walterwhite&pid=0xff"
			},
			wantBody: "E\t107\nH\tasof\terr\nD\t1771369200\tInvalid Syntax!\n$\t38\t$",
		},
		{
			name:     "skips response from provider supporting standard player verification",
			provider: provider.PlayBF2,
			prepare: func(res *http.Response) {
				res.Body = io.NopCloser(strings.NewReader("O\n$\t1\t$"))
			},
			wantBody: "O\n$\t1\t$",
		},
		{
			name:     "skips response from other endpoint",
			provider: provider.BF2Hub,
			prepare: func(res *http.Response) {
				res.Body = io.NopCloser(strings.NewReader("O\n$\t1\t$"))
				res.Request.URL.Path = "/ASP/getplayerinfo.aspx"
			},
			wantBody: "O\n$\t1\t$",
		},
		{
			name:     "skips response with non-200 HTTP status",
			provider: provider.BF2Hub,
			prepare: func(res *http.Response) {
				res.StatusCode = http.StatusNotFound
				res.Status = http.StatusText(http.StatusNotFound)
				res.Body = io.NopCloser(strings.NewReader("O\n$\t1\t$"))
			},
			wantBody: "O\n$\t1\t$",
		},
		{
			name:     "fails for unknown response code from BF2Hub",
			provider: provider.BF2Hub,
			prepare: func(res *http.Response) {
				res.Body = io.NopCloser(strings.NewReader("E\t420\n$\t4\t$"))
			},
			wantErrContains: "unknown player verification response code",
		},
	}

	// Mock time.Now in asp package
	var now = time.Date(2026, 2, 17, 23, 0, 0, 0, time.UTC)
	asp.Now = func() time.Time { return now }
	t.Cleanup(func() { asp.Now = time.Now })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN
			modifier := modify.VerificationResponseModifier{}
			res := givenResponse(
				"/ASP/VerifyPlayer.aspx",
				"auth=abcdefghijklmnopqrstuv__&SoldierNick=walterwhite&pid=1234567890",
				http.StatusOK,
				"E\t996\n$\t4\t$",
			)
			tt.prepare(res)

			// WHEN
			actual := *res
			err := modifier.Modify(tt.provider, &actual)

			// THEN
			if tt.wantErrContains != "" {
				assert.ErrorContains(t, err, tt.wantErrContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, res.StatusCode, actual.StatusCode)

				body, err2 := io.ReadAll(actual.Body)
				require.NoError(t, err2)
				assert.Equal(t, tt.wantBody, string(body))
			}
		})
	}
}

func givenResponse(path string, query string, status int, body string) *http.Response {
	return &http.Response{
		Status:        http.StatusText(status),
		StatusCode:    status,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request: &http.Request{
			URL: &url.URL{
				Path:     path,
				RawQuery: query,
			},
		},
	}
}
