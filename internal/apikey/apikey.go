package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/virtuos/languagetool-user-proxy/internal/database/queries"
)

const keyLength = 64

func GenerateKey() (string, error) {
	b := make([]byte, keyLength/2)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func GetKeyPrefix(key string) string {
	if len(key) >= 8 {
		return key[:8]
	}
	return key
}

type Manager struct {
	Queries *queries.Queries
}

func NewManager(queries *queries.Queries) *Manager {
	return &Manager{
		Queries: queries,
	}
}

func (m *Manager) GetOrCreateAPIKey(ctx context.Context, userID int64) (string, error) {
	key, err := m.GetAPIKeyByUserID(ctx, userID)
	if err == nil && key != "" {
		return key, nil
	}

	return m.RegenerateAPIKey(ctx, userID)
}

func (m *Manager) GetAPIKeyByUserID(ctx context.Context, userID int64) (string, error) {
	apiKey, err := m.Queries.GetAPIKeyByUserID(ctx, userID)
	if err != nil {
		return "", err
	}

	// We only store the hash, so we need to return a placeholder
	// The actual key is only shown once after generation
	return apiKey.KeyPrefix + "...(hidden)", nil
}

func (m *Manager) RegenerateAPIKey(ctx context.Context, userID int64) (string, error) {
	// Delete existing key
	existingKey, err := m.Queries.GetAPIKeyByUserID(ctx, userID)
	if err == nil {
		m.Queries.DeleteAPIKey(ctx, existingKey.ID)
	}

	// Generate new key
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}

	hash := HashKey(key)
	prefix := GetKeyPrefix(key)

	_, err = m.Queries.CreateAPIKey(ctx, userID, hash, prefix)
	if err != nil {
		return "", fmt.Errorf("failed to store API key: %w", err)
	}

	return key, nil
}

func (m *Manager) ValidateAPIKey(ctx context.Context, key string) (int64, error) {
	hash := HashKey(key)

	apiKey, err := m.Queries.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		return 0, fmt.Errorf("invalid API key")
	}

	return apiKey.UserID, nil
}

func ExtractKeyFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 0 {
		// Check if the first part looks like an API key (64 hex chars)
		if len(parts[0]) == keyLength {
			return parts[0]
		}
	}
	return ""
}
