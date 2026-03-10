package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseGenerateICEServersResponse_ReturnsCredentialedServers(t *testing.T) {
	body := []byte(`{"iceServers":[{"urls":["stun:stun.cloudflare.com:3478"]},{"urls":["turn:turn.cloudflare.com:3478?transport=udp"],"username":"u","credential":"p"}]}`)

	servers, err := ParseICEServers(body)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
}

func TestParseGenerateICEServersResponse_HandlesSingleStringURL(t *testing.T) {
	body := []byte(`{"iceServers":[{"urls":"stun:stun.cloudflare.com:3478"}]}`)

	servers, err := ParseICEServers(body)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := len(servers); got != 1 {
		t.Fatalf("expected 1 server, got %d", got)
	}

	if len(servers[0].URLs) != 1 {
		t.Fatalf("expected 1 url, got %d", len(servers[0].URLs))
	}
}

func TestParseGenerateICEServersResponse_InvalidURLsTypeReturnsError(t *testing.T) {
	body := []byte(`{"iceServers":[{"urls":123}]}`)

	if _, err := ParseICEServers(body); err == nil {
		t.Fatal("expected error when urls is not string or array")
	}
}

func TestParseGenerateICEServersResponse_SkipsEntriesWithNoURLs(t *testing.T) {
	body := []byte(`{"iceServers":[{"urls":[]},{"urls":"stun:stun.cloudflare.com:3478"}]}`)

	servers, err := ParseICEServers(body)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
}

func TestParseGenerateICEServersResponse_HandlesNestedResultEnvelope(t *testing.T) {
	body := []byte(`{"result":{"iceServers":[{"urls":"turn:turn.cloudflare.com:3478?transport=udp","username":"u","credential":"p"}]}}`)

	servers, err := ParseICEServers(body)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
}

type testHTTPClient struct {
	req  *http.Request
	resp *http.Response
	err  error
}

func (t *testHTTPClient) Do(req *http.Request) (*http.Response, error) {
	t.req = req
	return t.resp, t.err
}

type transportStub struct {
	req  *http.Request
	resp *http.Response
	err  error
}

func (t *transportStub) RoundTrip(req *http.Request) (*http.Response, error) {
	t.req = req
	return t.resp, t.err
}

func TestClientDo_UsesInjectedClient(t *testing.T) {
	stub := &testHTTPClient{resp: &http.Response{}}
	client := NewClient(stub)
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)

	if _, err := client.Do(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stub.req != req {
		t.Fatal("expected injected client to be used")
	}
}

func TestClientDo_DefaultsToHTTPDefaultClient(t *testing.T) {
	backup := http.DefaultClient
	stub := &transportStub{resp: &http.Response{}}
	http.DefaultClient = &http.Client{Transport: stub}
	defer func() { http.DefaultClient = backup }()

	client := NewClient(nil)
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)

	if _, err := client.Do(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stub.req != req {
		t.Fatal("expected default client to flow through")
	}
}

func TestFetch_BuildsCloudflareRequestWithTTL(t *testing.T) {
	keyID := "test-key"
	token := "token"
	ttl := 300
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		if r.URL.Path != "/v1/turn/keys/"+keyID+"/credentials/generate-ice-servers" {
			t.Fatalf("unexpected path, got %s", r.URL.Path)
		}
		var payload struct {
			TTL int `json:"ttl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.TTL != ttl {
			t.Fatalf("expected ttl %d, got %d", ttl, payload.TTL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"iceServers":[{"urls":["turn:turn.example.com"],"username":"u","credential":"p"}]}`))
	}))
	defer srv.Close()

	fetcher, err := NewFetcher(FetcherConfig{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		KeyID:      keyID,
		APIToken:   token,
		TTL:        ttl,
	})
	if err != nil {
		t.Fatalf("new fetcher: %v", err)
	}

	servers, err := fetcher.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected fetch error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if got := servers[0].URLs[0]; got != "turn:turn.example.com" {
		t.Fatalf("unexpected url: %s", got)
	}
}

func TestFetch_ReturnsErrorOnNonOKResponse(t *testing.T) {
	keyID := "test-key"
	token := "token"
	ttl := 60
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	}))
	defer srv.Close()

	fetcher, err := NewFetcher(FetcherConfig{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		KeyID:      keyID,
		APIToken:   token,
		TTL:        ttl,
	})
	if err != nil {
		t.Fatalf("new fetcher: %v", err)
	}

	if _, err := fetcher.Fetch(context.Background()); err == nil {
		t.Fatal("expected error on non-200 response")
	}
}

func TestFetch_ReturnsErrorOnInvalidJSON(t *testing.T) {
	keyID := "test-key"
	token := "token"
	ttl := 60
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid`))
	}))
	defer srv.Close()

	fetcher, err := NewFetcher(FetcherConfig{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		KeyID:      keyID,
		APIToken:   token,
		TTL:        ttl,
	})
	if err != nil {
		t.Fatalf("new fetcher: %v", err)
	}

	if _, err := fetcher.Fetch(context.Background()); err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

type errorHTTPClient struct {
	err error
}

func (h *errorHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return nil, h.err
}

func TestFetch_ReturnsErrorOnTransportFailure(t *testing.T) {
	keyID := "test-key"
	token := "token"
	ttl := 60
	fetcher, err := NewFetcher(FetcherConfig{
		HTTPClient: &errorHTTPClient{err: fmt.Errorf("boom")},
		BaseURL:    "https://rtc.live.cloudflare.com",
		KeyID:      keyID,
		APIToken:   token,
		TTL:        ttl,
	})
	if err != nil {
		t.Fatalf("new fetcher: %v", err)
	}

	if _, err := fetcher.Fetch(context.Background()); err == nil {
		t.Fatal("expected error on transport failure")
	}
}

func TestFetch_TrimsKeyAndToken(t *testing.T) {
	keyID := " test-key "
	token := " token "
	ttl := 60
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/turn/keys/test-key/credentials/generate-ice-servers" {
			t.Fatalf("unexpected path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("unexpected auth header, got %s", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"iceServers":[{"urls":["turn:turn.example.com"],"username":"u","credential":"p"}]}`))
	}))
	defer srv.Close()

	fetcher, err := NewFetcher(FetcherConfig{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		KeyID:      keyID,
		APIToken:   token,
		TTL:        ttl,
	})
	if err != nil {
		t.Fatalf("new fetcher: %v", err)
	}

	if _, err := fetcher.Fetch(context.Background()); err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
}
