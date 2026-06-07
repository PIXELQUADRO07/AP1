package services

import (
	"os"

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

type Profile struct {
	Name        string `yaml:"name" json:"name"`
	SSID        string `yaml:"ssid" json:"ssid"`
	Password    string `yaml:"password" json:"password"`
	Channel     int    `yaml:"channel" json:"channel"`
	Mode        string `yaml:"mode" json:"mode"`
	DHCPEnabled bool   `yaml:"dhcp_enabled" json:"dhcp_enabled"`
}

type Config struct {
	App           AppInfo       `yaml:"app" json:"app"`
	Network       NetworkConfig `yaml:"network" json:"network"`
	Logging       LoggingConfig `yaml:"logging" json:"logging"`
	ActiveProfile string        `yaml:"active_profile" json:"active_profile"`
	Profiles      []Profile     `yaml:"profiles" json:"profiles"`
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
	return &cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
