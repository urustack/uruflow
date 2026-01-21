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
	"github.com/urustack/uruflow/internal/models"
)

func (s *Store) AddDeploymentLog(log *models.DeploymentLog) error {
	_, err := s.db.Exec(`
		INSERT INTO deployment_logs (deployment_id, timestamp, stream, content)
		VALUES (?, ?, ?, ?)
	`, log.DeploymentID, log.Timestamp, log.Stream, log.Line)
	return err
}

func (s *Store) GetDeploymentLogs(deploymentID string) ([]models.DeploymentLog, error) {
	rows, err := s.db.Query(`
		SELECT id, deployment_id, timestamp, stream, content
		FROM deployment_logs WHERE deployment_id = ? ORDER BY id
	`, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.DeploymentLog
	for rows.Next() {
		var l models.DeploymentLog
		err := rows.Scan(&l.ID, &l.DeploymentID, &l.Timestamp, &l.Stream, &l.Line)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (s *Store) DeleteDeploymentLogs(deploymentID string) error {
	_, err := s.db.Exec(`DELETE FROM deployment_logs WHERE deployment_id = ?`, deploymentID)
	return err
}
