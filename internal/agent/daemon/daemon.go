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
	"log"
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
)

const Version = "1.0.0"

type Daemon struct {
	cfg      *config.Config
	conn     net.Conn
	reader   *protocol.Reader
	writer   *protocol.Writer
	docker   *docker.Service
	metrics  *metrics.Collector
	deployer *deploy.Executor
	logger   *log.Logger
	agentID  string
	name     string
	stopChan chan struct{}
	writeMu  sync.Mutex
}

func New(cfg *config.Config) (*Daemon, error) {
	logDir := filepath.Dir(cfg.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	logger := log.New(logFile, "[agent] ", log.LstdFlags)

	var dockerSvc *docker.Service
	if cfg.Docker.Enabled {
		dockerSvc, err = docker.New(cfg.Docker.Socket)
		if err != nil {
			logger.Printf("docker unavailable: %v", err)
		}
	}

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	workDir := filepath.Join(cfg.DataDir, "repos")

	return &Daemon{
		cfg:      cfg,
		docker:   dockerSvc,
		metrics:  metrics.NewCollector(),
		deployer: deploy.NewExecutor(workDir),
		logger:   logger,
		stopChan: make(chan struct{}),
	}, nil
}

func (d *Daemon) Run() error {
	d.logger.Println("starting agent")

	if err := d.writePid(); err != nil {
		return fmt.Errorf("write pid: %w", err)
	}
	defer d.removePid()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		d.logger.Println("shutdown signal received")
		close(d.stopChan)
	}()

	for {
		select {
		case <-d.stopChan:
			d.logger.Println("agent stopped")
			return nil
		default:
			if err := d.connect(); err != nil {
				d.logger.Printf("connect failed: %v", err)
				time.Sleep(time.Duration(d.cfg.Server.ReconnectSec) * time.Second)
				continue
			}

			d.runLoop()
		}
	}
}

func (d *Daemon) connect() error {
	addr := fmt.Sprintf("%s:%d", d.cfg.Server.Host, d.cfg.Server.Port)
	d.logger.Printf("connecting to %s", addr)

	var conn net.Conn
	var err error

	if d.cfg.Server.TLS {
		conn, err = d.connectTLS(addr)
	} else {
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

	d.logger.Printf("connected as %s (id: %s)", d.name, d.agentID)
	return nil
}

func (d *Daemon) authenticate() error {
	hostname, _ := os.Hostname()

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

	d.sendMetrics()

	for {
		select {
		case <-d.stopChan:
			cancel()
			d.disconnect()
			return

		case <-metricsTicker.C:
			d.sendMetrics()

		case msg := <-msgChan:
			d.handleMessage(msg)

		case err := <-errChan:
			d.logger.Printf("read error: %v", err)
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
		d.safeWrite(protocol.Pong())

	case protocol.TypeCommand:
		var cmd protocol.CommandPayload
		if err := msg.Decode(&cmd); err != nil {
			d.logger.Printf("decode command error: %v", err)
			return
		}
		go d.handleCommand(cmd)

	case protocol.TypeMetricsAck:

	case protocol.TypeDisconnect:
		d.disconnect()
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
		d.logger.Printf("metrics collect error: %v", err)
		return
	}

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
				cm := protocol.Container{
					ID:           c.ID,
					Name:         c.Name,
					Image:        c.Image,
					Status:       c.Status,
					Health:       c.Health,
					RestartCount: c.RestartCount,
					StartedAt:    c.StartedAt,
				}

				if c.State == "running" {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					stats, err := d.docker.GetContainerStats(ctx, c.ID)
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
		}
	}

	msg, err := protocol.NewMessage(protocol.TypeMetrics, payload)
	if err != nil {
		d.logger.Printf("create metrics message error: %v", err)
		return
	}

	if err := d.safeWrite(msg); err != nil {
		d.logger.Printf("send metrics error: %v", err)
	}
}

func (d *Daemon) handleCommand(cmd protocol.CommandPayload) {
	d.logger.Printf("received command: %s (id: %s)", cmd.Type, cmd.ID)

	ackMsg, _ := protocol.NewMessage(protocol.TypeCommandAck, protocol.CommandAckPayload{
		CommandID: cmd.ID,
		Status:    "received",
	})
	d.safeWrite(ackMsg)

	switch cmd.Type {
	case "deploy":
		d.handleDeploy(cmd)
	default:
		d.logger.Printf("unknown command type: %s", cmd.Type)
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
		d.sendCommandDone(cmd.ID, "failed", 1, err.Error())
		return
	}

	if deployPayload.URL == "" {
		d.sendCommandDone(cmd.ID, "failed", 1, "repository URL is required")
		return
	}

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
	}

	d.sendCommandDone(cmd.ID, status, exitCode, output)

	if result != nil {
		d.logger.Printf("deploy %s: %s (duration: %v)", cmd.ID, status, result.Duration)
	}
}

func (d *Daemon) sendCommandDone(cmdID, status string, exitCode int, output string) {
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
		d.safeWrite(protocol.Disconnect())
		d.conn.Close()
		d.conn = nil
	}
}

func (d *Daemon) writePid() error {
	dir := filepath.Dir(d.cfg.PidFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(d.cfg.PidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func (d *Daemon) removePid() {
	os.Remove(d.cfg.PidFile)
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

	return process.Signal(syscall.SIGTERM)
}
