package server

import (
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/encryptcookie"
	"github.com/gofiber/fiber/v3/middleware/session"
)

// TestEncryptCookieSessionRoundTrip verifies that the encryptcookie +
// session middleware stack does not panic when a client replays encrypted
// session cookies across multiple requests.  This was broken in Fiber
// v3.0.0-rc.3 (index-out-of-range in encryptcookie decryption).
func TestEncryptCookieSessionRoundTrip(t *testing.T) {
	// Use the same key-derivation as production (deriveEncryptionKey).
	secret := "test-secret-that-is-long-enough-for-production"
	encryptionKey := deriveEncryptionKey(secret)

	app := fiber.New()

	// Mirror the production middleware order exactly:
	// 1. encryptcookie  2. session  3. route handler
	app.Use(encryptcookie.New(encryptcookie.Config{
		Key: encryptionKey,
	}))

	sessionMiddleware, _ := session.NewWithStore(session.Config{
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	})
	app.Use(sessionMiddleware)

	// Handler that writes a session value on POST and reads it on GET.
	app.Post("/session-set", func(c fiber.Ctx) error {
		sess := session.FromContext(c)
		if sess == nil {
			return c.Status(500).SendString("no session")
		}
		sess.Set("user", "alice")
		return c.SendString("ok")
	})
	app.Get("/session-get", func(c fiber.Ctx) error {
		sess := session.FromContext(c)
		if sess == nil {
			return c.Status(500).SendString("no session")
		}
		val, _ := sess.Get("user").(string)
		return c.SendString(val)
	})

	// --- Request 1: establish a session ---
	req, _ := http.NewRequest("POST", "/session-set", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request 1 failed: %v", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("request 1: expected 200, got %d: %s", resp.StatusCode, body)
	}

	// Collect Set-Cookie headers from the response.
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("request 1: no cookies returned")
	}

	// --- Request 2: replay cookies (triggers encryptcookie decryption) ---
	req2, _ := http.NewRequest("GET", "/session-get", nil)
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request 2 failed (possible encryptcookie panic): %v", err)
	}
	body, _ := io.ReadAll(resp2.Body)
	if resp2.StatusCode != 200 {
		t.Fatalf("request 2: expected 200, got %d: %s", resp2.StatusCode, body)
	}
	if string(body) != "alice" {
		t.Errorf("request 2: expected session value 'alice', got %q", body)
	}

	// --- Request 3: one more round-trip to confirm stability ---
	cookies2 := resp2.Cookies()
	req3, _ := http.NewRequest("GET", "/session-get", nil)
	// Use cookies from resp2 if present, otherwise fall back to original.
	replayCookies := cookies2
	if len(replayCookies) == 0 {
		replayCookies = cookies
	}
	for _, c := range replayCookies {
		req3.AddCookie(c)
	}

	resp3, err := app.Test(req3)
	if err != nil {
		t.Fatalf("request 3 failed: %v", err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	if resp3.StatusCode != 200 {
		t.Fatalf("request 3: expected 200, got %d: %s", resp3.StatusCode, body3)
	}
	if string(body3) != "alice" {
		t.Errorf("request 3: expected session value 'alice', got %q", body3)
	}
}
