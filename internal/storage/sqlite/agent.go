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

func (s *Store) CreateAgent(agent *models.Agent) error {
	_, err := s.db.Exec(`
		INSERT INTO agents (id, name, token, host, hostname, version, status, last_heartbeat, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, agent.ID, agent.Name, agent.Token, agent.Host, agent.Hostname, agent.Version, agent.Status, agent.LastHeartbeat, time.Now())
	return err
}

func (s *Store) UpdateAgent(agent *models.Agent) error {
	_, err := s.db.Exec(`
		UPDATE agents SET
			host = COALESCE(NULLIF(?, ''), host),
			hostname = COALESCE(NULLIF(?, ''), hostname),
			version = COALESCE(NULLIF(?, ''), version),
			status = ?,
			last_heartbeat = ?
		WHERE id = ?
	`, agent.Host, agent.Hostname, agent.Version, agent.Status, agent.LastHeartbeat, agent.ID)
	return err
}

func (s *Store) UpdateAgentMetrics(id string, metrics *models.AgentMetrics) error {
	_, err := s.db.Exec(`
		UPDATE agents SET
			cpu_percent = ?,
			memory_percent = ?,
			disk_percent = ?,
			memory_used = ?,
			memory_total = ?,
			disk_used = ?,
			disk_total = ?,
			uptime = ?,
			status = 'online',
			last_heartbeat = ?
		WHERE id = ?
	`, metrics.CPUPercent, metrics.MemoryPercent, metrics.DiskPercent,
		metrics.MemoryUsed, metrics.MemoryTotal, metrics.DiskUsed, metrics.DiskTotal,
		metrics.Uptime, time.Now(), id)
	return err
}

func (s *Store) UpdateAgentStatus(id string, status models.AgentStatus) error {
	_, err := s.db.Exec(`UPDATE agents SET status = ?, last_heartbeat = ? WHERE id = ?`, status, time.Now(), id)
	return err
}

func (s *Store) GetAgent(id string) (*models.Agent, error) {
	agent := &models.Agent{}
	var lastHeartbeat, createdAt sql.NullTime
	var cpu, mem, disk float64
	var memUsed, memTotal, diskUsed, diskTotal uint64
	var uptime int64

	err := s.db.QueryRow(`
		SELECT id, name, token, host, hostname, version, status,
			cpu_percent, memory_percent, disk_percent,
			memory_used, memory_total, disk_used, disk_total, uptime,
			last_heartbeat, created_at
		FROM agents WHERE id = ?
	`, id).Scan(
		&agent.ID, &agent.Name, &agent.Token, &agent.Host, &agent.Hostname, &agent.Version, &agent.Status,
		&cpu, &mem, &disk,
		&memUsed, &memTotal, &diskUsed, &diskTotal, &uptime,
		&lastHeartbeat, &createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastHeartbeat.Valid {
		agent.LastHeartbeat = lastHeartbeat.Time
	}
	if createdAt.Valid {
		agent.RegisteredAt = createdAt.Time
	}

	agent.Metrics = &models.AgentMetrics{
		CPUPercent:    cpu,
		MemoryPercent: mem,
		DiskPercent:   disk,
		MemoryUsed:    memUsed,
		MemoryTotal:   memTotal,
		DiskUsed:      diskUsed,
		DiskTotal:     diskTotal,
		Uptime:        uptime,
	}

	return agent, nil
}

func (s *Store) GetAgentByToken(token string) (*models.Agent, error) {
	agent := &models.Agent{}
	err := s.db.QueryRow(`
		SELECT id, name, token, host, hostname, version, status
		FROM agents WHERE token = ?
	`, token).Scan(&agent.ID, &agent.Name, &agent.Token, &agent.Host, &agent.Hostname, &agent.Version, &agent.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return agent, err
}

func (s *Store) GetAllAgents() ([]models.Agent, error) {
	rows, err := s.db.Query(`
		SELECT id, name, token, host, hostname, version, status,
			cpu_percent, memory_percent, disk_percent,
			memory_used, memory_total, disk_used, disk_total, uptime,
			last_heartbeat, created_at
		FROM agents ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var a models.Agent
		var lastHeartbeat, createdAt sql.NullTime
		var cpu, mem, disk float64
		var memUsed, memTotal, diskUsed, diskTotal uint64
		var uptime int64

		err := rows.Scan(
			&a.ID, &a.Name, &a.Token, &a.Host, &a.Hostname, &a.Version, &a.Status,
			&cpu, &mem, &disk,
			&memUsed, &memTotal, &diskUsed, &diskTotal, &uptime,
			&lastHeartbeat, &createdAt,
		)
		if err != nil {
			return nil, err
		}

		if lastHeartbeat.Valid {
			a.LastHeartbeat = lastHeartbeat.Time
		}
		if createdAt.Valid {
			a.RegisteredAt = createdAt.Time
		}

		a.Metrics = &models.AgentMetrics{
			CPUPercent:    cpu,
			MemoryPercent: mem,
			DiskPercent:   disk,
			MemoryUsed:    memUsed,
			MemoryTotal:   memTotal,
			DiskUsed:      diskUsed,
			DiskTotal:     diskTotal,
			Uptime:        uptime,
		}

		agents = append(agents, a)
	}

	return agents, nil
}

func (s *Store) DeleteAgent(id string) error {
	_, err := s.db.Exec(`DELETE FROM agents WHERE id = ?`, id)
	return err
}
