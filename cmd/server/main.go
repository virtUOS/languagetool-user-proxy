package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/virtuos/languagetool-user-proxy/internal/apikey"
	"github.com/virtuos/languagetool-user-proxy/internal/config"
	"github.com/virtuos/languagetool-user-proxy/internal/database"
	"github.com/virtuos/languagetool-user-proxy/internal/database/queries"
	"github.com/virtuos/languagetool-user-proxy/internal/handlers"
	"github.com/virtuos/languagetool-user-proxy/internal/oidc"
	"github.com/virtuos/languagetool-user-proxy/internal/session"
)

func main() {
	// Load .env file (optional - won't error if not found)
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Read schema
	schemaSQL, err := os.ReadFile("internal/database/queries/schema.sql")
	if err != nil {
		log.Fatalf("Failed to read schema: %v", err)
	}

	// Run migrations
	if err := db.Migrate(string(schemaSQL)); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize queries
	queries := queries.New(db.DB)

	// Initialize OIDC provider
	oidcProvider, err := oidc.NewProvider(cfg, queries)
	if err != nil {
		log.Fatalf("Failed to create OIDC provider: %v", err)
	}

	// Initialize session manager
	sessionManager := session.NewManager(queries, cfg.SessionDuration, cfg.CookieSecret)

	// Initialize API key manager
	apiKeyManager := apikey.NewManager(queries)

	// Initialize handlers
	uiHandler := handlers.NewUIHandler(oidcProvider, sessionManager, apiKeyManager, cfg)
	proxyHandler, err := handlers.NewProxyHandler(cfg.BackendURL, apiKeyManager)
	if err != nil {
		log.Fatalf("Failed to create proxy handler: %v", err)
	}

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// UI routes
	r.Get("/", uiHandler.Dashboard)
	r.Post("/logout", uiHandler.Logout)
	r.Post("/regenerate-key", uiHandler.RegenerateKey)

	// OIDC routes
	r.Get("/login", oidcProvider.LoginHandler)
	r.Get("/callback", func(w http.ResponseWriter, r *http.Request) {
		result := oidcProvider.CallbackHandler(w, r)
		if result.Error != nil {
			http.Error(w, result.Error.Error(), http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		user, err := oidcProvider.GetOrCreateUser(ctx, result.UserInfo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Create or get API key for user
		_, err = apiKeyManager.GetAPIKeyByUserID(ctx, user.ID)
		if err != nil {
			log.Printf("Failed to get API key: %v", err)
		}

		// Create session
		token, err := sessionManager.CreateSession(ctx, user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sessionManager.SetSessionCookie(w, token)
		http.Redirect(w, r, "/", http.StatusFound)
	})

	// Proxy routes - catch all paths that start with /{apiKey}/
	r.Mount("/", proxyHandler)

	// Create server
	server := &http.Server{
		Addr:         cfg.ListenAddress + ":" + cfg.ListenPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Printf("Starting server on http://%s:%s", cfg.ListenAddress, cfg.ListenPort)
	log.Printf("Backend URL: %s", cfg.BackendURL)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
