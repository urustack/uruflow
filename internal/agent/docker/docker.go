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

package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	client *http.Client
	socket string
}

type Container struct {
	ID           string
	FullID       string
	Name         string
	Image        string
	Status       string
	State        string
	Health       string
	CPUPercent   float64
	MemoryUsage  uint64
	MemoryLimit  uint64
	NetworkRx    uint64
	NetworkTx    uint64
	RestartCount int
	StartedAt    int64
	IsManaged    bool
}

func New(socket string) (*Service, error) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", socket)
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	resp, err := client.Get("http://localhost/version")
	if err != nil {
		return nil, fmt.Errorf("docker not available: %w", err)
	}
	resp.Body.Close()

	return &Service{
		client: client,
		socket: socket,
	}, nil
}

func (s *Service) ListContainers(ctx context.Context) ([]Container, error) {
	resp, err := s.client.Get("http://localhost/containers/json?all=true")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var containers []struct {
		ID      string            `json:"Id"`
		Names   []string          `json:"Names"`
		Image   string            `json:"Image"`
		Status  string            `json:"Status"`
		State   string            `json:"State"`
		Created int64             `json:"Created"`
		Labels  map[string]string `json:"Labels"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, err
	}

	result := make([]Container, 0, len(containers))

	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		isManaged := s.checkManagedFromLabels(c.Labels)

		health := "none"
		restartCount := 0
		var startedAt int64

		inspect, err := s.inspectContainer(c.ID)
		if err == nil {
			if inspect.State.Health != nil {
				health = inspect.State.Health.Status
			}
			restartCount = inspect.RestartCount
			if inspect.State.StartedAt != "" {
				t, _ := time.Parse(time.RFC3339Nano, inspect.State.StartedAt)
				startedAt = t.Unix()
			}
		}

		result = append(result, Container{
			ID:           c.ID[:12],
			FullID:       c.ID,
			Name:         name,
			Image:        c.Image,
			Status:       c.Status,
			State:        c.State,
			Health:       health,
			RestartCount: restartCount,
			StartedAt:    startedAt,
			IsManaged:    isManaged,
		})
	}

	return result, nil
}

func (s *Service) ListManagedContainers(ctx context.Context) ([]Container, error) {
	all, err := s.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	managed := make([]Container, 0)
	for _, c := range all {
		if c.IsManaged {
			managed = append(managed, c)
		}
	}

	return managed, nil
}

func (s *Service) checkManagedFromLabels(labels map[string]string) bool {
	if labels == nil {
		return false
	}

	if managed, exists := labels["io.uruflow.managed"]; exists && managed == "true" {
		return true
	}

	if project, exists := labels["com.docker.compose.project"]; exists {
		if strings.HasPrefix(project, "uruflow-") {
			return true
		}
	}

	return false
}

func (s *Service) IsUruflowManaged(ctx context.Context, containerID string) (bool, error) {
	resp, err := s.client.Get(fmt.Sprintf("http://localhost/containers/%s/json", containerID))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var inspect struct {
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&inspect); err != nil {
		return false, err
	}

	return s.checkManagedFromLabels(inspect.Config.Labels), nil
}

func (s *Service) GetContainerStats(ctx context.Context, containerID string) (*Container, error) {
	resp, err := s.client.Get(fmt.Sprintf("http://localhost/containers/%s/stats?stream=false", containerID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs  int    `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
		} `json:"memory_stats"`
		Networks map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		} `json:"networks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	cpuPercent := 0.0
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(stats.CPUStats.OnlineCPUs) * 100.0
	}

	var rxBytes, txBytes uint64
	for _, net := range stats.Networks {
		rxBytes += net.RxBytes
		txBytes += net.TxBytes
	}

	return &Container{
		CPUPercent:  cpuPercent,
		MemoryUsage: stats.MemoryStats.Usage,
		MemoryLimit: stats.MemoryStats.Limit,
		NetworkRx:   rxBytes,
		NetworkTx:   txBytes,
	}, nil
}

type inspectResult struct {
	State struct {
		Health *struct {
			Status string `json:"Status"`
		} `json:"Health"`
		StartedAt string `json:"StartedAt"`
	} `json:"State"`
	RestartCount int `json:"RestartCount"`
}

func (s *Service) inspectContainer(id string) (*inspectResult, error) {
	resp, err := s.client.Get(fmt.Sprintf("http://localhost/containers/%s/json", id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result inspectResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *Service) StreamLogs(ctx context.Context, containerID string, onLine func(string)) error {
	return s.StreamLogsWithTail(ctx, containerID, 100, onLine)
}

func (s *Service) StreamLogsWithTail(ctx context.Context, containerID string, tail int, onLine func(string)) error {
	url := fmt.Sprintf("http://localhost/containers/%s/logs?stdout=true&stderr=true&follow=true&timestamps=true&tail=%d", containerID, tail)
	resp, err := s.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			header := make([]byte, 8)
			_, err := io.ReadFull(reader, header)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				line, readErr := reader.ReadString('\n')
				if readErr != nil {
					return err
				}
				if onLine != nil && strings.TrimSpace(line) != "" {
					onLine(strings.TrimSpace(line))
				}
				continue
			}

			streamType := "stdout"
			if header[0] == 2 {
				streamType = "stderr"
			}

			size := int(header[4])<<24 | int(header[5])<<16 | int(header[6])<<8 | int(header[7])
			if size > 0 && size < 1024*1024 {
				content := make([]byte, size)
				_, err := io.ReadFull(reader, content)
				if err != nil {
					return err
				}
				line := strings.TrimSpace(string(content))
				if onLine != nil && line != "" {
					if streamType == "stderr" {
						line = "[stderr] " + line
					}
					onLine(line)
				}
			}
		}
	}
}
