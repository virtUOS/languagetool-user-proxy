package models

import "time"

// User represents a user in the system
type User struct {
	ID        int64     `json:"id"`
	OIDCSub   string    `json:"oidc_sub"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// APIKey represents an API key for a user
type APIKey struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	KeyHash   string    `json:"key_hash"`
	KeyPrefix string    `json:"key_prefix"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Session represents a user session
type Session struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
