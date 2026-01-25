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

package deploy

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Executor struct {
	workDir string
	onLog   func(stream, line string)
}

type Config struct {
	URL         string
	Name        string
	Branch      string
	Commit      string
	Path        string
	BuildSystem string
	BuildFile   string
	BuildCmd    string
	Env         map[string]string
}

type Result struct {
	Success  bool
	Duration time.Duration
	Commit   string
	Error    string
}

func NewExecutor(workDir string) *Executor {
	os.MkdirAll(workDir, 0755)
	return &Executor{workDir: workDir}
}

func (e *Executor) OnLog(handler func(stream, line string)) {
	e.onLog = handler
}

func (e *Executor) Execute(ctx context.Context, cfg Config) (*Result, error) {
	start := time.Now()
	result := &Result{}

	repoDir := filepath.Join(e.workDir, cfg.Name)
	if cfg.Path != "" {
		repoDir = cfg.Path
	}

	e.log("stdout", fmt.Sprintf("› Deploying %s", cfg.Name))

	e.log("stdout", "› Cloning/pulling repository...")
	if err := e.cloneOrPull(ctx, cfg.URL, cfg.Branch, repoDir); err != nil {
		result.Error = err.Error()
		return result, err
	}

	if cfg.Commit != "" && cfg.Commit != "HEAD" {
		shortCommit := cfg.Commit
		if len(shortCommit) > 7 {
			shortCommit = shortCommit[:7]
		}
		e.log("stdout", fmt.Sprintf("› Checking out %s", shortCommit))
		if err := e.runCmd(ctx, repoDir, "git", "checkout", cfg.Commit); err != nil {
			result.Error = err.Error()
			return result, err
		}
	}

	hash, _ := e.getCommitHash(ctx, repoDir)
	result.Commit = hash

	cmd, err := e.resolveCommand(repoDir, cfg)
	if err != nil {
		result.Error = err.Error()
		e.log("stderr", result.Error)
		return result, err
	}

	e.log("stdout", fmt.Sprintf("› Running: %s", cmd))
	if err := e.runScript(ctx, repoDir, cmd, cfg.Env); err != nil {
		result.Error = err.Error()
		return result, err
	}

	result.Success = true
	result.Duration = time.Since(start)

	e.log("stdout", fmt.Sprintf("› Completed in %s", result.Duration.Round(time.Millisecond)))

	return result, nil
}

func (e *Executor) resolveCommand(repoDir string, cfg Config) (string, error) {
	if cfg.BuildCmd != "" {
		return cfg.BuildCmd, nil
	}

	switch cfg.BuildSystem {
	case "compose":
		file := cfg.BuildFile
		if file == "" {
			file = e.findComposeFile(repoDir)
		}
		if file == "" {
			return "", fmt.Errorf("no compose file found")
		}
		projectName := fmt.Sprintf("uruflow-%s", cfg.Name)
		return fmt.Sprintf("docker compose -p %s -f %s up -d --build", projectName, file), nil

	case "dockerfile":
		containerName := fmt.Sprintf("uruflow-%s", cfg.Name)
		if cfg.BuildFile != "" {
			return fmt.Sprintf("docker build -f %s -t %s . && docker run -d --name %s --label io.uruflow.managed=true %s",
				cfg.BuildFile, cfg.Name, containerName, cfg.Name), nil
		}
		if !e.fileExists(repoDir, "Dockerfile") {
			return "", fmt.Errorf("no Dockerfile found")
		}
		return fmt.Sprintf("docker build -t %s . && docker run -d --name %s --label io.uruflow.managed=true %s",
			cfg.Name, containerName, cfg.Name), nil

	case "makefile":
		file := cfg.BuildFile
		if file == "" {
			file = "Makefile"
		}
		if !e.fileExists(repoDir, file) {
			return "", fmt.Errorf("no %s found", file)
		}
		return fmt.Sprintf("make -f %s deploy", file), nil

	case "":
		return "", fmt.Errorf("build_system not specified in repository config")

	default:
		return "", fmt.Errorf("unknown build_system: %s", cfg.BuildSystem)
	}
}

func (e *Executor) findComposeFile(repoDir string) string {
	if e.fileExists(repoDir, "docker-compose.yml") {
		return "docker-compose.yml"
	}
	if e.fileExists(repoDir, "docker-compose.yaml") {
		return "docker-compose.yaml"
	}
	return ""
}

func (e *Executor) fileExists(repoDir, file string) bool {
	_, err := os.Stat(filepath.Join(repoDir, file))
	return err == nil
}

func (e *Executor) cloneOrPull(ctx context.Context, repoURL, branch, repoDir string) error {
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); os.IsNotExist(err) {
		parentDir := filepath.Dir(repoDir)
		os.MkdirAll(parentDir, 0755)
		return e.runCmd(ctx, parentDir, "git", "clone", "-b", branch, "--single-branch", repoURL, filepath.Base(repoDir))
	}

	if err := e.runCmd(ctx, repoDir, "git", "fetch", "origin"); err != nil {
		return err
	}

	return e.runCmd(ctx, repoDir, "git", "reset", "--hard", "origin/"+branch)
}

func (e *Executor) getCommitHash(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (e *Executor) runScript(ctx context.Context, dir, script string, env map[string]string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		e.scanPipe(stdout, "stdout")
		done <- struct{}{}
	}()
	go func() {
		e.scanPipe(stderr, "stderr")
		done <- struct{}{}
	}()

	<-done
	<-done

	return cmd.Wait()
}

func (e *Executor) runCmd(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		e.scanPipe(stdout, "stdout")
		done <- struct{}{}
	}()
	go func() {
		e.scanPipe(stderr, "stderr")
		done <- struct{}{}
	}()

	<-done
	<-done

	return cmd.Wait()
}

func (e *Executor) scanPipe(r interface{ Read([]byte) (int, error) }, stream string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		e.log(stream, scanner.Text())
	}
}

func (e *Executor) log(stream, line string) {
	if e.onLog != nil {
		e.onLog(stream, line)
	}
}
