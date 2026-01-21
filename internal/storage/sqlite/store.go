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

package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/urustack/uruflow/internal/storage"
)

type Store struct {
	db *sql.DB
}

func New(dataDir string) (storage.Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "uruflow-server.db")
	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(2)
	conn.SetConnMaxLifetime(time.Hour)

	store := &Store{db: conn}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) GetStats() (*storage.Stats, error) {
	stats := &storage.Stats{}

	s.db.QueryRow(`SELECT COUNT(*) FROM agents`).Scan(&stats.AgentsTotal)
	s.db.QueryRow(`SELECT COUNT(*) FROM agents WHERE status = 'online'`).Scan(&stats.AgentsOnline)
	s.db.QueryRow(`SELECT COUNT(*) FROM repositories`).Scan(&stats.ReposTotal)
	s.db.QueryRow(`SELECT COUNT(*) FROM deployments`).Scan(&stats.DeploymentsTotal)

	today := time.Now().Truncate(24 * time.Hour)
	s.db.QueryRow(`SELECT COUNT(*) FROM deployments WHERE started_at >= ?`, today).Scan(&stats.DeploymentsToday)

	var success, total int
	s.db.QueryRow(`SELECT COUNT(*) FROM deployments WHERE status = 'success'`).Scan(&success)
	s.db.QueryRow(`SELECT COUNT(*) FROM deployments WHERE status IN ('success', 'failed')`).Scan(&total)
	if total > 0 {
		stats.SuccessRate = float64(success) / float64(total) * 100
	}

	s.db.QueryRow(`SELECT COUNT(*) FROM containers WHERE status = 'running'`).Scan(&stats.ContainersRunning)
	s.db.QueryRow(`SELECT COUNT(*) FROM containers WHERE status != 'running'`).Scan(&stats.ContainersStopped)
	s.db.QueryRow(`SELECT COUNT(*) FROM alerts WHERE resolved = 0`).Scan(&stats.AlertsActive)

	return stats, nil
}
