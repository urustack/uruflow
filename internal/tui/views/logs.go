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

type LogsMode int

const (
	LogsModeSelect LogsMode = iota
	LogsModeView
)

type LogsModel struct {
	store        storage.Store
	Width        int
	Height       int
	Mode         LogsMode
	Deployments  []DeploymentData
	Cursor       int
	DeploymentID string
	Repo         string
	Commit       string
	Logs         []LogData
	Offset       int
	AutoFollow   bool
	err          error
}

func NewLogsModel(store storage.Store) LogsModel {
	return LogsModel{store: store, AutoFollow: true, Mode: LogsModeSelect}
}

func (m LogsModel) Init() tea.Cmd {
	return tea.Batch(m.fetchDeployments, m.tick)
}

func (m LogsModel) tick() tea.Msg {
	time.Sleep(1 * time.Second)
	return TickMsg(time.Now())
}

func (m LogsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		if m.Mode == LogsModeView && m.DeploymentID != "" {
			return m, tea.Batch(m.fetchLogs, m.tick)
		}
		return m, tea.Batch(m.fetchDeployments, m.tick)

	case tea.KeyMsg:
		switch m.Mode {
		case LogsModeSelect:
			return m.updateSelect(msg)
		case LogsModeView:
			return m.updateView(msg)
		}

	case []DeploymentData:
		m.Deployments = msg
		return m, nil

	case []LogData:
		m.Logs = msg
		if m.AutoFollow && len(m.Logs) > 0 {
			maxOffset := len(m.Logs) - (m.Height - 12)
			if maxOffset < 0 {
				maxOffset = 0
			}
			m.Offset = maxOffset
		}
		return m, nil

	case error:
		m.err = msg
		return m, nil
	}
	return m, nil
}

func (m LogsModel) updateSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < len(m.Deployments)-1 {
			m.Cursor++
		}
	case "enter":
		if len(m.Deployments) > 0 && m.Cursor < len(m.Deployments) {
			d := m.Deployments[m.Cursor]
			m.DeploymentID = d.ID
			m.Repo = d.Repo
			m.Commit = d.Commit
			m.Mode = LogsModeView
			m.Logs = nil
			m.Offset = 0
			m.AutoFollow = true
			return m, m.fetchLogs
		}
	case "r":
		return m, m.fetchDeployments
	}
	return m, nil
}

func (m LogsModel) updateView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Mode = LogsModeSelect
		m.DeploymentID = ""
		m.Logs = nil
		return m, m.fetchDeployments
	case "up", "k":
		if m.Offset > 0 {
			m.Offset--
			m.AutoFollow = false
		}
	case "down", "j":
		maxOffset := len(m.Logs) - (m.Height - 12)
		if maxOffset < 0 {
			maxOffset = 0
		}
		if m.Offset < maxOffset {
			m.Offset++
		}
	case "g":
		m.Offset = 0
		m.AutoFollow = false
	case "G":
		maxOffset := len(m.Logs) - (m.Height - 12)
		if maxOffset < 0 {
			maxOffset = 0
		}
		m.Offset = maxOffset
		m.AutoFollow = true
	case "f":
		m.AutoFollow = !m.AutoFollow
		if m.AutoFollow {
			maxOffset := len(m.Logs) - (m.Height - 12)
			if maxOffset < 0 {
				maxOffset = 0
			}
			m.Offset = maxOffset
		}
	case "r":
		return m, m.fetchLogs
	}
	return m, nil
}

func (m *LogsModel) SetDeployment(id, repo, commit string) {
	m.DeploymentID = id
	m.Repo = repo
	m.Commit = commit
	m.Mode = LogsModeView
	m.Logs = nil
	m.Offset = 0
	m.AutoFollow = true
}

func (m LogsModel) fetchDeployments() tea.Msg {
	deployments, err := m.store.GetRecentDeployments(20)
	if err != nil {
		return err
	}
	var data []DeploymentData
	for _, d := range deployments {
		commit := d.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		data = append(data, DeploymentData{
			ID: d.ID, Repo: d.Repository, Branch: d.Branch, Commit: commit,
			Agent: d.AgentName, Status: string(d.Status),
			Time: time.Since(d.StartedAt).Round(time.Second).String() + " ago",
		})
	}
	return data
}

