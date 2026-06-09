package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ap1/project/middleware"
	"github.com/ap1/project/services"
)

type loginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

type refreshPayload struct {
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Role         string `json:"role"`
}

func LoginHandler(authCfg *services.AuthConfig, store *middleware.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if authCfg == nil || !authCfg.Enabled {
			http.Error(w, "authentication is not enabled", http.StatusBadRequest)
			return
		}

		var payload loginPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		var user *services.APIUser
		var ok bool
		if payload.Token != "" {
			user, ok = authCfg.FindUserByToken(payload.Token)
		} else {
			user, ok = authCfg.FindUserByCredentials(payload.Username, payload.Password)
		}
		if !ok {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		role := authCfg.RoleForUser(user)
		accessToken, refreshToken := store.NewSession(user.Username, role, time.Duration(authCfg.AccessTokenTTL)*time.Second, time.Duration(authCfg.RefreshTokenTTL)*time.Second)

		response := authResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    authCfg.AccessTokenTTL,
			TokenType:    "Bearer",
			Role:         role,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func RefreshHandler(authCfg *services.AuthConfig, store *middleware.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if authCfg == nil || !authCfg.Enabled {
			http.Error(w, "authentication is not enabled", http.StatusBadRequest)
			return
		}

		var payload refreshPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		accessToken, refreshToken, session, err := store.RefreshSession(payload.RefreshToken, time.Duration(authCfg.AccessTokenTTL)*time.Second, time.Duration(authCfg.RefreshTokenTTL)*time.Second)
		if err != nil {
			http.Error(w, fmt.Sprintf("refresh failed: %v", err), http.StatusUnauthorized)
			return
		}

		response := authResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    authCfg.AccessTokenTTL,
			TokenType:    "Bearer",
			Role:         session.Role,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
