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

package tcp

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/urustack/uruflow/internal/config"
	"github.com/urustack/uruflow/internal/logic"
	"github.com/urustack/uruflow/internal/models"
	"github.com/urustack/uruflow/internal/storage"
	"github.com/urustack/uruflow/internal/tcp/protocol"
	"github.com/urustack/uruflow/pkg/helper"
)

const (
	AuthTimeout  = 10 * time.Second
	PingInterval = 30 * time.Second
	PongTimeout  = 45 * time.Second
)

type Server struct {
	cfg            *config.Config
	store          storage.Store
	listener       net.Listener
	connections    map[string]*Connection
	mu             sync.RWMutex
	done           chan struct{}
	onLog          func(agentID string, log *models.CommandLog)
	onMetrics      func(agentID string, metrics *models.AgentMetrics)
	onContainerLog func(agentID string, data protocol.ContainerLogsDataPayload)
}

func NewServer(cfg *config.Config, store storage.Store) *Server {
	return &Server{
		cfg:         cfg,
		store:       store,
		connections: make(map[string]*Connection),
		done:        make(chan struct{}),
	}
}

func (s *Server) SetLogHandler(handler func(agentID string, log *models.CommandLog)) {
	s.onLog = handler
}

func (s *Server) SetMetricsHandler(handler func(agentID string, metrics *models.AgentMetrics)) {
	s.onMetrics = handler
}

func (s *Server) SetContainerLogHandler(handler func(agentID string, data protocol.ContainerLogsDataPayload)) {
	s.onContainerLog = handler
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.TCPPort)

	var listener net.Listener
	var err error

	if s.cfg.TLS.Enabled {
		listener, err = s.listenTLS(addr)
		if err != nil {
			return fmt.Errorf("tls listen: %w", err)
		}
		log.Printf("[TCP] server listening on %s (TLS)", addr)
	} else {
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("tcp listen: %w", err)
		}
		log.Printf("[TCP] server listening on %s", addr)
	}

	s.listener = listener

	go s.acceptRequests()
	go s.pingService()

	return nil
}

