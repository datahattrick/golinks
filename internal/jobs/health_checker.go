package jobs

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"golinks/internal/db"
	"golinks/internal/models"
	"golinks/internal/validation"
)

// HealthChecker performs background health checks on links.
type HealthChecker struct {
	db       *db.DB
	interval time.Duration
	maxAge   time.Duration
	client   *http.Client
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(database *db.DB, interval, maxAge time.Duration) *HealthChecker {
	return &HealthChecker{
		db:       database,
		interval: interval,
		maxAge:   maxAge,
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
	}
}

// Start begins the background health check loop.
func (h *HealthChecker) Start(ctx context.Context) {
	log.Printf("Health checker started (interval: %v, maxAge: %v)", h.interval, h.maxAge)

	// Run immediately on start
	h.checkAll(ctx)

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Health checker stopped")
			return
		case <-ticker.C:
			h.checkAll(ctx)
		}
	}
}

// checkAll checks all links that need a health check.
func (h *HealthChecker) checkAll(ctx context.Context) {
	links, err := h.db.GetLinksNeedingHealthCheck(ctx, h.maxAge, 50)
	if err != nil {
		log.Printf("Health checker: failed to get links: %v", err)
		return
	}

	if len(links) == 0 {
		return
	}

	log.Printf("Health checker: checking %d links", len(links))

	for _, link := range links {
		// Check context before each link
		select {
		case <-ctx.Done():
			return
		default:
		}

		status, errorMsg := h.checkURL(ctx, link.URL)
		if err := h.db.UpdateLinkHealthStatus(ctx, link.ID, status, errorMsg); err != nil {
			log.Printf("Health checker: failed to update link %s: %v", link.Keyword, err)
			continue
		}

		// Delay between checks to avoid overwhelming external servers
		time.Sleep(1 * time.Second)
	}
}

// checkURL performs a HEAD request to check if a URL is healthy.
// Validates URLs before making requests to prevent SSRF attacks.
func (h *HealthChecker) checkURL(ctx context.Context, url string) (string, *string) {
	// Validate URL is safe to check (prevents SSRF)
	if valid, msg := validation.ValidateURLForHealthCheck(url); !valid {
		return models.HealthUnhealthy, &msg
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		errMsg := "invalid URL: " + err.Error()
		return models.HealthUnhealthy, &errMsg
	}

	req.Header.Set("User-Agent", "GoLinks-HealthChecker/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		errMsg := "connection failed: " + err.Error()
		return models.HealthUnknown, &errMsg
	}
	defer resp.Body.Close()

	// Any HTTP response means the site is reachable
	return models.HealthHealthy, nil
}
