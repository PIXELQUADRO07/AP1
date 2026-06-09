package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ap1/project/services"
)

type authContextKey string

const contextUserRoleKey authContextKey = "userRole"

func GetUserRole(r *http.Request) string {
	if role, ok := r.Context().Value(contextUserRoleKey).(string); ok {
		return role
	}
	return ""
}

type Session struct {
	Username         string
	Role             string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type TokenStore struct {
	mu            sync.RWMutex
	accessTokens  map[string]Session
	refreshTokens map[string]Session
}

func NewTokenStore() *TokenStore {
	return &TokenStore{
		accessTokens:  make(map[string]Session),
		refreshTokens: make(map[string]Session),
	}
}

func (ts *TokenStore) newToken() string {
	buffer := make([]byte, 32)
	_, _ = rand.Read(buffer)
	return hex.EncodeToString(buffer)
}

func (ts *TokenStore) NewSession(username, role string, accessTTL, refreshTTL time.Duration) (accessToken, refreshToken string) {
	accessToken = ts.newToken()
	refreshToken = ts.newToken()
	now := time.Now().UTC()
	s := Session{
		Username:         username,
		Role:             role,
		AccessExpiresAt:  now.Add(accessTTL),
		RefreshExpiresAt: now.Add(refreshTTL),
	}

	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.accessTokens[accessToken] = s
	ts.refreshTokens[refreshToken] = s
	return
}

func (ts *TokenStore) ValidateAccessToken(token string) (Session, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	session, ok := ts.accessTokens[token]
	if !ok || session.AccessExpiresAt.Before(time.Now().UTC()) {
		return Session{}, false
	}
	return session, true
}

func (ts *TokenStore) ValidateRefreshToken(token string) (Session, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	session, ok := ts.refreshTokens[token]
	if !ok || session.RefreshExpiresAt.Before(time.Now().UTC()) {
		return Session{}, false
	}
	return session, true
}

func (ts *TokenStore) RefreshSession(refreshToken string, accessTTL, refreshTTL time.Duration) (newAccessToken, newRefreshToken string, session Session, err error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	session, ok := ts.refreshTokens[refreshToken]
	if !ok || session.RefreshExpiresAt.Before(time.Now().UTC()) {
		return "", "", Session{}, fmt.Errorf("refresh token invalid or expired")
	}

	delete(ts.refreshTokens, refreshToken)
	newAccessToken = ts.newToken()
	newRefreshToken = ts.newToken()
	now := time.Now().UTC()
	session.AccessExpiresAt = now.Add(accessTTL)
	session.RefreshExpiresAt = now.Add(refreshTTL)
	ts.accessTokens[newAccessToken] = session
	ts.refreshTokens[newRefreshToken] = session
	return newAccessToken, newRefreshToken, session, nil
}

func getBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func hasRole(userRole, requiredRole string) bool {
	if requiredRole == "" {
		return true
	}
	if userRole == "admin" {
		return true
	}
	return strings.EqualFold(userRole, requiredRole)
}

func TokenAuth(envToken string, authCfg *services.AuthConfig, store *TokenStore, requiredRole string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authCfg == nil || !authCfg.Enabled {
			next(w, r)
			return
		}

		token := getBearerToken(r)
		if token == "" {
			token = r.Header.Get("X-API-Token")
		}

		role := ""
		if token != "" && envToken != "" && token == envToken {
			role = "admin"
		} else if token != "" {
			if session, ok := store.ValidateAccessToken(token); ok {
				role = session.Role
			} else if user, ok := authCfg.FindUserByToken(token); ok {
				role = authCfg.RoleForUser(user)
			}
		}

		if role == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if !hasRole(role, requiredRole) {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), contextUserRoleKey, role)))
	}
}
