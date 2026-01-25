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

package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/urustack/uruflow/internal/agent/config"
	"github.com/urustack/uruflow/internal/agent/deploy"
	"github.com/urustack/uruflow/internal/agent/docker"
	"github.com/urustack/uruflow/internal/agent/metrics"
	"github.com/urustack/uruflow/internal/tcp/protocol"
	"github.com/urustack/uruflow/pkg/logger"
)

const Version = "1.0.0"

type Daemon struct {
	cfg           *config.Config
	conn          net.Conn
	reader        *protocol.Reader
	writer        *protocol.Writer
	docker        *docker.Service
	metrics       *metrics.Collector
	deployer      *deploy.Executor
	agentID       string
	name          string
	stopChan      chan struct{}
	writeMu       sync.Mutex
	streamCancels map[string]context.CancelFunc
	streamMu      sync.Mutex
}

func New(cfg *config.Config) (*Daemon, error) {
	logDir := filepath.Dir(cfg.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	if err := logger.Init(cfg.LogFile, "info"); err != nil {
		return nil, fmt.Errorf("initialize logger: %w", err)
	}

	logger.Info("[AGENT] initializing uruflow-agent v%s", Version)

	var dockerSvc *docker.Service
	if cfg.Docker.Enabled {
		var err error
		dockerSvc, err = docker.New(cfg.Docker.Socket)
		if err != nil {
			logger.Warn("[AGENT] docker unavailable: %v", err)
		} else {
			logger.Info("[AGENT] docker connection established on %s", cfg.Docker.Socket)
		}
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	workDir := filepath.Join(cfg.DataDir, "repos")
	logger.Debug("[AGENT] work directory: %s", workDir)

	return &Daemon{
		cfg:           cfg,
		docker:        dockerSvc,
		metrics:       metrics.NewCollector(),
		deployer:      deploy.NewExecutor(workDir),
		stopChan:      make(chan struct{}),
		streamCancels: make(map[string]context.CancelFunc),
	}, nil
}

func (d *Daemon) Run() error {
	logger.Info("[AGENT] starting agent")

	if err := d.writePid(); err != nil {
		return fmt.Errorf("write pid: %w", err)
	}
	defer d.removePid()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("[AGENT] shutdown signal received")
		close(d.stopChan)
	}()

	for {
		select {
		case <-d.stopChan:
			logger.Info("[AGENT] Agent stopped")
			return nil
		default:
			if err := d.connect(); err != nil {
				logger.Error("[AGENT] connection failed: %v", err)
				logger.Info("[AGENT] reconnecting in %d seconds...", d.cfg.Server.ReconnectSec)
				time.Sleep(time.Duration(d.cfg.Server.ReconnectSec) * time.Second)
				continue
			}

			d.runLoop()
		}
	}
}

func (d *Daemon) connect() error {
	addr := fmt.Sprintf("%s:%d", d.cfg.Server.Host, d.cfg.Server.Port)
	logger.Info("[AGENT] connecting to %s", addr)

	var conn net.Conn
	var err error

	if d.cfg.Server.TLS {
		logger.Debug("[AGENT] using TLS connection")
		conn, err = d.connectTLS(addr)
	} else {
		logger.Debug("[AGENT] using plain TCP connection")
		conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
	}

	if err != nil {
		return err
	}

	d.conn = conn
	d.reader = protocol.NewReader(conn)
	d.writer = protocol.NewWriter(conn)

	if err := d.authenticate(); err != nil {
		d.conn.Close()
		return fmt.Errorf("auth: %w", err)
	}

	logger.Info("[AGENT] connected as '%s' (ID: %s)", d.name, d.agentID)
	return nil
}

func (d *Daemon) authenticate() error {
	hostname, _ := os.Hostname()

	logger.Debug("[AGENT] authenticating with token")

	authMsg, err := protocol.NewMessage(protocol.TypeAuth, protocol.AuthPayload{
		Token:    d.cfg.Token,
		Hostname: hostname,
		Version:  Version,
	})
	if err != nil {
		return err
	}

	if err := d.writer.WriteWithTimeout(authMsg, 10*time.Second); err != nil {
		return err
	}

	resp, err := d.reader.ReadWithTimeout(10 * time.Second)
	if err != nil {
		return err
	}

	if resp.Type == protocol.TypeAuthFail {
		var fail protocol.AuthFailPayload
		resp.Decode(&fail)
		logger.Error("[AGENT] authentication rejected: %s", fail.Reason)
		return fmt.Errorf("auth rejected: %s", fail.Reason)
	}

	if resp.Type != protocol.TypeAuthOK {
		return fmt.Errorf("unexpected response: %s", resp.Type)
	}

	var ok protocol.AuthOKPayload
	if err := resp.Decode(&ok); err != nil {
		return err
	}

	d.agentID = ok.AgentID
	d.name = ok.Name

	logger.Info("[AGENT] authentication successful")
	return nil
}

func (d *Daemon) runLoop() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgChan := make(chan *protocol.Message, 10)
	errChan := make(chan error, 1)

	go d.readMessages(ctx, msgChan, errChan)

	metricsTicker := time.NewTicker(time.Duration(d.cfg.Server.MetricsSec) * time.Second)
	defer metricsTicker.Stop()

	logger.Debug("[AGENT] starting metrics collection (interval: %ds)", d.cfg.Server.MetricsSec)
	d.sendMetrics()

	for {
		select {
		case <-d.stopChan:
			logger.Debug("[AGENT] stop signal received in run loop")
			cancel()
			d.disconnect()
			return

		case <-metricsTicker.C:
			d.sendMetrics()

		case msg := <-msgChan:
			d.handleMessage(msg)

		case err := <-errChan:
			logger.Error("[AGENT] read error: %v", err)
			cancel()
			d.disconnect()
			return
		}
	}
}

