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
	"time"

	"github.com/urustack/uruflow/internal/models"
)

func (s *Store) CreateAlert(a *models.Alert) error {
	_, err := s.db.Exec(`
		INSERT INTO alerts (id, type, severity, agent_id, agent_name, message, resolved, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.Type, a.Severity, a.AgentID, a.AgentName, a.Message, a.Resolved, a.CreatedAt)
	return err
}

func (s *Store) ResolveAlert(id string) error {
	_, err := s.db.Exec(`
		UPDATE alerts SET resolved = 1, resolved_at = ? WHERE id = ?
	`, time.Now(), id)
	return err
}

func (s *Store) GetActiveAlerts() ([]models.Alert, error) {
	rows, err := s.db.Query(`
		SELECT id, type, severity, agent_id, agent_name, message, resolved, created_at, resolved_at
		FROM alerts WHERE resolved = 0 ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAlerts(rows)
}

func (s *Store) GetRecentAlerts(hours int) ([]models.Alert, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	rows, err := s.db.Query(`
		SELECT id, type, severity, agent_id, agent_name, message, resolved, created_at, resolved_at
		FROM alerts WHERE created_at > ? ORDER BY created_at DESC
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAlerts(rows)
}

func (s *Store) GetAlertsByAgent(agentID string) ([]models.Alert, error) {
	rows, err := s.db.Query(`
		SELECT id, type, severity, agent_id, agent_name, message, resolved, created_at, resolved_at
		FROM alerts WHERE agent_id = ? ORDER BY created_at DESC
	`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAlerts(rows)
}

func scanAlerts(rows *sql.Rows) ([]models.Alert, error) {
	var alerts []models.Alert
	for rows.Next() {
		var a models.Alert
		var resolvedAt sql.NullTime
		err := rows.Scan(&a.ID, &a.Type, &a.Severity, &a.AgentID, &a.AgentName, &a.Message, &a.Resolved, &a.CreatedAt, &resolvedAt)
		if err != nil {
			return nil, err
		}
		if resolvedAt.Valid {
			a.ResolvedAt = &resolvedAt.Time
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}