func (s *Server) listenTLS(addr string) (net.Listener, error) {
	if s.cfg.TLS.AutoCert {
		return s.listenAutoTLS(addr)
	}

	cert, err := tls.LoadX509KeyPair(s.cfg.TLS.CertFile, s.cfg.TLS.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load cert: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return tls.Listen("tcp", addr, tlsConfig)
}

func (s *Server) listenAutoTLS(addr string) (net.Listener, error) {
	cert, err := generateSelfSignedCert()
	if err != nil {
		return nil, fmt.Errorf("generate cert: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	log.Printf("[TCP] using auto-generated self-signed certificate")
	return tls.Listen("tcp", addr, tlsConfig)
}

func (s *Server) Stop() error {
	close(s.done)
	if s.listener != nil {
		s.listener.Close()
	}
	s.mu.Lock()
	for _, conn := range s.connections {
		conn.Send(protocol.Disconnect())
		conn.Close()
	}
	s.mu.Unlock()
	return nil
}

func (s *Server) acceptRequests() {
	for {
		select {
		case <-s.done:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.done:
					return
				default:
					log.Printf("[TCP] accept error: %v", err)
					continue
				}
			}
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(netConn net.Conn) {
	connID := helper.GenerateID()
	conn := NewConnection(connID, netConn)

	agentID, err := s.authenticate(conn)
	if err != nil {
		log.Printf("[TCP] auth failed for %s: %v", conn.RemoteAddr(), err)
		conn.Close()
		return
	}

	s.addConnection(agentID, conn)
	defer s.removeConnection(agentID)

	log.Printf("[TCP] agent %s connected", conn.AgentName)
	s.handleMessages(conn)
}

func (s *Server) authenticate(conn *Connection) (string, error) {
	msg, err := conn.ReceiveWithTimeout(AuthTimeout)
	if err != nil {
		return "", fmt.Errorf("receive auth: %w", err)
	}

	if msg.Type != protocol.TypeAuth {
		errMsg, _ := protocol.Error(401, "expected AUTH message")
		conn.Send(errMsg)
		return "", fmt.Errorf("expected AUTH, got %s", msg.Type)
	}

	var auth protocol.AuthPayload
	if err := msg.Decode(&auth); err != nil {
		errMsg, _ := protocol.Error(400, "invalid auth payload")
		conn.Send(errMsg)
		return "", fmt.Errorf("decode auth: %w", err)
	}

	agentCfg := s.cfg.GetAgentByToken(auth.Token)
	if agentCfg == nil {
		failMsg, _ := protocol.NewMessage(protocol.TypeAuthFail, protocol.AuthFailPayload{
			Reason: "invalid token",
		})
		conn.Send(failMsg)
		return "", fmt.Errorf("invalid token")
	}

	host, _, _ := net.SplitHostPort(conn.RemoteAddr())

	existingAgent, _ := s.store.GetAgent(agentCfg.ID)
	if existingAgent == nil {
		agent := &models.Agent{
			ID:            agentCfg.ID,
			Name:          agentCfg.Name,
			Token:         agentCfg.Token,
			Host:          host,
			Hostname:      auth.Hostname,
			Version:       auth.Version,
			Status:        models.AgentOnline,
			LastHeartbeat: time.Now(),
			RegisteredAt:  time.Now(),
		}
		s.store.CreateAgent(agent)
	} else {
		existingAgent.Host = host
		existingAgent.Hostname = auth.Hostname
		existingAgent.Version = auth.Version
		existingAgent.Status = models.AgentOnline
		existingAgent.LastHeartbeat = time.Now()
		s.store.UpdateAgent(existingAgent)
	}

	conn.SetAgent(agentCfg.ID, agentCfg.Name)

	okMsg, _ := protocol.NewMessage(protocol.TypeAuthOK, protocol.AuthOKPayload{
		AgentID:       agentCfg.ID,
		Name:          agentCfg.Name,
		ServerVersion: "1.0.0",
	})
	conn.Send(okMsg)

	return agentCfg.ID, nil
}

func (s *Server) handleMessages(conn *Connection) {
	for {
		select {
		case <-s.done:
			return
		default:
			conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			msg, err := conn.Receive()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return
			}
			s.processMessage(conn, msg)
		}
	}
}

func (s *Server) processMessage(conn *Connection, msg *protocol.Message) {
	switch msg.Type {
	case protocol.TypeMetrics:
		s.handleMetrics(conn, msg)
	case protocol.TypeCommandAck:
		s.handleCommandAck(conn, msg)
	case protocol.TypeCommandStart:
		s.handleCommandStart(conn, msg)
	case protocol.TypeCommandLog:
		s.handleCommandLog(conn, msg)
	case protocol.TypeCommandDone:
		s.handleCommandDone(conn, msg)
	case protocol.TypePong:
		conn.UpdatePing()
	case protocol.TypeDisconnect:
		conn.Close()
	case protocol.TypeContainerLogsData:
		var data protocol.ContainerLogsDataPayload
		if err := msg.Decode(&data); err == nil && s.onContainerLog != nil {
			s.onContainerLog(conn.AgentID, data)
		}
	}
}

func (s *Server) handleMetrics(conn *Connection, msg *protocol.Message) {
	var metrics protocol.MetricsPayload
	if err := msg.Decode(&metrics); err != nil {
		return
	}

	agentMetrics := &models.AgentMetrics{
		CPUPercent:    metrics.System.CPUPercent,
		MemoryPercent: metrics.System.MemoryPercent,
		MemoryUsed:    metrics.System.MemoryUsed,
		MemoryTotal:   metrics.System.MemoryTotal,
		DiskPercent:   metrics.System.DiskPercent,
		DiskUsed:      metrics.System.DiskUsed,
		DiskTotal:     metrics.System.DiskTotal,
		LoadAvg:       metrics.System.LoadAvg,
		Uptime:        metrics.System.Uptime,
	}
	s.store.UpdateAgentMetrics(conn.AgentID, agentMetrics)

	if s.onMetrics != nil {
		s.onMetrics(conn.AgentID, agentMetrics)
	}

	activeAlerts, _ := s.store.GetActiveAlerts()

	activeAlertMap := make(map[string]bool)
	for _, a := range activeAlerts {
		if a.AgentID == conn.AgentID && a.Resolved == false {
			activeAlertMap[a.Message] = true
		}
	}

	for _, c := range metrics.Containers {
		container := &models.Container{
			ID:           c.ID,
			AgentID:      conn.AgentID,
			Name:         c.Name,
			Image:        c.Image,
			Status:       c.Status,
			Health:       models.ContainerHealth(c.Health),
			CPUPercent:   c.CPUPercent,
			MemoryUsage:  c.MemoryUsage,
			MemoryLimit:  c.MemoryLimit,
			NetworkRx:    c.NetworkRx,
			NetworkTx:    c.NetworkTx,
			RestartCount: c.RestartCount,
			StartedAt:    time.Unix(c.StartedAt, 0),
		}
		s.store.UpsertContainer(container)

		if c.Status != "running" &&
			c.Status != "created" &&
			c.Status != "starting" &&
			c.Status != "restarting" {

			potentialAlert := logic.CheckContainerDown(conn.AgentID, conn.AgentName, c.Name)

			if potentialAlert != nil {
				if !activeAlertMap[potentialAlert.Message] {
					s.store.CreateAlert(potentialAlert)
					activeAlertMap[potentialAlert.Message] = true
				}
			}
		}
	}

	createIfNotExists := func(alert *models.Alert) {
		if alert != nil && !activeAlertMap[alert.Message] {
			s.store.CreateAlert(alert)
			activeAlertMap[alert.Message] = true
		}
	}

	createIfNotExists(logic.CheckCPU(conn.AgentID, conn.AgentName, metrics.System.CPUPercent))
	createIfNotExists(logic.CheckMemory(conn.AgentID, conn.AgentName, metrics.System.MemoryPercent))
	createIfNotExists(logic.CheckDisk(conn.AgentID, conn.AgentName, metrics.System.DiskPercent))

	conn.Send(&protocol.Message{Type: protocol.TypeMetricsAck})
}

func (s *Server) handleCommandAck(conn *Connection, msg *protocol.Message) {
	var ack protocol.CommandAckPayload
	if err := msg.Decode(&ack); err != nil {
		return
	}

	deploy, _ := s.store.GetDeployment(ack.CommandID)
	if deploy != nil && deploy.Status == models.DeployPending {
		deploy.Status = models.DeployRunning
		s.store.UpdateDeployment(deploy)
	}

	log.Printf("[TCP] agent %s acknowledged command %s", conn.AgentName, ack.CommandID)
}

func (s *Server) handleCommandStart(conn *Connection, msg *protocol.Message) {
	var start protocol.CommandStartPayload
	if err := msg.Decode(&start); err != nil {
		return
	}

	deploy, _ := s.store.GetDeployment(start.CommandID)
	if deploy != nil {
		deploy.Status = models.DeployRunning
		s.store.UpdateDeployment(deploy)
	}
	log.Printf("[TCP] agent %s started deployment %s", conn.AgentName, start.CommandID)
}

func (s *Server) handleCommandLog(conn *Connection, msg *protocol.Message) {
	var logPayload protocol.CommandLogPayload
	if err := msg.Decode(&logPayload); err != nil {
		return
	}

	cmdLog := &models.DeploymentLog{
		DeploymentID: logPayload.CommandID,
		Line:         logPayload.Line,
		Stream:       logPayload.Stream,
		Timestamp:    time.Unix(logPayload.Timestamp, 0),
	}

	s.store.AddDeploymentLog(cmdLog)

	if s.onLog != nil {
		legacyLog := &models.CommandLog{
			CommandID: cmdLog.DeploymentID,
			Line:      cmdLog.Line,
			Stream:    cmdLog.Stream,
			Timestamp: cmdLog.Timestamp,
		}
		s.onLog(conn.AgentID, legacyLog)
	}
}

func (s *Server) handleCommandDone(conn *Connection, msg *protocol.Message) {
	var done protocol.CommandDonePayload
	if err := msg.Decode(&done); err != nil {
		return
	}

	deploy, _ := s.store.GetDeployment(done.CommandID)
	if deploy != nil {
		status := models.DeploySuccess
		if done.Status != "success" {
			status = models.DeployFailed
		}

		now := time.Now()
		deploy.Status = status
		deploy.Output = done.Output
		deploy.EndedAt = &now
		deploy.Duration = int64(now.Sub(deploy.StartedAt) / time.Millisecond)

		s.store.UpdateDeployment(deploy)

		if done.Output != "" {
			streamType := "stdout"
			if status == models.DeployFailed {
				streamType = "stderr"
			}
			cmdLog := &models.DeploymentLog{
				DeploymentID: done.CommandID,
				Line:         done.Output,
				Stream:       streamType,
				Timestamp:    now,
			}
			s.store.AddDeploymentLog(cmdLog)
		}
	}

	log.Printf("[TCP] agent %s completed deployment %s: %s", conn.AgentName, done.CommandID, done.Status)
}

func (s *Server) pingService() {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.pingAll()
		}
	}
}

func (s *Server) pingAll() {
	s.mu.RLock()
	conns := make([]*Connection, 0, len(s.connections))
	for _, conn := range s.connections {
		conns = append(conns, conn)
	}
	s.mu.RUnlock()

	for _, conn := range conns {
		if time.Since(conn.LastPing) > PongTimeout {
			log.Printf("[TCP] agent %s ping timeout, disconnecting", conn.AgentName)
			s.removeConnection(conn.AgentID)
			continue
		}
		conn.Send(protocol.Ping())
	}
}

func (s *Server) addConnection(agentID string, conn *Connection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, exists := s.connections[agentID]; exists {
		old.Close()
	}
	s.connections[agentID] = conn
}

func (s *Server) removeConnection(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if conn, exists := s.connections[agentID]; exists {
		conn.Close()
		delete(s.connections, agentID)

		s.store.UpdateAgentStatus(agentID, models.AgentOffline)

		if alert := logic.CheckOffline(agentID, conn.AgentName); alert != nil {
			s.store.CreateAlert(alert)
		}

		log.Printf("[TCP] agent %s disconnected", conn.AgentName)
	}
}

func (s *Server) IsAgentConnected(agentID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.connections[agentID]
	return exists
}

func (s *Server) SendCommand(agentID string, cmd *models.Command) error {
	s.mu.RLock()
	conn, exists := s.connections[agentID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent not connected")
	}

	cmdMsg, err := protocol.NewMessage(protocol.TypeCommand, protocol.CommandPayload{
		ID:      cmd.ID,
		Type:    cmd.Type,
		Payload: cmd.Payload,
	})
	if err != nil {
		return err
	}

	return conn.Send(cmdMsg)
}

func (s *Server) StreamContainerLogs(agentID, containerID string, tail int, follow bool) error {
	s.mu.RLock()
	conn, exists := s.connections[agentID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent not connected")
	}

	msg, _ := protocol.NewMessage(protocol.TypeContainerLogsRequest, protocol.ContainerLogsRequestPayload{
		ContainerID: containerID,
		Tail:        tail,
		Follow:      follow,
	})
	return conn.Send(msg)
}

func (s *Server) StopContainerLogs(agentID, containerID string) error {
	s.mu.RLock()
	conn, exists := s.connections[agentID]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	msg, _ := protocol.NewMessage(protocol.TypeContainerLogsStop, protocol.ContainerLogsStopPayload{
		ContainerID: containerID,
	})
	return conn.Send(msg)
}

func (s *Server) GetConnectedAgents() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]string, 0, len(s.connections))
	for id := range s.connections {
		agents = append(agents, id)
	}
	return agents
}
