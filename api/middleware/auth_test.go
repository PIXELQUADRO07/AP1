package middleware

import (
	"testing"
	"time"
)

func TestTokenStoreNewSession(t *testing.T) {
	store := NewTokenStore()
	accessToken, refreshToken := store.NewSession("admin", "admin", 1*time.Hour, 24*time.Hour)
	if accessToken == "" || refreshToken == "" {
		t.Fatal("expected valid tokens")
	}
}

func TestTokenStoreRefreshSession(t *testing.T) {
	store := NewTokenStore()
	_, refreshToken := store.NewSession("alice", "viewer", 10*time.Second, 1*time.Hour)
	time.Sleep(10 * time.Millisecond)
	newAccess, newRefresh, session, err := store.RefreshSession(refreshToken, 10*time.Second, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Fatal("expected refreshed tokens")
	}
	if session.Username != "alice" {
		t.Fatalf("expected username alice, got %q", session.Username)
	}
	if session.Role != "viewer" {
		t.Fatalf("expected role viewer, got %q", session.Role)
	}
	if newAccess == refreshToken || newRefresh == refreshToken {
		t.Fatal("expected new tokens to differ from the original refresh token")
	}
}
