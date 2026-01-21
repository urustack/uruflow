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

package protocol

import (
	"encoding/json"
)

type Message struct {
	Type    MessageType
	Payload []byte
}

func NewMessage(msgType MessageType, payload interface{}) (*Message, error) {
	var data []byte
	var err error

	if payload != nil {
		data, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	return &Message{
		Type:    msgType,
		Payload: data,
	}, nil
}

func (m *Message) Decode(v interface{}) error {
	if len(m.Payload) == 0 {
		return nil
	}
	return json.Unmarshal(m.Payload, v)
}

func (m *Message) Encode() []byte {
	header := EncodeHeader(m.Type, uint32(len(m.Payload)))
	result := make([]byte, HeaderSize+len(m.Payload))
	copy(result[:HeaderSize], header)
	copy(result[HeaderSize:], m.Payload)
	return result
}

type AuthPayload struct {
	Token    string `json:"token"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	Version  string `json:"version"`
}

type AuthOKPayload struct {
	AgentID       string `json:"agent_id"`
	Name          string `json:"name"`
	ServerVersion string `json:"server_version"`
}

type AuthFailPayload struct {
	Reason string `json:"reason"`
}

type MetricsPayload struct {
	Timestamp  int64         `json:"timestamp"`
	System     SystemMetrics `json:"system"`
	Containers []Container   `json:"containers"`
}

type SystemMetrics struct {
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	MemoryUsed    uint64    `json:"memory_used"`
	MemoryTotal   uint64    `json:"memory_total"`
	DiskPercent   float64   `json:"disk_percent"`
	DiskUsed      uint64    `json:"disk_used"`
	DiskTotal     uint64    `json:"disk_total"`
	LoadAvg       []float64 `json:"load_avg"`
	Uptime        int64     `json:"uptime"`
}

type Container struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Image        string  `json:"image"`
	Status       string  `json:"status"`
	Health       string  `json:"health"`
	CPUPercent   float64 `json:"cpu_percent"`
	MemoryUsage  uint64  `json:"memory_usage"`
	MemoryLimit  uint64  `json:"memory_limit"`
	NetworkRx    uint64  `json:"network_rx"`
	NetworkTx    uint64  `json:"network_tx"`
	RestartCount int     `json:"restart_count"`
	StartedAt    int64   `json:"started_at"`
}

type CommandPayload struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

type CommandAckPayload struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
}

type CommandStartPayload struct {
	CommandID string `json:"command_id"`
	StartedAt int64  `json:"started_at"`
}

type CommandLogPayload struct {
	CommandID string `json:"command_id"`
	Line      string `json:"line"`
	Stream    string `json:"stream"`
	Timestamp int64  `json:"timestamp"`
}

type CommandDonePayload struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
	ExitCode  int    `json:"exit_code"`
	Duration  int64  `json:"duration"`
	Output    string `json:"output"`
}

type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func Ping() *Message {
	return &Message{Type: TypePing}
}

func Pong() *Message {
	return &Message{Type: TypePong}
}

func Disconnect() *Message {
	return &Message{Type: TypeDisconnect}
}

func Error(code int, message string) (*Message, error) {
	return NewMessage(TypeError, ErrorPayload{
		Code:    code,
		Message: message,
	})
}
