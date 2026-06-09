package services

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type AppInfo struct {
	Name        string `yaml:"name" json:"name"`
	Environment string `yaml:"environment" json:"environment"`
	APIURL      string `yaml:"api_url" json:"api_url"`
	CoreURL     string `yaml:"core_url" json:"core_url"`
}

type NetworkConfig struct {
	DefaultInterface   string `yaml:"default_interface" json:"default_interface"`
	CaptivePortal      bool   `yaml:"captive_portal" json:"captive_portal"`
	PortalIP           string `yaml:"portal_ip" json:"portal_ip"`
	PortalPort         int    `yaml:"portal_port" json:"portal_port"`
	PortalFallbackPort int    `yaml:"portal_fallback_port" json:"portal_fallback_port"`
	DNSIP              string `yaml:"dns_ip" json:"dns_ip"`
	Subnet             int    `yaml:"subnet" json:"subnet"`
	Template           string `yaml:"template" json:"template"`
}

type LoggingConfig struct {
	Level          string `yaml:"level" json:"level"`
	CredentialsLog string `yaml:"credentials_log" json:"credentials_log"`
}

type APIUser struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	Token    string `yaml:"token" json:"token"`
	Role     string `yaml:"role" json:"role"`
}

type AuthConfig struct {
	Enabled         bool      `yaml:"enabled" json:"enabled"`
	DefaultRole     string    `yaml:"default_role" json:"default_role"`
	AccessTokenTTL  int       `yaml:"access_token_ttl" json:"access_token_ttl"`
	RefreshTokenTTL int       `yaml:"refresh_token_ttl" json:"refresh_token_ttl"`
	Users           []APIUser `yaml:"users" json:"users"`
}

type Profile struct {
	Name        string `yaml:"name" json:"name"`
	SSID        string `yaml:"ssid" json:"ssid"`
	Password    string `yaml:"password" json:"password"`
	Channel     int    `yaml:"channel" json:"channel"`
	Mode        string `yaml:"mode" json:"mode"`
	DHCPEnabled bool   `yaml:"dhcp_enabled" json:"dhcp_enabled"`
	Security    string `yaml:"security" json:"security"`
}

type Config struct {
	App           AppInfo       `yaml:"app" json:"app"`
	Network       NetworkConfig `yaml:"network" json:"network"`
	Logging       LoggingConfig `yaml:"logging" json:"logging"`
	Auth          AuthConfig    `yaml:"auth" json:"auth"`
	ActiveProfile string        `yaml:"active_profile" json:"active_profile"`
	Profiles      []Profile     `yaml:"profiles" json:"profiles"`
}

func (cfg *Config) SetDefaults() {
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Network.Template == "" {
		cfg.Network.Template = "DarkLogin"
	}
	if cfg.Auth.AccessTokenTTL <= 0 {
		cfg.Auth.AccessTokenTTL = 600
	}
	if cfg.Auth.RefreshTokenTTL <= 0 {
		cfg.Auth.RefreshTokenTTL = 86400
	}
	if strings.TrimSpace(cfg.Auth.DefaultRole) == "" {
		cfg.Auth.DefaultRole = "viewer"
	}
	if cfg.ActiveProfile == "" && len(cfg.Profiles) > 0 {
		cfg.ActiveProfile = cfg.Profiles[0].Name
	}
}

func (auth *AuthConfig) FindUserByToken(token string) (*APIUser, bool) {
	for _, user := range auth.Users {
		if user.Token != "" && user.Token == token {
			return &user, true
		}
	}
	return nil, false
}

func (auth *AuthConfig) FindUserByCredentials(username, password string) (*APIUser, bool) {
	for _, user := range auth.Users {
		if user.Username == username && user.Password == password {
			return &user, true
		}
	}
	return nil, false
}

func (auth *AuthConfig) RoleForUser(user *APIUser) string {
	if user == nil {
		return ""
	}
	if strings.TrimSpace(user.Role) == "" {
		return auth.DefaultRole
	}
	return user.Role
}

func (cfg *Config) Validate() error {
	if strings.TrimSpace(cfg.App.Name) == "" {
		return errors.New("app.name is required")
	}
	if strings.TrimSpace(cfg.Network.DefaultInterface) == "" {
		return errors.New("network.default_interface is required")
	}
	if strings.TrimSpace(cfg.Network.PortalIP) == "" {
		return errors.New("network.portal_ip is required")
	}
	if net.ParseIP(cfg.Network.PortalIP) == nil {
		return fmt.Errorf("network.portal_ip is invalid: %s", cfg.Network.PortalIP)
	}
	if cfg.Network.Subnet <= 0 || cfg.Network.Subnet > 32 {
		return errors.New("network.subnet must be between 1 and 32")
	}
	if cfg.Network.PortalPort <= 0 || cfg.Network.PortalPort > 65535 {
		return errors.New("network.portal_port must be a valid port")
	}
	if cfg.Network.PortalFallbackPort <= 0 || cfg.Network.PortalFallbackPort > 65535 {
		return errors.New("network.portal_fallback_port must be a valid port")
	}
	if len(cfg.Profiles) == 0 {
		return errors.New("at least one profile is required")
	}
	profileNames := make(map[string]struct{})
	for _, profile := range cfg.Profiles {
		if strings.TrimSpace(profile.Name) == "" {
			return errors.New("each profile must have a name")
		}
		if strings.TrimSpace(profile.SSID) == "" {
			return fmt.Errorf("profile %q must have an ssid", profile.Name)
		}
		if profile.Channel < 1 || profile.Channel > 165 {
			return fmt.Errorf("profile %q has invalid channel %d", profile.Name, profile.Channel)
		}
		mode := strings.ToLower(strings.TrimSpace(profile.Mode))
		if mode != "" && mode != "a" && mode != "b" && mode != "g" && mode != "n" && mode != "ac" && mode != "ax" {
			return fmt.Errorf("profile %q has invalid mode %q", profile.Name, profile.Mode)
		}
		if _, ok := profileNames[profile.Name]; ok {
			return fmt.Errorf("duplicate profile name %q", profile.Name)
		}
		profileNames[profile.Name] = struct{}{}
	}
	if strings.TrimSpace(cfg.ActiveProfile) == "" {
		return errors.New("active_profile is required")
	}
	if _, ok := profileNames[cfg.ActiveProfile]; !ok {
		return fmt.Errorf("active_profile %q is not defined", cfg.ActiveProfile)
	}
	if cfg.Auth.Enabled {
		if len(cfg.Auth.Users) == 0 {
			return errors.New("auth.enabled is true but no auth.users are defined")
		}
		for _, user := range cfg.Auth.Users {
			if strings.TrimSpace(user.Username) == "" {
				return errors.New("auth user entries must include username")
			}
			if strings.TrimSpace(user.Password) == "" && strings.TrimSpace(user.Token) == "" {
				return fmt.Errorf("auth user %q must include a password or token", user.Username)
			}
		}
	}
	return nil
}

func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
