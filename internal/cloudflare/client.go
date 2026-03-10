package cloudflare

import "net/http"

// HTTPClient allows the cloudflare package to accept any HTTP client that
// can execute a request and return a response. This keeps the API flexible
// for future orchestration work.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client wraps an HTTP client so callers can inject mocks or configure a
// custom transport without reaching into the standard library directly.
type Client struct {
	httpClient HTTPClient
}

// NewClient constructs a cloudflare client. If no HTTP client is passed in,
// it defaults to http.DefaultClient.
func NewClient(client HTTPClient) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{httpClient: client}
}

// Do executes the request using the configured HTTP client.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}
