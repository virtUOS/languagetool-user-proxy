package handlers

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/virtuos/languagetool-user-proxy/internal/apikey"
	"github.com/virtuos/languagetool-user-proxy/internal/metrics"
)

type ProxyHandler struct {
	BackendURL    *url.URL
	APIKeyManager *apikey.Manager
	Metrics       *metrics.Metrics
}

func NewProxyHandler(backendURL string, apiKeyManager *apikey.Manager, metrics *metrics.Metrics) (*ProxyHandler, error) {
	backend, err := url.Parse(backendURL)
	if err != nil {
		return nil, err
	}

	return &ProxyHandler{
		BackendURL:    backend,
		APIKeyManager: apiKeyManager,
		Metrics:       metrics,
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

	// Wrap the ResponseWriter to capture status code
	wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

	proxy.ServeHTTP(wrapped, r)

	// Increment request counter with status code
	h.Metrics.IncrementRequest(proxypath, wrapped.statusCodeStr())
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) statusCodeStr() string {
	if rw.statusCode >= 200 && rw.statusCode < 300 {
		return "2xx"
	} else if rw.statusCode >= 300 && rw.statusCode < 400 {
		return "3xx"
	} else if rw.statusCode >= 400 && rw.statusCode < 500 {
		return "4xx"
	}
	return "5xx"
}
