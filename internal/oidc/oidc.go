package oidc

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/virtuos/languagetool-user-proxy/internal/config"
	"github.com/virtuos/languagetool-user-proxy/internal/database/queries"
	"golang.org/x/oauth2"
)

func init() {
	// Register types for gob encoding if needed
}

type Provider struct {
	Provider           *oidc.Provider
	OIDCConfig         *oauth2.Config
	Config             *config.Config
	Queries            *queries.Queries
	Verifier           *oidc.IDTokenVerifier
	EndSessionEndpoint string
}

type UserInfo struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
}

func NewProvider(cfg *config.Config, db *queries.Queries) (*Provider, error) {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	oidcConfig := &oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.OIDCRedirectURI,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.OIDCClientID,
	})

	// Discover end_session_endpoint from OIDC provider metadata
	// The go-oidc library loads the discovery document when creating the provider
	// We'll fetch the discovery document to get the actual end_session_endpoint
	endSessionEndpoint := ""

	// Fetch the discovery document to get the end_session_endpoint
	discoveryURL, err := url.JoinPath(cfg.OIDCIssuerURL, ".well-known/openid-configuration")
	if err == nil {
		resp, err := http.Get(discoveryURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err == nil {
				var discovery struct {
					EndSessionEndpoint string `json:"end_session_endpoint"`
				}
				if err := json.Unmarshal(body, &discovery); err == nil && discovery.EndSessionEndpoint != "" {
					endSessionEndpoint = discovery.EndSessionEndpoint
				}
			}
		}
	}

	return &Provider{
		Provider:           provider,
		OIDCConfig:         oidcConfig,
		Config:             cfg,
		Queries:            db,
		Verifier:           verifier,
		EndSessionEndpoint: endSessionEndpoint,
	}, nil
}

func (p *Provider) LoginHandler(w http.ResponseWriter, r *http.Request) {
	state := generateState()

	http.SetCookie(w, &http.Cookie{
		Name:     "oidc_state_" + state,
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})

	authURL := p.OIDCConfig.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

type CallbackResult struct {
	UserInfo *UserInfo
	IDToken  string
	Error    error
}

func (p *Provider) CallbackHandler(w http.ResponseWriter, r *http.Request) *CallbackResult {
	state := r.URL.Query().Get("state")

	stateCookie, err := r.Cookie("oidc_state_" + state)
	if err != nil {
		return &CallbackResult{Error: fmt.Errorf("missing state cookie")}
	}
	if stateCookie.Value != state {
		return &CallbackResult{Error: fmt.Errorf("invalid state")}
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "oidc_state_" + state,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	ctx := r.Context()
	code := r.URL.Query().Get("code")

	tokens, err := p.OIDCConfig.Exchange(ctx, code)
	if err != nil {
		return &CallbackResult{Error: fmt.Errorf("failed to exchange code: %w", err)}
	}

	rawIDToken, ok := tokens.Extra("id_token").(string)
	if !ok {
		return &CallbackResult{Error: fmt.Errorf("no id_token in response")}
	}

	idToken, err := p.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return &CallbackResult{Error: fmt.Errorf("failed to verify id_token: %w", err)}
	}

	var claims struct {
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Subject           string `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return &CallbackResult{Error: fmt.Errorf("failed to parse claims: %w", err)}
	}

	return &CallbackResult{
		UserInfo: &UserInfo{
			Subject: claims.Subject,
			Email:   claims.Email,
			Name:    claims.PreferredUsername,
		},
		IDToken: rawIDToken,
	}
}

func (p *Provider) GetOrCreateUser(ctx context.Context, userInfo *UserInfo) (*queries.User, error) {
	user, err := p.Queries.GetUserByOIDCSub(ctx, userInfo.Subject)
	if err == nil {
		return &user, nil
	}

	var name sql.NullString
	if userInfo.Name != "" {
		name = sql.NullString{String: userInfo.Name, Valid: true}
	} else {
		name = sql.NullString{String: userInfo.Email, Valid: true}
	}

	user, err = p.Queries.CreateUser(ctx, queries.CreateUserParams{
		OidcSub: userInfo.Subject,
		Email:   userInfo.Email,
		Name:    name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

func generateState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Failed to generate random state: %v", err)
	}
	return hex.EncodeToString(b)
}

// GetLogoutURL generates the URL to redirect to for OIDC logout
// It includes the id_token_hint and post_logout_redirect_uri
// Returns empty string if end_session_endpoint is not configured
func (p *Provider) GetLogoutURL(idToken string) string {
	if p.EndSessionEndpoint == "" {
		return ""
	}

	// Parse the end session endpoint URL
	u, err := url.Parse(p.EndSessionEndpoint)
	if err != nil {
		return ""
	}

	// Get existing query parameters and add our logout parameters
	q := u.Query()
	q.Set("id_token_hint", idToken)
	q.Set("post_logout_redirect_uri", p.Config.FrontendURL)
	u.RawQuery = q.Encode()

	return u.String()
}
