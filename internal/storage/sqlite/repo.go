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

func (s *Store) CreateRepository(repo *models.Repository) error {
	result, err := s.db.Exec(`
		INSERT INTO repositories (name, url, branch, agent_id, path, auto_deploy)
		VALUES (?, ?, ?, ?, ?, ?)
	`, repo.Name, repo.URL, repo.Branch, repo.AgentID, repo.Path, repo.AutoDeploy)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	repo.ID = id
	return nil
}

func (s *Store) UpdateRepository(repo *models.Repository) error {
	_, err := s.db.Exec(`
		UPDATE repositories SET
			url = ?, branch = ?, agent_id = ?, path = ?, auto_deploy = ?, updated_at = ?
		WHERE name = ?
	`, repo.URL, repo.Branch, repo.AgentID, repo.Path, repo.AutoDeploy, time.Now(), repo.Name)
	return err
}

func (s *Store) GetRepository(name string) (*models.Repository, error) {
	repo := &models.Repository{}
	var createdAt sql.NullTime
	err := s.db.QueryRow(`
		SELECT id, name, url, branch, agent_id, path, auto_deploy, created_at
		FROM repositories WHERE name = ?
	`, name).Scan(&repo.ID, &repo.Name, &repo.URL, &repo.Branch, &repo.AgentID, &repo.Path, &repo.AutoDeploy, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if createdAt.Valid {
		repo.CreatedAt = createdAt.Time
	}
	return repo, err
}

func (s *Store) GetAllRepositories() ([]models.Repository, error) {
	rows, err := s.db.Query(`
		SELECT id, name, url, branch, agent_id, path, auto_deploy, created_at
		FROM repositories ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []models.Repository
	for rows.Next() {
		var r models.Repository
		var createdAt sql.NullTime
		err := rows.Scan(&r.ID, &r.Name, &r.URL, &r.Branch, &r.AgentID, &r.Path, &r.AutoDeploy, &createdAt)
		if err != nil {
			return nil, err
		}
		if createdAt.Valid {
			r.CreatedAt = createdAt.Time
		}
		repos = append(repos, r)
	}
	return repos, nil
}

func (s *Store) DeleteRepository(name string) error {
	_, err := s.db.Exec(`DELETE FROM repositories WHERE name = ?`, name)
	return err
}
