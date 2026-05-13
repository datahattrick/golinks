package oidchealth

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestProbeTransitions(t *testing.T) {
	// Capture slog output so we can see the warn/info transitions.
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	// Toggleable test server: when "down" flips true, it 500s; otherwise 200.
	var down atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if down.Load() {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	p := &Probe{
		url:    srv.URL + "/.well-known/openid-configuration",
		client: srv.Client(),
	}
	p.healthy.Store(true)

	ctx := context.Background()

	// 1. Healthy → healthy: no transition log.
	p.tick(ctx)
	if !p.IsHealthy() {
		t.Fatalf("expected probe to remain healthy on 200; got unhealthy")
	}

	// 2. Server starts 500ing → expect Warn transition.
	down.Store(true)
	p.tick(ctx)
	if p.IsHealthy() {
		t.Fatalf("expected probe to go unhealthy after 500; still healthy")
	}

	// 3. Server back to 200 → expect Info transition.
	down.Store(false)
	p.tick(ctx)
	if !p.IsHealthy() {
		t.Fatalf("expected probe to recover; still unhealthy")
	}

	logs := buf.String()
	t.Logf("captured slog output:\n%s", logs)

	if !strings.Contains(logs, "oidc issuer reachable\"") {
		t.Errorf("expected first-probe reachable info in logs, got:\n%s", logs)
	}
	if !strings.Contains(logs, "oidc issuer unreachable") {
		t.Errorf("expected unreachable warning in logs, got:\n%s", logs)
	}
	if !strings.Contains(logs, "oidc issuer reachable again") {
		t.Errorf("expected recovery info in logs, got:\n%s", logs)
	}
}

func TestProbeNetworkFailure(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	// Point at a server that's already shut down → instant connection refused.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close()

	p := New(url)
	p.tick(context.Background())

	if p.IsHealthy() {
		t.Fatalf("expected probe to mark unhealthy when issuer is unreachable")
	}

	t.Logf("captured slog output:\n%s", buf.String())
	if !strings.Contains(buf.String(), "oidc issuer unreachable") {
		t.Errorf("expected unreachable warning in logs, got:\n%s", buf.String())
	}
}
