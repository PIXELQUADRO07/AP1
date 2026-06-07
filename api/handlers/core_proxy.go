package handlers

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

func postToCore(coreURL, path string, payload interface{}) (*http.Response, error) {
    if coreURL == "" {
        return nil, fmt.Errorf("core service URL not configured")
    }

    raw, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to encode payload: %w", err)
    }

    req, err := http.NewRequest(http.MethodPost, coreURL+path, bytes.NewReader(raw))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to send request to core: %w", err)
    }
    return resp, nil
}

func getFromCore(coreURL, path string) (*http.Response, error) {
    if coreURL == "" {
        return nil, fmt.Errorf("core service URL not configured")
    }

    resp, err := http.Get(coreURL + path)
    if err != nil {
        return nil, fmt.Errorf("failed to send request to core: %w", err)
    }
    return resp, nil
}

func writeCoreResponse(w http.ResponseWriter, resp *http.Response) {
    defer resp.Body.Close()
    w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
    w.WriteHeader(resp.StatusCode)
    _, _ = io.Copy(w, resp.Body)
}
