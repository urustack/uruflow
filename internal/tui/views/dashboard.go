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
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urustack/uruflow/internal/storage"
	"github.com/urustack/uruflow/internal/tui/components"
	"github.com/urustack/uruflow/internal/tui/styles"
	"github.com/urustack/uruflow/pkg/helper"
)

type DashboardModel struct {
	store        storage.Store
	Width        int
	Height       int
	Agents       []AgentData
	Deployments  []DeploymentData
	Alerts       []AlertData
	Message      string
	MessageType  string
	Loading      bool
	SpinnerFrame int
	ShowHelp     bool
	err          error
}

func NewDashboardModel(store storage.Store) DashboardModel {
	return DashboardModel{store: store}
}

func (m *DashboardModel) SetMessage(msg, t string) {
	m.Message = msg
	m.MessageType = t
}

func (m *DashboardModel) ClearMessage() {
	m.Message = ""
	m.MessageType = ""
}

func (m DashboardModel) Init() tea.Cmd {
	m.Loading = true
	return tea.Batch(m.fetchData, m.tick, m.spinnerTick)
}

func (m DashboardModel) spinnerTick() tea.Msg {
	time.Sleep(80 * time.Millisecond)
	return SpinnerTickMsg{}
}

type SpinnerTickMsg struct{}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?":
			m.ShowHelp = !m.ShowHelp
		}
	case SpinnerTickMsg:
		m.SpinnerFrame++
		if m.Loading {
			return m, m.spinnerTick
		}
	case TickMsg:
		m.Loading = true
		return m, tea.Batch(m.fetchData, m.tick, m.spinnerTick)
	case DataMsg:
		m.Agents = msg.Agents
		m.Deployments = msg.Deployments
		m.Alerts = msg.Alerts
		m.Loading = false
		return m, nil
	case error:
		m.err = msg
		m.Loading = false
		return m, nil
	}
	return m, nil
}

func (m DashboardModel) tick() tea.Msg {
	time.Sleep(2 * time.Second)
	return TickMsg(time.Now())
}

func (m DashboardModel) fetchData() tea.Msg {
	agents, err := m.store.GetAllAgents()
	if err != nil {
		return err
	}
	deployments, err := m.store.GetRecentDeployments(5)
	if err != nil {
		return err
	}
	alerts, err := m.store.GetActiveAlerts()
	if err != nil {
		return err
	}

	var agentData []AgentData
	for _, a := range agents {
		uptime := time.Since(a.LastHeartbeat).Round(time.Second).String()
		if a.Status == "offline" {
			uptime = a.LastHeartbeat.Format("2006-01-02 15:04")
		}
		cpu, mem, disk := 0.0, 0.0, 0.0
		if a.Metrics != nil {
			cpu = a.Metrics.CPUPercent
			mem = a.Metrics.MemoryPercent
			disk = a.Metrics.DiskPercent
		}
		containers, _ := m.store.GetContainersByAgent(a.ID)
		agentData = append(agentData, AgentData{
			ID: a.ID, Name: a.Name, Host: a.Host, Online: a.Status == "online",
			Uptime: uptime, CPU: cpu, Memory: mem, Disk: disk,
			Containers: make([]ContainerData, len(containers)),
		})
	}

	var deployData []DeploymentData
	for _, d := range deployments {
		commit := d.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		deployData = append(deployData, DeploymentData{
			ID: d.ID, Repo: d.Repository, Branch: d.Branch, Commit: commit,
			Agent: d.AgentName, Status: string(d.Status),
			Time: time.Since(d.StartedAt).Round(time.Second).String() + " ago",
		})
	}

	var alertData []AlertData
	for _, a := range alerts {
		alertData = append(alertData, AlertData{
			ID: a.ID, Type: a.Type, Agent: a.AgentName, Message: a.Message,
			Time: time.Since(a.CreatedAt).Round(time.Second).String(), Active: true,
			Severity: string(a.Severity),
		})
	}

	return DataMsg{Agents: agentData, Deployments: deployData, Alerts: alertData}
}

