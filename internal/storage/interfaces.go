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

package storage

import "github.com/urustack/uruflow/internal/models"

type Store interface {
	CreateAgent(agent *models.Agent) error
	UpdateAgent(agent *models.Agent) error
	UpdateAgentMetrics(id string, metrics *models.AgentMetrics) error
	UpdateAgentStatus(id string, status models.AgentStatus) error
	GetAgent(id string) (*models.Agent, error)
	GetAgentByToken(token string) (*models.Agent, error)
	GetAllAgents() ([]models.Agent, error)
	DeleteAgent(id string) error

	UpsertContainer(c *models.Container) error
	GetContainersByAgent(agentID string) ([]models.Container, error)
	DeleteContainersByAgent(agentID string) error

	CreateRepository(repo *models.Repository) error
	UpdateRepository(repo *models.Repository) error
	GetRepository(name string) (*models.Repository, error)
	GetAllRepositories() ([]models.Repository, error)
	DeleteRepository(name string) error

	CreateDeployment(d *models.Deployment) error
	UpdateDeployment(d *models.Deployment) error
	GetDeployment(id string) (*models.Deployment, error)
	GetRecentDeployments(limit int) ([]models.Deployment, error)
	GetDeploymentsByAgent(agentID string, limit int) ([]models.Deployment, error)
	GetDeploymentsByRepo(repoName string, limit int) ([]models.Deployment, error)

	AddDeploymentLog(log *models.DeploymentLog) error
	GetDeploymentLogs(deploymentID string) ([]models.DeploymentLog, error)

	CreateAlert(a *models.Alert) error
	ResolveAlert(id string) error
	GetActiveAlerts() ([]models.Alert, error)
	GetRecentAlerts(hours int) ([]models.Alert, error)
	GetAlertsByAgent(agentID string) ([]models.Alert, error)

	GetStats() (*Stats, error)
	Close() error
}

type Stats struct {
	AgentsTotal       int
	AgentsOnline      int
	ReposTotal        int
	DeploymentsTotal  int
	DeploymentsToday  int
	SuccessRate       float64
	ContainersRunning int
	ContainersStopped int
	AlertsActive      int
}
