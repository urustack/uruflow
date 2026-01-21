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
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

var (
	DefaultConfigPath string
	DefaultDataDir    string
	DefaultPidFile    string
	DefaultLogFile    string
)

func init() {
	if runtime.GOOS == "windows" {
		homeDir, _ := os.UserHomeDir()
		DefaultConfigPath = filepath.Join(homeDir, ".uruflow", "agent.yaml")
		DefaultDataDir = filepath.Join(homeDir, ".uruflow", "data")
		DefaultPidFile = filepath.Join(homeDir, ".uruflow", "agent.pid")
		DefaultLogFile = filepath.Join(homeDir, ".uruflow", "agent.log")
	} else {
		if os.Geteuid() == 0 {
			DefaultConfigPath = "/etc/uruflow/agent.yaml"
			DefaultDataDir = "/var/lib/uruflow-agent"
			DefaultPidFile = "/var/run/uruflow-agent.pid"
			DefaultLogFile = "/var/log/uruflow-agent.log"
		} else {
			homeDir, _ := os.UserHomeDir()
			DefaultConfigPath = filepath.Join(homeDir, ".config", "uruflow", "agent.yaml")
			DefaultDataDir = filepath.Join(homeDir, ".local", "share", "uruflow-agent")
			DefaultPidFile = filepath.Join(homeDir, ".local", "share", "uruflow-agent", "agent.pid")
			DefaultLogFile = filepath.Join(homeDir, ".local", "share", "uruflow-agent", "agent.log")
		}
	}
}

type Config struct {
	Token   string       `yaml:"token"`
	DataDir string       `yaml:"data_dir"`
	PidFile string       `yaml:"pid_file"`
	LogFile string       `yaml:"log_file"`
	Server  ServerConfig `yaml:"server"`
	Docker  DockerConfig `yaml:"docker"`
}

type ServerConfig struct {
	Host          string `yaml:"host"`
	Port          int    `yaml:"port"`
	TLS           bool   `yaml:"tls"`
	TLSSkipVerify bool   `yaml:"tls_skip_verify"`
	ReconnectSec  int    `yaml:"reconnect_sec"`
	MetricsSec    int    `yaml:"metrics_sec"`
}

type DockerConfig struct {
	Enabled bool   `yaml:"enabled"`
	Socket  string `yaml:"socket"`
}

func Default() *Config {
	return &Config{
		Token:   "",
		DataDir: DefaultDataDir,
		PidFile: DefaultPidFile,
		LogFile: DefaultLogFile,
		Server: ServerConfig{
			Host:          "",
			Port:          9001,
			TLS:           false,
			TLSSkipVerify: false,
			ReconnectSec:  5,
			MetricsSec:    10,
		},
		Docker: DockerConfig{
			Enabled: true,
			Socket:  "/var/run/docker.sock",
		},
	}
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	if !Exists(path) {
		return nil, &ConfigError{Path: path, Err: os.ErrNotExist}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &ConfigError{Path: path, Err: err}
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, &ConfigError{Path: path, Err: err}
	}

	return cfg, nil
}

func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultConfigPath
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &ConfigError{Path: path, Err: err}
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func (c *Config) Validate() error {
	if c.Token == "" {
		return errors.New("token is required")
	}
	if c.Server.Host == "" {
		return errors.New("server.host is required")
	}
	return nil
}

func Exists(path string) bool {
	if path == "" {
		path = DefaultConfigPath
	}
	_, err := os.Stat(path)
	return err == nil
}

type ConfigError struct {
	Path string
	Err  error
}

func (e *ConfigError) Error() string {
	if os.IsNotExist(e.Err) {
		return "config not found at " + e.Path
	}
	if os.IsPermission(e.Err) {
		return "permission denied reading " + e.Path
	}
	return "config error: " + e.Err.Error()
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}