func (d *Daemon) readMessages(ctx context.Context, msgChan chan<- *protocol.Message, errChan chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			d.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			msg, err := d.reader.Read()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				select {
				case errChan <- err:
				case <-ctx.Done():
				}
				return
			}
			select {
			case msgChan <- msg:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (d *Daemon) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.TypePing:
		logger.Debug("[AGENT] received ping, sending pong")
		d.safeWrite(protocol.Pong())

	case protocol.TypeCommand:
		var cmd protocol.CommandPayload
		if err := msg.Decode(&cmd); err != nil {
			logger.Error("[AGENT] failed to decode command: %v", err)
			return
		}
		go d.handleCommand(cmd)

	case protocol.TypeMetricsAck:
		logger.Debug("[AGENT] metrics acknowledged by server")

	case protocol.TypeDisconnect:
		logger.Info("[AGENT] disconnect request received from server")
		d.disconnect()

	case protocol.TypeContainerLogsRequest:
		var req protocol.ContainerLogsRequestPayload
		if err := msg.Decode(&req); err == nil {
			go d.handleContainerLogsRequest(req)
		} else {
			logger.Error("[AGENT] failed to decode container logs request: %v", err)
		}

	case protocol.TypeContainerLogsStop:
		var stop protocol.ContainerLogsStopPayload
		if err := msg.Decode(&stop); err == nil {
			d.stopContainerStream(stop.ContainerID)
		} else {
			logger.Error("[AGENT] failed to decode container logs stop: %v", err)
		}

	default:
		logger.Warn("[AGENT] unknown message type: %v", msg.Type)
	}
}

func (d *Daemon) safeWrite(msg *protocol.Message) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	return d.writer.Write(msg)
}

