package server

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/encryptcookie"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"

	"golinks/internal/config"
)

// Server wraps the Fiber app and configuration.
type Server struct {
	App *fiber.App
	Cfg *config.Config
}

// New creates a new server with middleware configured.
func New(cfg *config.Config) *Server {
	// Setup template engine
	engine := html.New("./views", ".html")
	engine.Reload(cfg.IsDev())

	// Initialize Fiber
	app := fiber.New(fiber.Config{
		Views:       engine,
		ViewsLayout: "layouts/main",
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			message := "Internal Server Error"

			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				message = e.Message
			}

			return c.Status(code).Render("error", fiber.Map{
				"Title":       "Error",
				"Message":     message,
				"SiteTitle":   cfg.SiteTitle,
				"SiteTagline": cfg.SiteTagline,
				"SiteFooter":  cfg.SiteFooter,
				"SiteLogoURL": cfg.SiteLogoURL,
			})
		},
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// CORS middleware
	corsOrigins := cfg.BaseURL
	if cfg.CORSOrigins != "" {
		corsOrigins = cfg.CORSOrigins
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(corsOrigins, ","),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "HX-Request", "HX-Current-URL", "HX-Target"},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Cookie encryption middleware
	encryptionKey := deriveEncryptionKey(cfg.SessionSecret)
	app.Use(encryptcookie.New(encryptcookie.Config{
		Key: encryptionKey,
	}))

	// Session middleware
	sessionMiddleware, _ := session.NewWithStore(session.Config{
		CookieSecure:   cfg.TLSEnabled || !cfg.IsDev(),
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	})
	app.Use(sessionMiddleware)

	// Rate limiting middleware - 100 requests per minute per IP
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded. Please try again later.",
			})
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
	}))

	// Static files
	app.Get("/static/*", static.New("./static"))

	return &Server{
		App: app,
		Cfg: cfg,
	}
}

// Start starts the server with the configured address and TLS settings.
func (s *Server) Start() error {
	if s.Cfg.TLSEnabled {
		tlsConfig := buildTLSConfig(s.Cfg)
		listenConfig := fiber.ListenConfig{
			CertFile:      s.Cfg.TLSCertFile,
			CertKeyFile:   s.Cfg.TLSKeyFile,
			TLSConfigFunc: func(tc *tls.Config) { *tc = *tlsConfig },
		}
		if s.Cfg.TLSCAFile != "" {
			log.Printf("Starting server with mTLS on %s", s.Cfg.ServerAddr)
		} else {
			log.Printf("Starting server with TLS on %s", s.Cfg.ServerAddr)
		}
		return s.App.Listen(s.Cfg.ServerAddr, listenConfig)
	}
	return s.App.Listen(s.Cfg.ServerAddr)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	return s.App.Shutdown()
}

// deriveEncryptionKey derives a 32-byte encryption key from the session secret.
func deriveEncryptionKey(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// buildTLSConfig creates a TLS config for mTLS if CA file is provided.
func buildTLSConfig(cfg *config.Config) *tls.Config {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if cfg.TLSCAFile != "" {
		caCert, err := os.ReadFile(cfg.TLSCAFile)
		if err != nil {
			log.Fatalf("Failed to read CA file: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			log.Fatal("Failed to parse CA certificate")
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig
}
