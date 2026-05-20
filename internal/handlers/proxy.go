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
	key, proxypath, err := apikey.ExtractKeyFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "Invalid path or API key", http.StatusUnauthorized)
		return
	}

	// Only allow /v2/ paths to be proxied
	if !strings.HasPrefix(proxypath, "/v2/") {
		http.NotFound(w, r)
		return
	}

	// Validate API key
	userID, err := h.APIKeyManager.ValidateAPIKey(r.Context(), key)
	if err != nil {
		log.Printf("Invalid API key: %v", err)
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	keyPrefix := apikey.GetKeyPrefix(key)
	safePath := "/" + keyPrefix + "..." + proxypath
	log.Printf("Proxy request from user %d: %s %s", userID, r.Method, safePath)

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(h.BackendURL)

	// Modify the request to remove the API key from the path
	r.URL.Path = proxypath

	// Set X-Forwarded headers
	proxy.ServeHTTP(w, r)
}
