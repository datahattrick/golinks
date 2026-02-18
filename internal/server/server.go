package server

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"log/slog"
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
	pgstore "github.com/gofiber/storage/postgres/v3"
	"github.com/gofiber/template/html/v3"

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

			slog.Error("request error",
				"status", code,
				"method", c.Method(),
				"path", c.Path(),
				"ip", c.IP(),
				"error", err.Error(),
			)

			// For API requests, static files, and non-HTML clients, return JSON
			if strings.HasPrefix(c.Path(), "/api/") ||
				strings.HasPrefix(c.Path(), "/static/") ||
				!strings.Contains(c.Get("Accept"), "text/html") {
				return c.Status(code).JSON(fiber.Map{
					"status": "error",
					"error":  message,
				})
			}

			// Render HTML error page; fall back to plain text if template fails
			renderErr := c.Status(code).Render("error", fiber.Map{
				"Title":                    "Error",
				"Message":                  message,
				"SiteTitle":                cfg.SiteTitle,
				"SiteTagline":              cfg.SiteTagline,
				"SiteFooter":               cfg.SiteFooter,
				"SiteLogoURL":              cfg.SiteLogoURL,
				"EnableAnimatedBackground": cfg.EnableAnimatedBackground,
			})
			if renderErr != nil {
				slog.Error("failed to render error template",
					"render_error", renderErr.Error(),
					"original_error", err.Error(),
				)
				return c.Status(code).SendString(message)
			}
			return nil
		},
	})

	// --- Middleware applied to ALL routes (including static) ---
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	app.Use(logger.New(logger.Config{
		// Write to stderr so container log collectors capture Fiber request logs
		// alongside slog output (which also writes to stderr).
		Stream:     os.Stderr,
		Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))

	// Static files - registered BEFORE session/cookie/rate-limit middleware.
	// This prevents 500 errors caused by cookie decryption or session
	// initialization failures on asset requests.
	app.Get("/static/*", static.New("./static", static.Config{
		MaxAge: 3600,
	}))

	slog.Debug("static file middleware registered", "root", "./static")

	// --- Middleware applied only to dynamic routes (registered after static) ---

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
	sessionCfg := session.Config{
		CookieSecure:   cfg.TLSEnabled || !cfg.IsDev(),
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	}
	if cfg.SessionStore == "postgres" {
		sessionCfg.Storage = pgstore.New(pgstore.Config{
			ConnectionURI: cfg.DatabaseURL,
			Table:         "fiber_sessions",
			GCInterval:    10 * time.Minute,
		})
		slog.Info("session store: postgres")
	} else {
		slog.Info("session store: memory")
	}
	sessionMiddleware, _ := session.NewWithStore(sessionCfg)
	app.Use(sessionMiddleware)

	// Rate limiting middleware - 100 requests per minute per IP
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			slog.Warn("rate limit exceeded", "ip", c.IP(), "path", c.Path())
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded. Please try again later.",
			})
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
	}))

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
			CertFile:    s.Cfg.TLSCertFile,
			CertKeyFile: s.Cfg.TLSKeyFile,
			TLSConfigFunc: func(tc *tls.Config) {
				tc.MinVersion = tlsConfig.MinVersion
				tc.ClientCAs = tlsConfig.ClientCAs
				tc.ClientAuth = tlsConfig.ClientAuth
			},
		}
		if s.Cfg.TLSCAFile != "" {
			slog.Info("starting server with mTLS", "addr", s.Cfg.ServerAddr)
		} else {
			slog.Info("starting server with TLS", "addr", s.Cfg.ServerAddr)
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
			slog.Error("failed to read CA file", "path", cfg.TLSCAFile, "error", err)
			os.Exit(1)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			slog.Error("failed to parse CA certificate", "path", cfg.TLSCAFile)
			os.Exit(1)
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig
}
