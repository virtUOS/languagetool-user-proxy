package oidc

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/virtuos/languagetool-user-proxy/internal/config"
	"github.com/virtuos/languagetool-user-proxy/internal/database/queries"
	"golang.org/x/oauth2"
)

func init() {
	// Register types for gob encoding if needed
}

type Provider struct {
	Provider   *oidc.Provider
	OIDCConfig *oauth2.Config
	Config     *config.Config
	Queries    *queries.Queries
	Verifier   *oidc.IDTokenVerifier
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

	return &Provider{
		Provider:   provider,
		OIDCConfig: oidcConfig,
		Config:     cfg,
		Queries:    db,
		Verifier:   verifier,
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
		Secure:   false,
	})

	authURL := p.OIDCConfig.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

type CallbackResult struct {
	UserInfo *UserInfo
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

	user, err = p.Queries.CreateUser(ctx, userInfo.Subject, userInfo.Email, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
