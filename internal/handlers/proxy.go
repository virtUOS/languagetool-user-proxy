package handlers

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/virtuos/languagetool-user-proxy/internal/apikey"
)

type ProxyHandler struct {
	BackendURL    *url.URL
	APIKeyManager *apikey.Manager
}

func NewProxyHandler(backendURL string, apiKeyManager *apikey.Manager) (*ProxyHandler, error) {
	backend, err := url.Parse(backendURL)
	if err != nil {
		return nil, err
	}

	return &ProxyHandler{
		BackendURL:    backend,
		APIKeyManager: apiKeyManager,
	}, nil
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract API key from path
	key := apikey.ExtractKeyFromPath(r.URL.Path)
	if key == "" {
		http.Error(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	// Validate API key
	userID, err := h.APIKeyManager.ValidateAPIKey(r.Context(), key)
	if err != nil {
		log.Printf("Invalid API key: %v", err)
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	log.Printf("Proxy request from user %d: %s %s", userID, r.Method, r.URL.Path)

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(h.BackendURL)

	// Modify the request to remove the API key from the path
	originalPath := r.URL.Path
	r.URL.Path = strings.TrimPrefix(originalPath, "/"+key)
	if !strings.HasPrefix(r.URL.Path, "/") {
		r.URL.Path = "/" + r.URL.Path
	}

	// Only allow /v2/ paths to be proxied
	if !strings.HasPrefix(r.URL.Path, "/v2/") {
		http.NotFound(w, r)
		return
	}

	// Set X-Forwarded headers
	proxy.ServeHTTP(w, r)
}