func (d *Daemon) sendMetrics() {
	sysMetrics, err := d.metrics.Collect()
	if err != nil {
		logger.Error("[AGENT] failed to collect system metrics: %v", err)
		return
	}

	logger.Debug("[AGENT] collected metrics - CPU: %.1f%%, Memory: %.1f%%, Disk: %.1f%%",
		sysMetrics.CPUPercent, sysMetrics.MemoryPercent, sysMetrics.DiskPercent)

	payload := protocol.MetricsPayload{
		Timestamp: time.Now().Unix(),
		System: protocol.SystemMetrics{
			CPUPercent:    sysMetrics.CPUPercent,
			MemoryPercent: sysMetrics.MemoryPercent,
			MemoryUsed:    sysMetrics.MemoryUsed,
			MemoryTotal:   sysMetrics.MemoryTotal,
			DiskPercent:   sysMetrics.DiskPercent,
			DiskUsed:      sysMetrics.DiskUsed,
			DiskTotal:     sysMetrics.DiskTotal,
			LoadAvg:       sysMetrics.LoadAvg,
			Uptime:        sysMetrics.Uptime,
		},
	}

	if d.docker != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		containers, err := d.docker.ListContainers(ctx)
		cancel()

		if err == nil {
			for _, c := range containers {
				if !c.IsManaged {
					logger.Debug("[AGENT] skipping non-uruflow container: %s", c.Name)
					continue
				}

				cm := protocol.Container{
					ID:           c.ID,
					Name:         c.Name,
					Image:        c.Image,
					Status:       c.State,
					Health:       c.Health,
					RestartCount: c.RestartCount,
					StartedAt:    c.StartedAt,
				}

				if c.State == "running" {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					stats, err := d.docker.GetContainerStats(ctx, c.FullID)
					cancel()
					if err == nil {
						cm.CPUPercent = stats.CPUPercent
						cm.MemoryUsage = stats.MemoryUsage
						cm.MemoryLimit = stats.MemoryLimit
						cm.NetworkRx = stats.NetworkRx
						cm.NetworkTx = stats.NetworkTx
					}
				}

				payload.Containers = append(payload.Containers, cm)
			}
			if len(payload.Containers) > 0 {
				logger.Debug("[AGENT] reporting %d uruflow-managed containers", len(payload.Containers))
			}
		} else {
			logger.Warn("[AGENT] failed to list containers: %v", err)
		}
	}

	msg, err := protocol.NewMessage(protocol.TypeMetrics, payload)
	if err != nil {
		logger.Error("[AGENT] failed to create metrics message: %v", err)
		return
	}

	if err := d.safeWrite(msg); err != nil {
		logger.Error("[AGENT] failed to send metrics: %v", err)
	}
}

func (d *Daemon) handleCommand(cmd protocol.CommandPayload) {
	logger.Info("[AGENT] received command: %s (ID: %s)", cmd.Type, cmd.ID)

	ackMsg, _ := protocol.NewMessage(protocol.TypeCommandAck, protocol.CommandAckPayload{
		CommandID: cmd.ID,
		Status:    "received",
	})
	d.safeWrite(ackMsg)

	switch cmd.Type {
	case "deploy":
		d.handleDeploy(cmd)
	default:
		logger.Warn("[AGENT] unknown command type: %s", cmd.Type)
		d.sendCommandDone(cmd.ID, "failed", 1, fmt.Sprintf("unknown command type: %s", cmd.Type))
	}
}

func (d *Daemon) handleDeploy(cmd protocol.CommandPayload) {
	payloadBytes, _ := json.Marshal(cmd.Payload)
	var deployPayload struct {
		URL         string `json:"url"`
		Name        string `json:"name"`
		Branch      string `json:"branch"`
		Commit      string `json:"commit"`
		Path        string `json:"path"`
		BuildSystem string `json:"build_system"`
		BuildFile   string `json:"build_file"`
		BuildCmd    string `json:"build_cmd"`
	}

	if err := json.Unmarshal(payloadBytes, &deployPayload); err != nil {
		logger.Error("[AGENT] failed to parse deploy payload: %v", err)
		d.sendCommandDone(cmd.ID, "failed", 1, err.Error())
		return
	}

	commitShort := deployPayload.Commit
	if len(commitShort) > 7 {
		commitShort = commitShort[:7]
	}
	logger.Info("[AGENT] starting deployment: repo=%s branch=%s commit=%s build_system=%s",
		deployPayload.Name, deployPayload.Branch, commitShort, deployPayload.BuildSystem)

	startMsg, _ := protocol.NewMessage(protocol.TypeCommandStart, protocol.CommandStartPayload{
		CommandID: cmd.ID,
		StartedAt: time.Now().Unix(),
	})
	d.safeWrite(startMsg)

	d.deployer.OnLog(func(stream, line string) {
		logMsg, _ := protocol.NewMessage(protocol.TypeCommandLog, protocol.CommandLogPayload{
			CommandID: cmd.ID,
			Line:      line,
			Stream:    stream,
			Timestamp: time.Now().Unix(),
		})
		d.safeWrite(logMsg)
	})

	cfg := deploy.Config{
		URL:         deployPayload.URL,
		Name:        deployPayload.Name,
		Branch:      deployPayload.Branch,
		Commit:      deployPayload.Commit,
		Path:        deployPayload.Path,
		BuildSystem: deployPayload.BuildSystem,
		BuildFile:   deployPayload.BuildFile,
		BuildCmd:    deployPayload.BuildCmd,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	result, err := d.deployer.Execute(ctx, cfg)

	status := "success"
	exitCode := 0
	output := ""

	if err != nil {
		status = "failed"
		exitCode = 1
		output = err.Error()
		logger.Error("[AGENT] deployment %s failed: %v", cmd.ID, err)
	} else {
		logger.Info("[AGENT] deployment %s succeeded (duration: %v)", cmd.ID, result.Duration)
	}

	d.sendCommandDone(cmd.ID, status, exitCode, output)

	if result != nil && result.Commit != "" {
		commitShort := result.Commit
		if len(commitShort) > 7 {
			commitShort = commitShort[:7]
		}
		logger.Info("[AGENT] deploy %s completed: status=%s duration=%v commit=%s",
			cmd.ID, status, result.Duration, commitShort)
	}
}

func (d *Daemon) handleContainerLogsRequest(req protocol.ContainerLogsRequestPayload) {
	d.stopContainerStream(req.ContainerID)

	ctx, cancel := context.WithCancel(context.Background())
	d.streamMu.Lock()
	d.streamCancels[req.ContainerID] = cancel
	d.streamMu.Unlock()

	containerID := req.ContainerID
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}

	logger.Info("[AGENT] starting log stream for container %s (tail: %d, follow: %t)",
		containerID, req.Tail, req.Follow)

	err := d.docker.StreamLogsWithTail(ctx, req.ContainerID, req.Tail, func(line string) {
		payload := protocol.ContainerLogsDataPayload{
			ContainerID: req.ContainerID,
			Line:        line,
			Stream:      "stdout",
			Timestamp:   time.Now().Unix(),
		}

		if len(line) > 9 && line[:9] == "[stderr] " {
			payload.Stream = "stderr"
			payload.Line = line[9:]
		}

		msg, _ := protocol.NewMessage(protocol.TypeContainerLogsData, payload)
		d.safeWrite(msg)
	})

	if err != nil && err != context.Canceled {
		logger.Error("[AGENT] log stream error for container %s: %v", containerID, err)
	} else {
		logger.Debug("[AGENT] log stream ended for container %s", containerID)
	}

	d.stopContainerStream(req.ContainerID)
}

