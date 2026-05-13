// Package oidchealth provides a cached reachability probe for the configured
// OIDC issuer. The redirect flow consults it so that unauthenticated users
// can still resolve global keywords when the auth server is unreachable.
package oidchealth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const (
	probeInterval = 30 * time.Second
	probeTimeout  = 2 * time.Second
)

// Probe holds the cached reachability state for an OIDC issuer.
type Probe struct {
	url      string
	client   *http.Client
	healthy  atomic.Bool
	probedAt atomic.Bool // false until the first probe completes
}

// New constructs a Probe targeting the issuer's discovery document.
// The initial state is healthy: the server only reaches this point after a
// successful provider discovery at startup.
func New(issuer string) *Probe {
	p := &Probe{
		url:    strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration",
		client: &http.Client{Timeout: probeTimeout},
	}
	p.healthy.Store(true)
	return p
}

// IsHealthy reports whether the last probe succeeded.
func (p *Probe) IsHealthy() bool {
	return p.healthy.Load()
}

// Start runs probes on a ticker until the context is cancelled.
// Intended to be invoked once as `go probe.Start(ctx)` at server startup.
func (p *Probe) Start(ctx context.Context) {
	p.tick(ctx)
	ticker := time.NewTicker(probeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.tick(ctx)
		}
	}
}

func (p *Probe) tick(ctx context.Context) {
	reqCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, p.url, nil)
	if err != nil {
		p.markUnhealthy(err.Error())
		return
	}
	resp, err := p.client.Do(req)
	if err != nil {
		p.markUnhealthy(err.Error())
		return
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		p.markUnhealthy("status " + resp.Status)
		return
	}
	p.markHealthy()
}

func (p *Probe) markUnhealthy(reason string) {
	wasHealthy := p.healthy.Swap(false)
	firstProbe := !p.probedAt.Swap(true)
	if wasHealthy || firstProbe {
		slog.Warn("oidc issuer unreachable", "url", p.url, "reason", reason)
	}
}

func (p *Probe) markHealthy() {
	wasUnhealthy := !p.healthy.Swap(true)
	firstProbe := !p.probedAt.Swap(true)
	switch {
	case firstProbe:
		slog.Info("oidc issuer reachable", "url", p.url)
	case wasUnhealthy:
		slog.Info("oidc issuer reachable again", "url", p.url)
	}
}
