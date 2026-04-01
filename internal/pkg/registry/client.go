package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strconv"
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

type FilterFunc func(q url.Values)

func WithProviderFilter(provider string) FilterFunc {
	return func(q url.Values) {
		q.Set("provider", provider)
	}
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

func (c *Client) GetPlayers(ctx context.Context, filters ...FilterFunc) (PageIterator, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	u = u.JoinPath("players")

	q := u.Query()
	for _, filter := range filters {
		filter(q)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	return newPageIterator(c, req), nil
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", "playerpath")

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

type PageIterator interface {
	After(id string) iter.Seq[Player]
	Err() error
}

type pageIterator struct {
	client *Client
	req    *http.Request
	err    error
}

func newPageIterator(client *Client, req *http.Request) *pageIterator {
	return &pageIterator{
		client: client,
		req:    req,
	}
}

func (i *pageIterator) After(id string) iter.Seq[Player] {
	return func(yield func(p Player) bool) {
		q := i.req.URL.Query()
		q.Set("perPage", strconv.Itoa(pageSize))

		hasMore := true
		after := id
		for hasMore {
			if after != "" {
				q.Set("after", after)
				i.req.URL.RawQuery = q.Encode()
			}

			body, err := i.client.do(i.req)
			if err != nil {
				i.err = err
				return
			}

			resp := struct {
				Players []Player `json:"players"`
				HasMore bool     `json:"hasMore"`
			}{}
			if err = json.Unmarshal(body, &resp); err != nil {
				i.err = err
				return
			}

			for _, p := range resp.Players {
				if err = i.req.Context().Err(); err != nil {
					i.err = err
					return
				}

				if !yield(p) {
					return
				}
			}

			hasMore = resp.HasMore
			if size := len(resp.Players); size > 0 {
				after = resp.Players[size-1].ID
			}
		}
	}
}

func (i *pageIterator) Err() error {
	return i.err
}

func isSuccessStatusCode(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode <= http.StatusIMUsed
}
