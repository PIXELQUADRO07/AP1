package services

import (
	"os"

	"gopkg.in/yaml.v3"
)

type PluginManifest struct {
	Name        string `yaml:"name" json:"name"`
	Type        string `yaml:"type" json:"type"`
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	Description string `yaml:"description" json:"description"`
}

type PluginConfig struct {
	Plugins []PluginManifest `yaml:"plugins" json:"plugins"`
}

func LoadPluginConfig(path string) (*PluginConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg PluginConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SavePluginConfig(path string, cfg *PluginConfig) error {
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func FindPlugin(cfg *PluginConfig, name string) *PluginManifest {
	for i := range cfg.Plugins {
		if cfg.Plugins[i].Name == name {
			return &cfg.Plugins[i]
		}
	}
	return nil
}