func (m LogsModel) fetchLogs() tea.Msg {
	if m.DeploymentID == "" {
		return []LogData{}
	}
	logs, err := m.store.GetDeploymentLogs(m.DeploymentID)
	if err != nil {
		return err
	}
	var data []LogData
	for _, l := range logs {
		data = append(data, LogData{Time: l.Timestamp.Format("15:04:05"), Content: l.Line, Stream: l.Stream})
	}
	return data
}

func (m LogsModel) View() string {
	if m.Width == 0 {
		return ""
	}

	switch m.Mode {
	case LogsModeView:
		return m.viewLogs()
	default:
		return m.viewSelect()
	}
}

func (m LogsModel) viewSelect() string {
	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.Header("LOGS", w) + "\n\n")

	b.WriteString(components.Section("SELECT DEPLOYMENT", w) + "\n\n")

	var listContent strings.Builder
	if len(m.Deployments) == 0 {
		listContent.WriteString("  " + styles.MutedStyle.Render("No deployments found") + "\n")
		listContent.WriteString("  " + styles.SubtleStyle.Render("Deploy a repository first"))
	} else {
		for i, d := range m.Deployments {
			selected := i == m.Cursor
			ptr := "   "
			if selected {
				ptr = " " + styles.Pointer() + " "
			}

			icon := styles.SuccessStyle.Render(styles.IconSuccess)
			if d.Status == "failed" {
				icon = styles.ErrorStyle.Render(styles.IconError)
			} else if d.Status == "running" {
				icon = styles.PrimaryStyle.Render(styles.IconSpin)
			} else if d.Status == "pending" {
				icon = styles.WarningStyle.Render(styles.IconWarning)
			}

			nameStyle := styles.BrightStyle
			if selected {
				nameStyle = styles.PrimaryStyle
			}

			listContent.WriteString(fmt.Sprintf("%s%s  %s  %s  %s  %s\n",
				ptr,
				icon,
				nameStyle.Render(styles.Pad(styles.Trunc(d.Repo, 16), 16)),
				styles.MutedStyle.Render(styles.Pad(d.Branch, 10)),
				styles.Pad(d.Commit, 8),
				styles.MutedStyle.Render(d.Time)))
		}
	}
	b.WriteString(components.Wrap(listContent.String(), w) + "\n")

	content := b.String()
	lines := helper.CountLines(content)
	for i := 0; i < m.Height-lines-3; i++ {
		content += "\n"
	}

	content += "\n" + styles.Line(w) + "\n"
	content += components.Help([][]string{
		{"↑↓", "navigate"}, {"enter", "view logs"}, {"r", "refresh"}, {"esc", "back"},
	})

	return content
}

func (m LogsModel) viewLogs() string {
	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.Header("LOGS", w) + "\n\n")

	title := m.Repo
	if title == "" {
		title = "Deployment " + m.DeploymentID[:8]
	}
	if m.Commit != "" {
		title += " / " + m.Commit
	}

	b.WriteString(components.Section(title, w) + "\n\n")

	visibleLines := m.Height - 12
	if visibleLines < 1 {
		visibleLines = 1
	}

	var logContent strings.Builder
	if len(m.Logs) == 0 {
		logContent.WriteString("  " + styles.MutedStyle.Render("No logs available") + "\n")
		logContent.WriteString("  " + styles.SubtleStyle.Render("Waiting for deployment output..."))
	} else {
		endIdx := m.Offset + visibleLines
		if endIdx > len(m.Logs) {
			endIdx = len(m.Logs)
		}
		for i := m.Offset; i < endIdx; i++ {
			log := m.Logs[i]
			logContent.WriteString(components.LogLine(log.Time, log.Content, log.Stream, w) + "\n")
		}
	}
	b.WriteString(components.Wrap(logContent.String(), w) + "\n")

	content := b.String()
	lines := helper.CountLines(content)
	for i := 0; i < m.Height-lines-3; i++ {
		content += "\n"
	}

	content += "\n" + styles.Line(w) + "\n"

	followStatus := styles.MutedStyle.Render("follow: off")
	if m.AutoFollow {
		followStatus = styles.SuccessStyle.Render("follow: on")
	}

	content += components.Help([][]string{
		{"↑↓", "scroll"}, {"g", "top"}, {"G", "bottom"}, {"f", "toggle follow"}, {"r", "refresh"}, {"esc", "back"},
	})
	content += "   " + followStatus

	return content
}
