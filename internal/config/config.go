/*
 * Copyright (C) 2026 Mustafa Naseer (Mustafa Gaeed)
 *
 * This file is part of uruflow.
 *
 * uruflow is free software: you can redistribute it and/or modify
 * it under the terms of the MIT License as described in the
 * LICENSE file distributed with this project.
 *
 * uruflow is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * MIT License for more details.
 *
 * You should have received a copy of the MIT License
 * along with uruflow. If not, see the LICENSE file in the project root.
 */

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urustack/uruflow/internal/models"
	"github.com/urustack/uruflow/pkg/helper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       ServerConfig        `yaml:"server"`
	Webhook      WebhookConfig       `yaml:"webhook"`
	TLS          TLSConfig           `yaml:"tls"`
	Agents       []AgentConfig       `yaml:"agents"`
	Repositories []models.Repository `yaml:"repositories"`
}

type ServerConfig struct {
	HTTPPort int    `yaml:"http_port"`
	TCPPort  int    `yaml:"tcp_port"`
	Host     string `yaml:"host"`
	DataDir  string `yaml:"data_dir"`
}

type WebhookConfig struct {
	Path   string `yaml:"path"`
	Secret string `yaml:"secret"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	AutoCert bool   `yaml:"auto_cert"`
}

type AgentConfig struct {
	ID    string `yaml:"id"`
	Name  string `yaml:"name"`
	Token string `yaml:"token"`
}

var (
	DefaultConfigPath = "/etc/uruflow/config.yaml"
	DefaultDataDir    = "/var/lib/uruflow"
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.setDefaults()
	return &cfg, nil
}

func (c *Config) setDefaults() {
	if c.Server.HTTPPort == 0 {
		c.Server.HTTPPort = 9000
	}
	if c.Server.TCPPort == 0 {
		c.Server.TCPPort = 9001
	}
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.DataDir == "" {
		c.Server.DataDir = DefaultDataDir
	}
	if c.Webhook.Path == "" {
		c.Webhook.Path = "/webhook"
	}
}

func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPPort: 9000,
			TCPPort:  9001,
			Host:     "0.0.0.0",
			DataDir:  DefaultDataDir,
		},
		Webhook: WebhookConfig{
			Path:   "/webhook",
			Secret: helper.GenerateSecret(),
		},
		TLS: TLSConfig{
			Enabled:  false,
			AutoCert: false,
		},
		Agents:       []AgentConfig{},
		Repositories: []models.Repository{},
	}
}

func (c *Config) AddAgent(name string) (string, string, error) {
	for _, a := range c.Agents {
		if a.Name == name {
			return "", "", fmt.Errorf("agent %s already exists", name)
		}
	}

	id := helper.GenerateID()
	token := helper.GenerateToken()

	c.Agents = append(c.Agents, AgentConfig{
		ID:    id,
		Name:  name,
		Token: token,
	})

	return id, token, nil
}

func (c *Config) GetAgent(id string) *AgentConfig {
	for i := range c.Agents {
		if c.Agents[i].ID == id {
			return &c.Agents[i]
		}
	}
	return nil
}

func (c *Config) GetAgentByToken(token string) *AgentConfig {
	for i := range c.Agents {
		if c.Agents[i].Token == token {
			return &c.Agents[i]
		}
	}
	return nil
}

func (c *Config) GetAgentByName(name string) *AgentConfig {
	for i := range c.Agents {
		if c.Agents[i].Name == name {
			return &c.Agents[i]
		}
	}
	return nil
}

func (c *Config) RemoveAgent(id string) bool {
	for i := range c.Agents {
		if c.Agents[i].ID == id {
			c.Agents = append(c.Agents[:i], c.Agents[i+1:]...)
			return true
		}
	}
	return false
}

func (c *Config) AddRepository(repo models.Repository) error {
	for _, r := range c.Repositories {
		if r.Name == repo.Name {
			return fmt.Errorf("repository %s already exists", repo.Name)
		}
	}
	c.Repositories = append(c.Repositories, repo)
	return nil
}

func (c *Config) GetRepository(name string) *models.Repository {
	for i := range c.Repositories {
		if c.Repositories[i].Name == name {
			return &c.Repositories[i]
		}
	}
	return nil
}

func (c *Config) RemoveRepository(name string) bool {
	for i := range c.Repositories {
		if c.Repositories[i].Name == name {
			c.Repositories = append(c.Repositories[:i], c.Repositories[i+1:]...)
			return true
		}
	}
	return false
}
