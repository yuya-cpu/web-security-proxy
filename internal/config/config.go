package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config はアプリケーション全体の設定を保持する。
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Proxy    ProxyConfig    `yaml:"proxy"`
	Database DatabaseConfig `yaml:"database"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type ProxyConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// Load はYAMLファイルから設定を読み込む。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Proxy.Host == "" {
		c.Proxy.Host = "127.0.0.1"
	}
	if c.Proxy.Port == 0 {
		c.Proxy.Port = 8888
	}
	if c.Database.Path == "" {
		c.Database.Path = "./data/proxy.db"
	}
}

func (c ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c ProxyConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
