package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	BaseURL = "https://api.registry.bf2.co/v1/"

	pageSize = 1000
)

type RequestError struct {
	requestURL *url.URL
	statusCode int
}

func newRequestError(requestURL *url.URL, statusCode int) *RequestError {
	return &RequestError{
		requestURL: requestURL,
		statusCode: statusCode,
	}
}

func (e RequestError) Error() string {
	return fmt.Sprintf("request to %s failed with status code %d", e.requestURL.String(), e.statusCode)
}

func (e RequestError) StatusCode() int {
	return e.statusCode
}

type Client struct {
	baseURL string

	client *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) GetPlayers(ctx context.Context, provider string, cb func(ctx context.Context, pid int, nick string) error) error {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}

	u = u.JoinPath("players")

	q := u.Query()
	q.Set("provider", strings.ToLower(provider))
	q.Set("perPage", strconv.Itoa(pageSize))
	u.RawQuery = q.Encode()

	hasMore := true
	after := ""
	for hasMore {
		if after != "" {
			q.Set("after", after)
			u.RawQuery = q.Encode()
		}

		req, err2 := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err2 != nil {
			return err2
		}

		body, err2 := c.do(req)
		if err2 != nil {
			return err2
		}

		resp := PlayersResponse{}
		if err2 = json.Unmarshal(body, &resp); err2 != nil {
			return err2
		}

		for _, player := range resp.Players {
			if err2 = ctx.Err(); err2 != nil {
				return err2
			}

			if err2 = cb(ctx, player.PID, player.Nick); err2 != nil {
				return err2
			}
		}

		hasMore = resp.HasMore
		if size := len(resp.Players); size > 0 {
			after = resp.Players[size-1].ID
		}
	}

	return nil
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", "blueberry")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()

	if !isSuccessStatusCode(res.StatusCode) {
		return nil, newRequestError(req.URL, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func isSuccessStatusCode(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode <= http.StatusIMUsed
}
