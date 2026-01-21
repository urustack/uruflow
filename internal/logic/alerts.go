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

package logic

import (
	"time"

	"github.com/urustack/uruflow/internal/models"
	"github.com/urustack/uruflow/pkg/helper"
)

func CheckCPU(agentID, agentName string, cpuPercent float64) *models.Alert {
	if cpuPercent > 90 {
		return newAlert(agentID, agentName, "high_cpu", "CPU usage above 90%", models.SeverityCritical)
	}
	if cpuPercent > 80 {
		return newAlert(agentID, agentName, "high_cpu", "CPU usage above 80%", models.SeverityWarning)
	}
	return nil
}

func CheckMemory(agentID, agentName string, memPercent float64) *models.Alert {
	if memPercent > 95 {
		return newAlert(agentID, agentName, "high_memory", "Memory usage above 95%", models.SeverityCritical)
	}
	if memPercent > 90 {
		return newAlert(agentID, agentName, "high_memory", "Memory usage above 90%", models.SeverityWarning)
	}
	return nil
}

func CheckDisk(agentID, agentName string, diskPercent float64) *models.Alert {
	if diskPercent > 95 {
		return newAlert(agentID, agentName, "high_disk", "Disk usage above 95%", models.SeverityCritical)
	}
	if diskPercent > 85 {
		return newAlert(agentID, agentName, "high_disk", "Disk usage above 85%", models.SeverityWarning)
	}
	return nil
}

func CheckContainerDown(agentID, agentName, containerName string) *models.Alert {
	return newAlert(
		agentID,
		agentName,
		"container_down",
		"Container "+containerName+" is not running",
		models.SeverityCritical,
	)
}

func CheckOffline(agentID, agentName string) *models.Alert {
	return newAlert(
		agentID,
		agentName,
		"agent_offline",
		"Agent "+agentName+" is offline",
		models.SeverityCritical,
	)
}

func newAlert(agentID, agentName, alertType, msg string, severity models.AlertSeverity) *models.Alert {
	return &models.Alert{
		ID:        helper.GenerateID(),
		AgentID:   agentID,
		AgentName: agentName,
		Type:      alertType,
		Message:   msg,
		Severity:  severity,
		Resolved:  false,
		CreatedAt: time.Now(),
	}
}