func (m DashboardModel) View() string {
	if m.Width == 0 {
		return ""
	}

	var b strings.Builder
	w := m.Width
	b.WriteString("\n")
	b.WriteString(components.ViewHeader(w, "Dashboard") + "\n")
	online, offline := 0, 0
	for _, a := range m.Agents {
		if a.Online {
			online++
		} else {
			offline++
		}
	}
	b.WriteString(components.StatusBar(online, offline, len(m.Alerts), w) + "\n\n")
	if m.Message != "" {
		switch m.MessageType {
		case "success":
			b.WriteString(components.MsgSuccess(m.Message, w) + "\n\n")
		case "error":
			b.WriteString(components.MsgError(m.Message, w) + "\n\n")
		case "warning":
			b.WriteString(components.MsgWarning(m.Message, w) + "\n\n")
		default:
			b.WriteString(components.MsgInfo(m.Message, w) + "\n\n")
		}
	}

	if m.Loading && len(m.Agents) == 0 {
		b.WriteString(components.Loading(m.SpinnerFrame, "Loading data...") + "\n\n")
	}

	b.WriteString(components.Section("AGENTS", w) + "\n\n")
	var agentContent strings.Builder
	if len(m.Agents) == 0 && !m.Loading {
		agentContent.WriteString("  " + styles.MutedStyle.Render("No agents registered. Press 'a' to add one."))
	} else if len(m.Agents) > 0 {
		agentContent.WriteString(components.AgentHeader(w) + "\n")
		agentContent.WriteString("  " + styles.Line(w-8) + "\n")
		for _, a := range m.Agents {
			agentContent.WriteString(components.AgentRow(a.Name, a.Online, a.CPU, a.Memory, a.Disk, a.Uptime, false, w) + "\n")
		}
	}
	if agentContent.Len() > 0 {
		b.WriteString(components.Wrap(agentContent.String(), w) + "\n\n")
	}

	b.WriteString(components.Section("RECENT DEPLOYMENTS", w) + "\n\n")

	var deployContent strings.Builder
	if len(m.Deployments) == 0 {
		deployContent.WriteString("  " + styles.MutedStyle.Render("No deployments yet"))
	} else {
		for _, d := range m.Deployments {
			icon := styles.SuccessStyle.Render(styles.IconSuccess)
			if d.Status == "failed" {
				icon = styles.ErrorStyle.Render(styles.IconError)
			} else if d.Status == "running" {
				icon = styles.PrimaryStyle.Render(styles.IconSpin)
			} else if d.Status == "pending" {
				icon = styles.WarningStyle.Render(styles.IconWarning)
			}
			deployContent.WriteString(components.DeployRow(icon, d.Repo, d.Branch, d.Commit, d.Agent, d.Time, w) + "\n")
		}
	}
	b.WriteString(components.Wrap(deployContent.String(), w) + "\n\n")
	b.WriteString(components.Section("ALERTS", w) + "\n\n")

	var alertContent strings.Builder
	if len(m.Alerts) == 0 {
		alertContent.WriteString("  " + styles.SuccessStyle.Render(styles.IconSuccess) + "  " + styles.MutedStyle.Render("All systems operational"))
	} else {
		for _, a := range m.Alerts {
			icon := styles.WarningStyle.Render(styles.IconWarning)
			if a.Severity == "critical" {
				icon = styles.ErrorStyle.Render(styles.IconError)
			}
			alertContent.WriteString(components.AlertRow(icon, a.Type, a.Agent, a.Message, a.Time, false, w) + "\n")
		}
	}
	b.WriteString(components.Wrap(alertContent.String(), w) + "\n")

	content := b.String()
	lines := helper.CountLines(content)
	for i := 0; i < m.Height-lines-3; i++ {
		content += "\n"
	}

	content += "\n" + styles.Line(w) + "\n"
	helpItems := [][]string{
		{"a", "agents"}, {"r", "repos"}, {"x", "alerts"}, {"l", "history"}, {"d", "deploy"}, {"tab", "cycle"}, {"?", "help"}, {"q", "quit"},
	}
	content += components.Help(helpItems)

	if m.Loading {
		content += "  " + components.LoadingInline(m.SpinnerFrame)
	}

	if m.ShowHelp {
		content += "\n\n" + styles.MutedStyle.Render("  Navigation: tab to cycle views, esc to return to dashboard")
		content += "\n" + styles.MutedStyle.Render("  Quick access: a=agents, r=repos, x=alerts, l=logs, d=deploy")
	}

	return content
}
