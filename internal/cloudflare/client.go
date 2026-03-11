package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const defaultCloudflareBaseURL = "https://rtc.live.cloudflare.com"

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

// FetcherConfig describes the dependencies required to call the Cloudflare TURN
// credential endpoint.
type FetcherConfig struct {
	HTTPClient HTTPClient
	BaseURL    string
	KeyID      string
	APIToken   string
	TTL        int
}

// Fetcher requests Cloudflare TURN credentials and normalizes them into []ICEServer.
type Fetcher struct {
	client   HTTPClient
	endpoint string
	token    string
	ttl      int
}

// NewFetcher builds a Fetcher with the provided configuration.
func NewFetcher(cfg FetcherConfig) (*Fetcher, error) {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	keyID := strings.TrimSpace(cfg.KeyID)
	if keyID == "" {
		return nil, fmt.Errorf("cloudflare: key id is required")
	}
	token := strings.TrimSpace(cfg.APIToken)
	if token == "" {
		return nil, fmt.Errorf("cloudflare: api token is required")
	}
	if cfg.TTL <= 0 {
		return nil, fmt.Errorf("cloudflare: ttl must be positive")
	}
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = defaultCloudflareBaseURL
	}
	endpoint := buildEndpoint(base, keyID)
	return &Fetcher{
		client:   cfg.HTTPClient,
		endpoint: endpoint,
		token:    token,
		ttl:      cfg.TTL,
	}, nil
}

func buildEndpoint(base, keyID string) string {
	trimmed := strings.TrimRight(base, "/")
	return fmt.Sprintf("%s/v1/turn/keys/%s/credentials/generate-ice-servers", trimmed, keyID)
}

// Fetch requests TURN credentials from Cloudflare.
func (f *Fetcher) Fetch(ctx context.Context) ([]ICEServer, error) {
	slog.Debug("cloudflare fetch starting", "component", "cloudflare", "ttl", f.ttl, "endpoint", f.endpoint)
	payload := struct {
		TTL int `json:"ttl"`
	}{TTL: f.ttl}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cloudflare: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: do request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		slog.Error("cloudflare fetch failed", "component", "cloudflare", "status", resp.StatusCode)
		return nil, fmt.Errorf("cloudflare: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	servers, err := ParseICEServers(respBody)
	if err != nil {
		slog.Error("cloudflare parse failed", "component", "cloudflare", "err", err)
		return nil, fmt.Errorf("cloudflare: parse response: %w", err)
	}
	slog.Info("cloudflare fetch completed", "component", "cloudflare", "server_count", len(servers), "status", resp.StatusCode)
	return servers, nil
}
