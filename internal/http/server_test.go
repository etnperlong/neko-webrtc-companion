package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/etnperlong/neko-webrtc-companion/internal/refresh"
)

func TestHealthz_ReturnsOK(t *testing.T) {
	deps := Dependencies{
		Ready:   func() bool { return true },
		Trigger: func(context.Context) refresh.Result { return refresh.Result{} },
	}
	srv := New(deps)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestHealthzRejectsNonGET(t *testing.T) {
	srv := New(Dependencies{})
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status %d", res.Code)
	}
	if got := res.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("expected Allow GET, got %s", got)
	}
}

func TestReadyz_ReturnsOKWhenReady(t *testing.T) {
	deps := Dependencies{Ready: func() bool { return true }}
	srv := New(deps)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestReadyz_ReturnsServiceUnavailableWhenNotReady(t *testing.T) {
	deps := Dependencies{Ready: func() bool { return false }}
	srv := New(deps)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", res.Code)
	}
}

func TestReadyzRejectsNonGET(t *testing.T) {
	srv := New(Dependencies{Ready: func() bool { return true }})
	req := httptest.NewRequest(http.MethodPost, "/readyz", nil)
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status %d", res.Code)
	}
	if got := res.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("expected Allow GET, got %s", got)
	}
}

func TestTriggerRejectsNonPOST(t *testing.T) {
	srv := New(Dependencies{Trigger: func(context.Context) refresh.Result { return refresh.Result{} }})
	req := httptest.NewRequest(http.MethodGet, "/trigger", nil)
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status %d", res.Code)
	}
	if got := res.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("expected Allow POST, got %s", got)
	}
}

func TestTrigger_ReturnsConflictWhenRefreshAlreadyRunning(t *testing.T) {
	srv := New(Dependencies{Trigger: func(context.Context) refresh.Result {
		return refresh.Result{Busy: true, Skipped: true}
	}})
	req := httptest.NewRequest(http.MethodPost, "/trigger", nil)
	res := httptest.NewRecorder()
	srv.ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.Code)
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "busy" {
		t.Fatalf("expected busy status, got %q", body.Status)
	}
}
