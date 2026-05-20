package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/virtuos/languagetool-user-proxy/internal/database/queries"
)

const sessionCookieName = "session_token"

type Manager struct {
	Queries         *queries.Queries
	SessionDuration time.Duration
	CookieSecret    string
}

// SessionWithUser contains session data along with the associated user information
type SessionWithUser struct {
	Session queries.Session
	User    queries.User
}

func NewManager(queries *queries.Queries, duration time.Duration, secret string) *Manager {
	return &Manager{
		Queries:         queries,
		SessionDuration: duration,
		CookieSecret:    secret,
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (m *Manager) CreateSession(ctx context.Context, userID int64, idToken string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(m.SessionDuration)

	_, err = m.Queries.CreateSession(ctx, queries.CreateSessionParams{
		UserID:    userID,
		Token:     token,
		IDToken:   idToken,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}

func (m *Manager) GetSessionByToken(ctx context.Context, token string) (*SessionWithUser, error) {
	session, err := m.Queries.GetSessionByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		m.Queries.DeleteSession(ctx, session.ID)
		return nil, fmt.Errorf("session expired")
	}

	// Fetch user data
	user, err := m.Queries.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	return &SessionWithUser{
		Session: session,
		User:    user,
	}, nil
}

func (m *Manager) DeleteSession(ctx context.Context, sessionID int64) error {
	return m.Queries.DeleteSession(ctx, sessionID)
}

func (m *Manager) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	result, err := m.Queries.DeleteExpiredSessions(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return result, nil
}

func (m *Manager) SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(m.SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Manager) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

func (m *Manager) GetTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
