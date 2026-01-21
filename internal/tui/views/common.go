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

package views

import "time"

type RefreshMsg struct{}
type TickMsg time.Time

type DataMsg struct {
	Agents      []AgentData
	Deployments []DeploymentData
	Alerts      []AlertData
	Repos       []RepoData
}

type AgentData struct {
	ID         string
	Name       string
	Host       string
	Version    string
	Uptime     string
	Online     bool
	CPU        float64
	Memory     float64
	Disk       float64
	Containers []ContainerData
}

type ContainerData struct {
	Name    string
	Running bool
	Healthy bool
	CPU     float64
	Memory  string
}

type DeploymentData struct {
	ID     string
	Repo   string
	Branch string
	Commit string
	Agent  string
	Status string
	Time   string
}

type RepoData struct {
	Name        string
	URL         string
	Branch      string
	Agent       string
	AgentID     string
	AutoDeploy  bool
	BuildSystem string
	BuildFile   string
	LastStatus  string
	LastCommit  string
	LastTime    string
}

type AlertData struct {
	ID       string
	Type     string
	Agent    string
	Message  string
	Time     string
	Active   bool
	Severity string
}

type LogData struct {
	Time    string
	Content string
	Stream  string
}
