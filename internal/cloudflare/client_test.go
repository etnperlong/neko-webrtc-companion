package cloudflare

import (
	"net/http"
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
