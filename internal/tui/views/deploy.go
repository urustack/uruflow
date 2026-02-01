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

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urustack/uruflow/internal/storage"
	"github.com/urustack/uruflow/internal/tui/components"
	"github.com/urustack/uruflow/internal/tui/styles"
	"github.com/urustack/uruflow/pkg/helper"
)

type DeployModel struct {
	store      storage.Store
	Width      int
	Height     int
	Deployment DeploymentData
	Steps      []DeployStep
	CurrentLog string
	err        error
}

type DeployStep struct {
	Name     string
	Status   string
	Duration string
}

func NewDeployModel(store storage.Store) DeployModel {
	return DeployModel{store: store, Deployment: DeploymentData{Status: "idle"}}
}

func (m DeployModel) Init() tea.Cmd {
	return m.pollStatus
}

func (m DeployModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return m, m.fetchStatus
		}
	case TickMsg:
		if m.Deployment.Status == "running" || m.Deployment.Status == "pending" {
			return m, tea.Batch(m.fetchStatus, m.pollStatus)
		}
	case DeploymentData:
		m.Deployment = msg
		return m, nil
	case error:
		m.err = msg
		return m, nil
	}
	return m, nil
}

func (m DeployModel) pollStatus() tea.Msg {
	time.Sleep(1 * time.Second)
	return TickMsg(time.Now())
}

func (m DeployModel) fetchStatus() tea.Msg {
	if m.Deployment.ID == "" {
		return nil
	}
	d, err := m.store.GetDeployment(m.Deployment.ID)
	if err != nil {
		return err
	}
	if d == nil {
		return nil
	}
	return DeploymentData{
		ID: d.ID, Repo: d.Repository, Branch: d.Branch, Commit: d.Commit,
		Agent: d.AgentName, Status: string(d.Status),
		Time: time.Since(d.StartedAt).Round(time.Second).String(),
	}
}

func (m *DeployModel) SetDeployment(id, repo, branch, commit, agent string) {
	m.Deployment = DeploymentData{ID: id, Repo: repo, Branch: branch, Commit: commit, Agent: agent, Status: "pending"}
}

func (m DeployModel) View() string {
	if m.Width == 0 {
		return ""
	}

	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.ViewHeader(w, "Dashboard", "Deployment") + "\n\n")

	if m.Deployment.ID == "" {
		b.WriteString(components.Section("STATUS", w) + "\n\n")
		b.WriteString(components.Empty("No deployment in progress", "Start a deployment from the repositories view", w) + "\n")
	} else {
		b.WriteString(components.Section("CURRENT DEPLOYMENT", w) + "\n\n")

		var infoContent strings.Builder
		infoContent.WriteString(styles.TitleStyle.Render(m.Deployment.Repo) + "  " + components.Badge(m.Deployment.Status) + "\n")
		infoContent.WriteString("\n" + styles.SubtleStyle.Render("Branch ") + m.Deployment.Branch)
		infoContent.WriteString("\n" + styles.SubtleStyle.Render("Commit ") + styles.MutedStyle.Render(m.Deployment.Commit))
		infoContent.WriteString("\n" + styles.SubtleStyle.Render("Agent  ") + m.Deployment.Agent)
		b.WriteString(components.Wrap(infoContent.String(), w) + "\n\n")

		if len(m.Steps) > 0 {
			b.WriteString(components.Section("PROGRESS", w) + "\n\n")
			steps := make([]components.ProgressStep, len(m.Steps))
			for i, s := range m.Steps {
				steps[i] = components.ProgressStep{Label: s.Name, Status: s.Status, Duration: s.Duration}
			}
			b.WriteString(components.Wrap(components.Progress(steps, w-8), w) + "\n")

			if m.CurrentLog != "" && m.Deployment.Status == "running" {
				b.WriteString("\n  " + styles.SubtleStyle.Render(m.CurrentLog) + "\n")
			}
		}

		if m.Deployment.Status == "success" {
			b.WriteString("\n" + components.MsgSuccess(fmt.Sprintf("Deployment completed in %s", m.Deployment.Time), w) + "\n")
		} else if m.Deployment.Status == "failed" {
			b.WriteString("\n" + components.MsgError("Deployment failed", w) + "\n")
		}
	}

	content := b.String()
	lines := helper.CountLines(content)
	for i := 0; i < m.Height-lines-3; i++ {
		content += "\n"
	}

	content += "\n" + styles.Line(w) + "\n"
	if m.Deployment.Status == "running" {
		content += components.Help([][]string{{"l", "logs"}})
		content += "   " + styles.PrimaryStyle.Render(styles.IconSpin) + " " + styles.MutedStyle.Render("deploying...")
	} else {
		content += components.Help([][]string{{"l", "logs"}, {"esc", "back"}})
	}

	return content
}
