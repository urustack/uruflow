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
			fmt.Printf("Warning: Failed to load config: %v\n", err)
		}
	}
}

func runApplication(cmd *cobra.Command, args []string) {
	if cfg == nil {
		if err := tui.RunInit(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		var err error
		cfg, err = config.Load(cfgPath)
		if err != nil {
			fmt.Printf("Error loading config after init: %v\n", err)
			os.Exit(1)
		}
	}

	store, err := sqlite.New(cfg.Server.DataDir)
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	server := api.NewServer(cfg, store)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
		os.Exit(0)
	}()

	go func() {
		if err := server.Start(); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	if err := tui.Run(store, cfg, server); err != nil {
		fmt.Printf("TUI Error: %v\n", err)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
		os.Exit(1)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)
}
