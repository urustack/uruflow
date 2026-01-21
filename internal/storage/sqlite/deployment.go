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

func (s *Store) CreateDeployment(d *models.Deployment) error {
	_, err := s.db.Exec(`
		INSERT INTO deployments (id, repo_name, branch, commit_hash, agent_id, agent_name, status, trigger_type, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, d.ID, d.Repository, d.Branch, d.Commit, d.AgentID, d.AgentName, d.Status, d.Trigger, d.StartedAt)
	return err
}

func (s *Store) UpdateDeployment(d *models.Deployment) error {
	_, err := s.db.Exec(`
		UPDATE deployments SET status = ?, finished_at = ?, duration_ms = ?, output = ?
		WHERE id = ?
	`, d.Status, d.EndedAt, d.Duration, d.Output, d.ID)
	return err
}

func (s *Store) GetDeployment(id string) (*models.Deployment, error) {
	d := &models.Deployment{}
	var finishedAt sql.NullTime
	var duration sql.NullInt64
	var output sql.NullString

	err := s.db.QueryRow(`
		SELECT id, repo_name, branch, commit_hash, agent_id, agent_name, status, trigger_type, started_at, finished_at, duration_ms, output
		FROM deployments WHERE id = ?
	`, id).Scan(&d.ID, &d.Repository, &d.Branch, &d.Commit, &d.AgentID, &d.AgentName, &d.Status, &d.Trigger, &d.StartedAt, &finishedAt, &duration, &output)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if finishedAt.Valid {
		d.EndedAt = &finishedAt.Time
	}
	if duration.Valid {
		d.Duration = duration.Int64
	}
	if output.Valid {
		d.Output = output.String
	}

	return d, nil
}

func (s *Store) GetRecentDeployments(limit int) ([]models.Deployment, error) {
	rows, err := s.db.Query(`
		SELECT id, repo_name, branch, commit_hash, agent_id, agent_name, status, trigger_type, started_at, finished_at, duration_ms, output
		FROM deployments ORDER BY started_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDeployments(rows)
}

func (s *Store) GetDeploymentsByAgent(agentID string, limit int) ([]models.Deployment, error) {
	rows, err := s.db.Query(`
		SELECT id, repo_name, branch, commit_hash, agent_id, agent_name, status, trigger_type, started_at, finished_at, duration_ms, output
		FROM deployments WHERE agent_id = ? ORDER BY started_at DESC LIMIT ?
	`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDeployments(rows)
}

func (s *Store) GetDeploymentsByRepo(repoName string, limit int) ([]models.Deployment, error) {
	rows, err := s.db.Query(`
		SELECT id, repo_name, branch, commit_hash, agent_id, agent_name, status, trigger_type, started_at, finished_at, duration_ms, output
		FROM deployments WHERE repo_name = ? ORDER BY started_at DESC LIMIT ?
	`, repoName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDeployments(rows)
}

func scanDeployments(rows *sql.Rows) ([]models.Deployment, error) {
	var deployments []models.Deployment
	for rows.Next() {
		var d models.Deployment
		var finishedAt sql.NullTime
		var duration sql.NullInt64
		var output sql.NullString

		err := rows.Scan(&d.ID, &d.Repository, &d.Branch, &d.Commit, &d.AgentID, &d.AgentName, &d.Status, &d.Trigger, &d.StartedAt, &finishedAt, &duration, &output)
		if err != nil {
			return nil, err
		}

		if finishedAt.Valid {
			d.EndedAt = &finishedAt.Time
		}
		if duration.Valid {
			d.Duration = duration.Int64
		}
		if output.Valid {
			d.Output = output.String
		}

		deployments = append(deployments, d)
	}
	return deployments, nil
}
