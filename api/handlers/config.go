package handlers

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/ap1/project/services"
)

func ConfigHandler(cfg *services.Config) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        encoded, err := json.Marshal(cfg)
        if err != nil {
            http.Error(w, fmt.Sprintf("failed to encode config: %v", err), http.StatusInternalServerError)
            return
        }
        w.Write(encoded)
    }
}
