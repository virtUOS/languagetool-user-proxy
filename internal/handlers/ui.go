package handlers

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/virtuos/languagetool-user-proxy/internal/apikey"
	"github.com/virtuos/languagetool-user-proxy/internal/config"
	"github.com/virtuos/languagetool-user-proxy/internal/oidc"
	"github.com/virtuos/languagetool-user-proxy/internal/session"
)

type UIHandler struct {
	OIDCProvider     *oidc.Provider
	SessionManager   *session.Manager
	APIKeyManager    *apikey.Manager
	AccentColorStart string
	AccentColorEnd   string
	FrontendURL      string
	TemplateFS       fs.FS
}

type DashboardData struct {
	APIKey           string
	HasAPIKey        bool
	RegenError       string
	Username         string
	AccentColorStart string
	AccentColorEnd   string
	FrontendURL      string
}

func NewUIHandler(oidcProvider *oidc.Provider, sessionManager *session.Manager, apiKeyManager *apikey.Manager, cfg *config.Config, templateFS fs.FS) *UIHandler {
	return &UIHandler{
		OIDCProvider:     oidcProvider,
		SessionManager:   sessionManager,
		APIKeyManager:    apiKeyManager,
		AccentColorStart: cfg.UIAccentColorStart,
		AccentColorEnd:   cfg.UIAccentColorEnd,
		FrontendURL:      cfg.FrontendURL,
		TemplateFS:       templateFS,
	}
}

func (h *UIHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sessionToken := h.SessionManager.GetTokenFromRequest(r)
	if sessionToken == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	sess, err := h.SessionManager.GetSessionByToken(ctx, sessionToken)
	if err != nil {
		h.SessionManager.ClearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	apiKey, err := h.APIKeyManager.GetAPIKeyByUserID(ctx, sess.Session.UserID)
	if err != nil {
		apiKey = ""
	}

	data := DashboardData{
		APIKey:           apiKey,
		HasAPIKey:        apiKey != "",
		Username:         sess.User.Name.String,
		AccentColorStart: h.AccentColorStart,
		AccentColorEnd:   h.AccentColorEnd,
		FrontendURL:      h.FrontendURL,
	}

	tmplBytes, err := fs.ReadFile(h.TemplateFS, "templates/dashboard.html")
	if err != nil {
		log.Printf("Failed to read dashboard template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	tmpl := template.Must(template.New("dashboard").Parse(string(tmplBytes)))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func (h *UIHandler) RegenerateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	sessionToken := h.SessionManager.GetTokenFromRequest(r)
	if sessionToken == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sess, err := h.SessionManager.GetSessionByToken(ctx, sessionToken)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	newKey, err := h.APIKeyManager.RegenerateAPIKey(ctx, sess.Session.UserID)
	if err != nil {
		log.Printf("Failed to regenerate API key for user %d: %v", sess.Session.UserID, err)
		http.Error(w, "Failed to regenerate key", http.StatusInternalServerError)
		return
	}

	// Return the new key as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"key": newKey,
	})
}

func (h *UIHandler) SessionStatus(w http.ResponseWriter, r *http.Request) {
	token := h.SessionManager.GetTokenFromRequest(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := h.SessionManager.GetSessionByToken(r.Context(), token); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *UIHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	ctx := r.Context()
	sessionToken := h.SessionManager.GetTokenFromRequest(r)

	if sessionToken != "" {
		// Get the session to retrieve the ID token
		sess, err := h.SessionManager.GetSessionByToken(ctx, sessionToken)
		if err == nil {
			// Delete the session
			h.SessionManager.DeleteSession(ctx, sess.Session.ID)

			// Clear the session cookie
			h.SessionManager.ClearSessionCookie(w)

			// Get the logout URL from the OIDC provider
			logoutURL := h.OIDCProvider.GetLogoutURL(sess.Session.IDToken)
			if logoutURL != "" {
				// Redirect to the OIDC provider's logout endpoint if configured
				http.Redirect(w, r, logoutURL, http.StatusFound)
				return
			}
		}

		// Clear the session cookie even if we couldn't get the session
		h.SessionManager.ClearSessionCookie(w)
	}

	// Fallback: just redirect to home if no session
	http.Redirect(w, r, "/", http.StatusFound)
}
