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

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/urustack/uruflow/internal/agent/config"
	"github.com/urustack/uruflow/internal/agent/daemon"
)

const version = "1.1.0"

const (
	colorReset = "\033[0m"
	colorBlue  = "\033[38;2;59;130;246m"
	colorGreen = "\033[38;2;34;197;94m"
	colorRed   = "\033[38;2;239;68;68m"
	colorGray  = "\033[38;2;107;114;128m"
)

var configPath string

func main() {
	parseFlags()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	if strings.HasPrefix(cmd, "-") {
		cmd = os.Args[len(os.Args)-1]
	}

	switch cmd {
	case "init":
		cmdInit()
	case "start":
		cmdStart()
	case "stop":
		cmdStop()
	case "restart":
		cmdRestart()
	case "status":
		cmdStatus()
	case "run":
		cmdRun()
	case "version", "-v", "--version":
		fmt.Printf("uruflow-agent %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("%s✗%s Unknown command: %s\n", colorRed, colorReset, cmd)
		printUsage()
		os.Exit(1)
	}
}

func parseFlags() {
	for i, arg := range os.Args {
		if arg == "-c" || arg == "--config" {
			if i+1 < len(os.Args) {
				configPath = os.Args[i+1]
			}
		}
		if strings.HasPrefix(arg, "--config=") {
			configPath = strings.TrimPrefix(arg, "--config=")
		}
	}
	if configPath == "" {
		configPath = config.DefaultConfigPath
	}
}

func printUsage() {
	fmt.Println()
	fmt.Printf("  %suruflow-agent%s v%s\n", colorBlue, colorReset, version)
	fmt.Println()
	fmt.Println("  Usage:")
	fmt.Println("    uruflow-agent [options] <command>")
	fmt.Println()
	fmt.Println("  Options:")
	fmt.Printf("    -c, --config    %sConfig file path (default: %s)%s\n", colorGray, config.DefaultConfigPath, colorReset)
	fmt.Println()
	fmt.Println("  Commands:")
	fmt.Printf("    init      %sSetup agent configuration%s\n", colorGray, colorReset)
	fmt.Printf("    start     %sStart the agent daemon%s\n", colorGray, colorReset)
	fmt.Printf("    stop      %sStop the agent daemon%s\n", colorGray, colorReset)
	fmt.Printf("    restart   %sRestart the agent daemon%s\n", colorGray, colorReset)
	fmt.Printf("    status    %sShow agent status%s\n", colorGray, colorReset)
	fmt.Printf("    version   %sShow version%s\n", colorGray, colorReset)
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Printf("    uruflow-agent init\n")
	fmt.Printf("    uruflow-agent start\n")
	fmt.Printf("    uruflow-agent --config /custom/path/agent.yaml status\n")
	fmt.Println()
}

func cmdInit() {
	fmt.Println()
	fmt.Printf("  %s┬ ┬┬─┐┬ ┬┌─┐┬  ┌─┐┬ ┬%s\n", colorBlue, colorReset)
	fmt.Printf("  %s│ │├┬┘│ │├┤ │  │ ││││%s\n", colorBlue, colorReset)
	fmt.Printf("  %s└─┘┴└─└─┘└  ┴─┘└─┘└┴┘%s  agent\n", colorBlue, colorReset)
	fmt.Println()
	fmt.Printf("  %sAgent Setup%s\n", colorGray, colorReset)
	fmt.Printf("  %sConfig: %s%s\n", colorGray, configPath, colorReset)
	fmt.Println()

	if config.Exists(configPath) {
		fmt.Printf("  %s!%s Config already exists at %s\n", colorRed, colorReset, configPath)
		fmt.Print("  Overwrite? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Println("  Aborted.")
			return
		}
		fmt.Println()
	}

	cfg := config.Default()
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("  Agent token: ")
	cfg.Token, _ = reader.ReadString('\n')
	cfg.Token = strings.TrimSpace(cfg.Token)

	fmt.Print("  Server host: ")
	cfg.Server.Host, _ = reader.ReadString('\n')
	cfg.Server.Host = strings.TrimSpace(cfg.Server.Host)

	fmt.Print("  Server port [9001]: ")
	portStr, _ := reader.ReadString('\n')
	portStr = strings.TrimSpace(portStr)
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			cfg.Server.Port = p
		}
	}

	fmt.Println()

	if err := cfg.Validate(); err != nil {
		fmt.Printf("  %s✗%s %s\n", colorRed, colorReset, err.Error())
		os.Exit(1)
	}

	os.MkdirAll(cfg.DataDir, 0755)

	if err := cfg.Save(configPath); err != nil {
		fmt.Printf("  %s✗%s Failed to save config: %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	fmt.Printf("  %s✓%s Config saved to %s\n", colorGreen, colorReset, configPath)
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("    %suruflow-agent start%s\n", colorBlue, colorReset)
	fmt.Println()
}

func cmdStart() {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("  %s✗%s %v\n", colorRed, colorReset, err)
		fmt.Printf("  %s→%s Run: uruflow-agent init\n", colorGray, colorReset)
		os.Exit(1)
	}

	running, pid := daemon.IsRunning(cfg.PidFile)
	if running {
		fmt.Printf("  %s!%s Agent already running (pid %d)\n", colorRed, colorReset, pid)
		os.Exit(1)
	}

	args := []string{"run"}
	if configPath != config.DefaultConfigPath {
		args = []string{"--config", configPath, "run"}
	}

	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		fmt.Printf("  %s✗%s Failed to start: %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	time.Sleep(500 * time.Millisecond)

	running, pid = daemon.IsRunning(cfg.PidFile)
	if running {
		fmt.Printf("  %s✓%s Agent started (pid %d)\n", colorGreen, colorReset, pid)
	} else {
		fmt.Printf("  %s✗%s Failed to start. Check logs: %s\n", colorRed, colorReset, cfg.LogFile)
		os.Exit(1)
	}
}

func cmdStop() {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("  %s✗%s %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	running, pid := daemon.IsRunning(cfg.PidFile)
	if !running {
		fmt.Printf("  %s!%s Agent not running\n", colorGray, colorReset)
		return
	}

	if err := daemon.Stop(cfg.PidFile); err != nil {
		fmt.Printf("  %s✗%s Failed to stop: %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if running, _ := daemon.IsRunning(cfg.PidFile); !running {
			fmt.Printf("  %s✓%s Agent stopped (was pid %d)\n", colorGreen, colorReset, pid)
			return
		}
	}

	fmt.Printf("  %s!%s Agent may still be running\n", colorRed, colorReset)
}

func cmdRestart() {
	cmdStop()
	time.Sleep(time.Second)
	cmdStart()
}

func cmdStatus() {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("  %s✗%s %v\n", colorRed, colorReset, err)
		fmt.Printf("  %s→%s Run: uruflow-agent init\n", colorGray, colorReset)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("  %sAgent Status%s\n", colorGray, colorReset)
	fmt.Println()

	running, pid := daemon.IsRunning(cfg.PidFile)

	if running {
		fmt.Printf("  Status   %s● running%s (pid %d)\n", colorGreen, colorReset, pid)
	} else {
		fmt.Printf("  Status   %s○ stopped%s\n", colorGray, colorReset)
	}

	fmt.Printf("  Server   %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  Config   %s\n", configPath)
	fmt.Printf("  PID      %s\n", cfg.PidFile)
	fmt.Printf("  Logs     %s\n", cfg.LogFile)
	fmt.Println()
}

func cmdRun() {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	d, err := daemon.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create daemon: %v\n", err)
		os.Exit(1)
	}

	if err := d.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run daemon: %v\n", err)
		os.Exit(1)
	}
}