func (d *Daemon) stopContainerStream(containerID string) {
	d.streamMu.Lock()
	defer d.streamMu.Unlock()
	if cancel, ok := d.streamCancels[containerID]; ok {
		shortID := containerID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		logger.Debug("[AGENT] stopping log stream for container %s", shortID)
		cancel()
		delete(d.streamCancels, containerID)
	}
}

func (d *Daemon) sendCommandDone(cmdID, status string, exitCode int, output string) {
	logger.Debug("[AGENT] sending command done: id=%s status=%s exit_code=%d", cmdID, status, exitCode)

	doneMsg, _ := protocol.NewMessage(protocol.TypeCommandDone, protocol.CommandDonePayload{
		CommandID: cmdID,
		Status:    status,
		ExitCode:  exitCode,
		Output:    output,
	})
	d.safeWrite(doneMsg)
}

func (d *Daemon) disconnect() {
	if d.conn != nil {
		logger.Info("[AGENT] disconnecting from server")
		d.safeWrite(protocol.Disconnect())
		d.conn.Close()
		d.conn = nil
	}

	d.streamMu.Lock()
	streamCount := len(d.streamCancels)
	for _, cancel := range d.streamCancels {
		cancel()
	}
	d.streamCancels = make(map[string]context.CancelFunc)
	d.streamMu.Unlock()

	if streamCount > 0 {
		logger.Debug("[AGENT] cancelled %d active log streams", streamCount)
	}
}

func (d *Daemon) writePid() error {
	dir := filepath.Dir(d.cfg.PidFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	pid := os.Getpid()
	if err := os.WriteFile(d.cfg.PidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return err
	}

	logger.Debug("[AGENT] PID file written: %s (PID: %d)", d.cfg.PidFile, pid)
	return nil
}

func (d *Daemon) removePid() {
	if err := os.Remove(d.cfg.PidFile); err != nil {
		logger.Debug("[AGENT] failed to remove PID file: %v", err)
	} else {
		logger.Debug("[AGENT] PID file removed: %s", d.cfg.PidFile)
	}
}

func IsRunning(pidFile string) (bool, int) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false, 0
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return false, 0
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil, pid
}

func Stop(pidFile string) error {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("read pid file: %w", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return fmt.Errorf("parse pid: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}

	logger.Info("[AGENT] sending SIGTERM to process %d", pid)
	return process.Signal(syscall.SIGTERM)
}
