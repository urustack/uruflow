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

package tui

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urustack/uruflow/internal/api"
	"github.com/urustack/uruflow/internal/config"
	"github.com/urustack/uruflow/internal/storage"
)

func Run(store storage.Store, cfg *config.Config, server *api.Server) error {
	f, err := tea.LogToFile("uruflow-server.log", "debug")
	if err != nil {
		return fmt.Errorf("fatal: could not open log file: %w", err)
	}
	defer f.Close()

	log.SetOutput(f)

	cfgPath := config.DefaultConfigPath
	model := NewModel(store, cfg, cfgPath, server)
	p := tea.NewProgram(&model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui error: %w", err)
	}

	return nil
}

func RunInit() error {
	f, err := tea.LogToFile("uruflow-server-init.log", "debug")
	if err != nil {
		return fmt.Errorf("fatal: could not open init log file: %w", err)
	}
	defer f.Close()
	log.SetOutput(f)

	model := NewInitModel()
	p := tea.NewProgram(&model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui error: %w", err)
	}

	return nil
}
