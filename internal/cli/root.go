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

package cli

import (
	"context"
	"fmt"
	"github.com/urustack/uruflow/pkg/logger"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/urustack/uruflow/internal/api"
	"github.com/urustack/uruflow/internal/config"
	"github.com/urustack/uruflow/internal/storage/sqlite"
	"github.com/urustack/uruflow/internal/tui"
	"github.com/urustack/uruflow/pkg/helper"
)

var (
	cfgPath string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "uruflow",
	Short: "UruFlow - Deployment Automation Server",
	Long:  `UruFlow is a lightweight, self-hosted deployment system.`,
	Run:   runApplication,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "config file path")
}

func initConfig() {
	if cfgPath == "" {
		cfgPath = os.Getenv("URUFLOW_CONFIG")
	}
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath
	}

	if helper.Exists(cfgPath) {
		var err error
		cfg, err = config.Load(cfgPath)
		if err != nil {
			logger.Warn("failed to load config from %s: %v", cfgPath, err)
		} else {
			logger.Info("config loaded from %s", cfgPath)
		}
	}
}

func runApplication(cmd *cobra.Command, args []string) {
	if cfg == nil {
		logger.Info("No config found, running initialization")
		if err := tui.RunInit(); err != nil {
			logger.Error("Initialization failed: %v", err)
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		var err error
		cfg, err = config.Load(cfgPath)
		if err != nil {
			logger.Error("Failed to load config after init: %v", err)
			fmt.Printf("Error loading config after init: %v\n", err)
			os.Exit(1)
		}
	}

	logger.Info("Initializing database at %s", cfg.Server.DataDir)
	store, err := sqlite.New(cfg.Server.DataDir)
	if err != nil {
		logger.Error("Database initialization failed: %v", err)
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	logger.Info("Starting API server")
	server := api.NewServer(cfg, store)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
		os.Exit(0)
	}()

	go func() {
		if err := server.Start(); err != nil {
			logger.Error("Server error: %v", err)
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	logger.Info("Starting TUI")
	if err := tui.Run(store, cfg, server); err != nil {
		logger.Error("TUI error: %v", err)
		fmt.Printf("TUI Error: %v\n", err)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
		os.Exit(1)
	}

	logger.Info("Shutting down gracefully")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)
}
