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

	"github.com/urustack/uruflow/internal/models"
)

func (s *Store) UpsertContainer(c *models.Container) error {
	_, err := s.db.Exec(`
		INSERT INTO containers (id, agent_id, name, image, status, health, cpu_percent, memory_usage, memory_limit, network_rx, network_tx, restart_count, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			health = excluded.health,
			cpu_percent = excluded.cpu_percent,
			memory_usage = excluded.memory_usage,
			memory_limit = excluded.memory_limit,
			network_rx = excluded.network_rx,
			network_tx = excluded.network_tx,
			restart_count = excluded.restart_count
	`, c.ID, c.AgentID, c.Name, c.Image, c.Status, c.Health, c.CPUPercent, c.MemoryUsage, c.MemoryLimit, c.NetworkRx, c.NetworkTx, c.RestartCount, c.StartedAt)
	return err
}

func (s *Store) GetContainersByAgent(agentID string) ([]models.Container, error) {
	rows, err := s.db.Query(`
		SELECT id, agent_id, name, image, status, health, cpu_percent, memory_usage, memory_limit, network_rx, network_tx, restart_count, started_at
		FROM containers WHERE agent_id = ? ORDER BY name
	`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []models.Container
	for rows.Next() {
		var c models.Container
		var startedAt sql.NullTime
		err := rows.Scan(&c.ID, &c.AgentID, &c.Name, &c.Image, &c.Status, &c.Health, &c.CPUPercent, &c.MemoryUsage, &c.MemoryLimit, &c.NetworkRx, &c.NetworkTx, &c.RestartCount, &startedAt)
		if err != nil {
			return nil, err
		}
		if startedAt.Valid {
			c.StartedAt = startedAt.Time
		}
		containers = append(containers, c)
	}
	return containers, nil
}

func (s *Store) DeleteContainersByAgent(agentID string) error {
	_, err := s.db.Exec(`DELETE FROM containers WHERE agent_id = ?`, agentID)
	return err
}
