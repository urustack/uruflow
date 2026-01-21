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

package models

import "time"

type AgentStatus string

const (
	AgentOnline  AgentStatus = "online"
	AgentOffline AgentStatus = "offline"
)

type ContainerHealth string

type DeployStatus string

const (
	DeployPending DeployStatus = "pending"
	DeployRunning DeployStatus = "running"
	DeploySuccess DeployStatus = "success"
	DeployFailed  DeployStatus = "failed"
)

type AlertSeverity string

const (
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

type BuildSystem string

type Agent struct {
	ID            string        `json:"id" yaml:"id"`
	Name          string        `json:"name" yaml:"name"`
	Token         string        `json:"-" yaml:"token"`
	Host          string        `json:"host" yaml:"host"`
	Hostname      string        `json:"hostname" yaml:"hostname"`
	Version       string        `json:"version" yaml:"version"`
	Status        AgentStatus   `json:"status" yaml:"status"`
	LastHeartbeat time.Time     `json:"last_heartbeat" yaml:"last_heartbeat"`
	Metrics       *AgentMetrics `json:"metrics,omitempty" yaml:"metrics,omitempty"`
	Containers    []Container   `json:"containers,omitempty" yaml:"containers,omitempty"`
	RegisteredAt  time.Time     `json:"registered_at" yaml:"registered_at"`
}

type AgentMetrics struct {
	CPUPercent    float64   `json:"cpu_percent" yaml:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent" yaml:"memory_percent"`
	MemoryUsed    uint64    `json:"memory_used" yaml:"memory_used"`
	MemoryTotal   uint64    `json:"memory_total" yaml:"memory_total"`
	DiskPercent   float64   `json:"disk_percent" yaml:"disk_percent"`
	DiskUsed      uint64    `json:"disk_used" yaml:"disk_used"`
	DiskTotal     uint64    `json:"disk_total" yaml:"disk_total"`
	LoadAvg       []float64 `json:"load_avg" yaml:"load_avg"`
	Uptime        int64     `json:"uptime" yaml:"uptime"`
}

type Container struct {
	ID           string          `json:"id" yaml:"id"`
	AgentID      string          `json:"agent_id" yaml:"agent_id"`
	Name         string          `json:"name" yaml:"name"`
	Image        string          `json:"image" yaml:"image"`
	Status       string          `json:"status" yaml:"status"`
	Health       ContainerHealth `json:"health" yaml:"health"`
	CPUPercent   float64         `json:"cpu_percent" yaml:"cpu_percent"`
	MemoryUsage  uint64          `json:"memory_usage" yaml:"memory_usage"`
	MemoryLimit  uint64          `json:"memory_limit" yaml:"memory_limit"`
	NetworkRx    uint64          `json:"network_rx" yaml:"network_rx"`
	NetworkTx    uint64          `json:"network_tx" yaml:"network_tx"`
	RestartCount int             `json:"restart_count" yaml:"restart_count"`
	StartedAt    time.Time       `json:"started_at" yaml:"started_at"`
}

type Repository struct {
	ID          int64       `json:"id" yaml:"id"`
	Name        string      `json:"name" yaml:"name"`
	URL         string      `json:"url" yaml:"url"`
	Branch      string      `json:"branch" yaml:"branch"`
	AgentID     string      `json:"agent_id" yaml:"agent_id"`
	Path        string      `json:"path" yaml:"path"`
	AutoDeploy  bool        `json:"auto_deploy" yaml:"auto_deploy"`
	BuildSystem BuildSystem `json:"build_system" yaml:"build_system"`
	BuildFile   string      `json:"build_file" yaml:"build_file"`
	BuildCmd    string      `json:"build_cmd" yaml:"build_cmd"`
	CreatedAt   time.Time   `json:"created_at" yaml:"created_at"`
}

type Command struct {
	ID        string                 `json:"id" yaml:"id"`
	Type      string                 `json:"type" yaml:"type"`
	AgentID   string                 `json:"agent_id" yaml:"agent_id"`
	Payload   map[string]interface{} `json:"payload" yaml:"payload"`
	Status    DeployStatus           `json:"status" yaml:"status"`
	Output    string                 `json:"output,omitempty" yaml:"output,omitempty"`
	CreatedAt time.Time              `json:"created_at" yaml:"created_at"`
	StartedAt *time.Time             `json:"started_at,omitempty" yaml:"started_at,omitempty"`
	EndedAt   *time.Time             `json:"ended_at,omitempty" yaml:"ended_at,omitempty"`
}

type Alert struct {
	ID         string        `json:"id" yaml:"id"`
	AgentID    string        `json:"agent_id" yaml:"agent_id"`
	AgentName  string        `json:"agent_name" yaml:"agent_name"`
	Type       string        `json:"type" yaml:"type"`
	Message    string        `json:"message" yaml:"message"`
	Severity   AlertSeverity `json:"severity" yaml:"severity"`
	Resolved   bool          `json:"resolved" yaml:"resolved"`
	CreatedAt  time.Time     `json:"created_at" yaml:"created_at"`
	ResolvedAt *time.Time    `json:"resolved_at,omitempty" yaml:"resolved_at,omitempty"`
}

type Deployment struct {
	ID         string       `json:"id" yaml:"id"`
	Repository string       `json:"repository" yaml:"repository"`
	Branch     string       `json:"branch" yaml:"branch"`
	Commit     string       `json:"commit" yaml:"commit"`
	AgentID    string       `json:"agent_id" yaml:"agent_id"`
	AgentName  string       `json:"agent_name" yaml:"agent_name"`
	Status     DeployStatus `json:"status" yaml:"status"`
	Output     string       `json:"output,omitempty" yaml:"output,omitempty"`
	Duration   int64        `json:"duration" yaml:"duration"`
	StartedAt  time.Time    `json:"started_at" yaml:"started_at"`
	EndedAt    *time.Time   `json:"ended_at,omitempty" yaml:"ended_at,omitempty"`
	Trigger    string       `json:"trigger" yaml:"trigger"`
}

type DeploymentLog struct {
	ID           int64     `json:"id"`
	DeploymentID string    `json:"deployment_id"`
	Line         string    `json:"line"`
	Stream       string    `json:"stream"`
	Timestamp    time.Time `json:"timestamp"`
}

type CommandLog struct {
	CommandID string    `json:"command_id"`
	Line      string    `json:"line"`
	Stream    string    `json:"stream"`
	Timestamp time.Time `json:"timestamp"`
}
