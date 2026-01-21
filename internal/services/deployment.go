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

package services

import (
	"fmt"
	"time"

	"github.com/urustack/uruflow/internal/config"
	"github.com/urustack/uruflow/internal/models"
	"github.com/urustack/uruflow/internal/storage"
	"github.com/urustack/uruflow/internal/tcp"
	"github.com/urustack/uruflow/pkg/helper"
)

type DeploymentService struct {
	cfg       *config.Config
	store     storage.Store
	tcpServer *tcp.Server
}

func NewDeploymentService(cfg *config.Config, store storage.Store, tcpServer *tcp.Server) *DeploymentService {
	return &DeploymentService{
		cfg:       cfg,
		store:     store,
		tcpServer: tcpServer,
	}
}

func (s *DeploymentService) TriggerDeploy(agentID, repoName, branch, commit, trigger string) (*models.Deployment, error) {
	if !s.tcpServer.IsAgentConnected(agentID) {
		return nil, ErrAgentNotConnected
	}

	repo := s.cfg.GetRepository(repoName)
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	agent, err := s.store.GetAgent(agentID)
	agentName := "unknown"
	if err == nil && agent != nil {
		agentName = agent.Name
	}

	deploy := &models.Deployment{
		ID:         helper.GenerateID(),
		Repository: repoName,
		Branch:     branch,
		Commit:     commit,
		AgentID:    agentID,
		AgentName:  agentName,
		Status:     models.DeployPending,
		StartedAt:  time.Now(),
		Trigger:    trigger,
	}

	if err := s.store.CreateDeployment(deploy); err != nil {
		return nil, err
	}

	cmd := &models.Command{
		ID:      deploy.ID,
		Type:    "deploy",
		AgentID: agentID,
		Payload: map[string]interface{}{
			"url":          repo.URL,
			"name":         repo.Name,
			"branch":       branch,
			"commit":       commit,
			"path":         repo.Path,
			"build_system": string(repo.BuildSystem),
			"build_file":   repo.BuildFile,
			"build_cmd":    repo.BuildCmd,
		},
	}

	if err := s.tcpServer.SendCommand(agentID, cmd); err != nil {
		deploy.Status = models.DeployFailed
		deploy.Output = fmt.Sprintf("Failed to send command: %v", err)
		deploy.EndedAt = &deploy.StartedAt

		_ = s.store.UpdateDeployment(deploy)
		return nil, err
	}

	return deploy, nil
}

func (s *DeploymentService) GetRecent(limit int) ([]models.Deployment, error) {
	return s.store.GetRecentDeployments(limit)
}

func (s *DeploymentService) GetByRepo(repoName string, limit int) ([]models.Deployment, error) {
	return s.store.GetDeploymentsByRepo(repoName, limit)
}

func (s *DeploymentService) GetLogs(deployID string) ([]models.DeploymentLog, error) {
	return s.store.GetDeploymentLogs(deployID)
}
