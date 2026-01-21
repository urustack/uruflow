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
	"github.com/urustack/uruflow/internal/config"
	"github.com/urustack/uruflow/internal/models"
	"github.com/urustack/uruflow/internal/storage"
	"github.com/urustack/uruflow/internal/tui/components"
	"github.com/urustack/uruflow/internal/tui/styles"
	"github.com/urustack/uruflow/pkg/helper"
)

type AgentMode int

const (
	AgentModeList AgentMode = iota
	AgentModeAdd
	AgentModeResult
)

type AgentResultMsg struct {
	Success bool
	Name    string
	ID      string
	Token   string
	Error   error
}

type AgentsModel struct {
	store    storage.Store
	cfg      *config.Config
	cfgPath  string
	Width    int
	Height   int
	Agents   []AgentData
	Cursor   int
	Expanded bool
	Mode     AgentMode
	Input    string
	Result   AgentAddResult
	err      error
}

type AgentAddResult struct {
	Name  string
	ID    string
	Token string
}

func NewAgentsModel(store storage.Store, cfg *config.Config, cfgPath string) AgentsModel {
	return AgentsModel{store: store, cfg: cfg, cfgPath: cfgPath, Mode: AgentModeList}
}

func (m AgentsModel) Init() tea.Cmd {
	return m.fetchAgents
}

func (m AgentsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.Mode {
		case AgentModeList:
			return m.updateList(msg)
		case AgentModeAdd:
			return m.updateAdd(msg)
		case AgentModeResult:
			return m.updateResult(msg)
		}
	case AgentResultMsg:
		if msg.Success {
			m.Result = AgentAddResult{Name: msg.Name, ID: msg.ID, Token: msg.Token}
			m.Mode = AgentModeResult
			m.err = nil
		} else {
			m.err = msg.Error
		}
		return m, m.fetchAgents
	case []AgentData:
		m.Agents = msg
		return m, nil
	case error:
		m.err = msg
		return m, nil
	}
	return m, nil
}

func (m AgentsModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < len(m.Agents)-1 {
			m.Cursor++
		}
	case "enter":
		m.Expanded = !m.Expanded
	case "+", "n":
		m.Mode = AgentModeAdd
		m.Input = ""
		m.err = nil
	case "-", "delete", "backspace":
		if len(m.Agents) > 0 {
			return m, m.deleteAgent(m.Agents[m.Cursor].ID)
		}
	case "r":
		return m, m.fetchAgents
	}
	return m, nil
}

func (m AgentsModel) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Mode = AgentModeList
		m.Input = ""
		m.err = nil
	case "enter":
		if m.Input != "" {
			return m, m.addAgent(m.Input)
		}
	case "backspace":
		if len(m.Input) > 0 {
			m.Input = m.Input[:len(m.Input)-1]
		}
	default:
		inputStr := msg.String()
		if len(m.Input)+len(inputStr) <= 30 {
			m.Input += inputStr
		}
	}
	return m, nil
}

func (m AgentsModel) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.Mode = AgentModeList
		return m, m.fetchAgents
	}
	return m, nil
}

func (m AgentsModel) addAgent(name string) tea.Cmd {
	return func() tea.Msg {
		id, token, err := m.cfg.AddAgent(name)
		if err != nil {
			return AgentResultMsg{Success: false, Error: err}
		}
		if err := m.cfg.Save(m.cfgPath); err != nil {
			return AgentResultMsg{Success: false, Error: err}
		}
		agent := &models.Agent{
			ID: id, Name: name, Token: token, Status: models.AgentOffline, RegisteredAt: time.Now(),
		}
		m.store.CreateAgent(agent)
		return AgentResultMsg{Success: true, Name: name, ID: id, Token: token}
	}
}

func (m AgentsModel) deleteAgent(id string) tea.Cmd {
	return func() tea.Msg {
		m.cfg.RemoveAgent(id)
		m.cfg.Save(m.cfgPath)
		m.store.DeleteAgent(id)
		return m.fetchAgents()
	}
}

func (m AgentsModel) fetchAgents() tea.Msg {
	agents, err := m.store.GetAllAgents()
	if err != nil {
		return err
	}
	var data []AgentData
	for _, a := range agents {
		containers, _ := m.store.GetContainersByAgent(a.ID)
		containerData := make([]ContainerData, len(containers))
		for i, c := range containers {
			containerData[i] = ContainerData{
				Name: c.Name, Running: c.Status == "running", Healthy: c.Health == "healthy",
				CPU: c.CPUPercent, Memory: fmt.Sprintf("%dMB", c.MemoryUsage/1024/1024),
			}
		}
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
		data = append(data, AgentData{
			ID: a.ID, Name: a.Name, Host: a.Host, Version: a.Version, Uptime: uptime,
			Online: a.Status == "online", CPU: cpu, Memory: mem, Disk: disk, Containers: containerData,
		})
	}
	return data
}

func (m AgentsModel) View() string {
	if m.Width == 0 {
		return ""
	}
	switch m.Mode {
	case AgentModeAdd:
		return m.viewAdd()
	case AgentModeResult:
		return m.viewResult()
	default:
		return m.viewList()
	}
}

func (m AgentsModel) viewList() string {
	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.Header("AGENTS", w) + "\n\n")

	online, offline := 0, 0
	for _, a := range m.Agents {
		if a.Online {
			online++
		} else {
			offline++
		}
	}

	b.WriteString(components.Section("OVERVIEW", w) + "\n\n")

	var statsContent strings.Builder
	statsContent.WriteString(components.Stats(online, offline, len(m.Agents)))
	b.WriteString(components.Wrap(statsContent.String(), w) + "\n\n")

	if m.err != nil {
		b.WriteString(components.MsgError(m.err.Error(), w) + "\n\n")
	}

	b.WriteString(components.Section("AGENT LIST", w) + "\n\n")

	var listContent strings.Builder
	if len(m.Agents) == 0 {
		listContent.WriteString("  " + styles.MutedStyle.Render("No agents registered") + "\n")
		listContent.WriteString("  " + styles.SubtleStyle.Render("Press '+' to add your first agent"))
	} else {
		for i, a := range m.Agents {
			selected := i == m.Cursor
			if selected && m.Expanded {
				card := components.AgentCardData{
					Name: a.Name, Host: a.Host, Version: a.Version, Online: a.Online,
					CPU: a.CPU, Memory: a.Memory, Disk: a.Disk, Selected: true,
					Containers: make([]components.ContainerInfo, len(a.Containers)),
				}
				for j, c := range a.Containers {
					card.Containers[j] = components.ContainerInfo{
						Name: c.Name, Running: c.Running, Healthy: c.Healthy, CPU: c.CPU, Memory: c.Memory,
					}
				}
				listContent.WriteString(components.AgentCard(card, w-8) + "\n")
			} else {
				listContent.WriteString(components.AgentRow(a.Name, a.Online, a.CPU, a.Memory, a.Disk, a.Uptime, selected, w) + "\n")
			}
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
		{"↑↓", "navigate"}, {"enter", "expand"}, {"+", "add"}, {"-", "remove"}, {"r", "refresh"}, {"esc", "back"},
	})

	return content
}

func (m AgentsModel) viewAdd() string {
	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.Header("ADD AGENT", w) + "\n\n")

	b.WriteString(components.Section("AGENT NAME", w) + "\n\n")

	var formContent strings.Builder
	formContent.WriteString(components.Input("Enter a name to identify this agent", m.Input, true, w-8))
	if m.err != nil {
		formContent.WriteString("\n\n" + styles.ErrorStyle.Render(styles.IconError) + "  " + styles.ErrorStyle.Render(m.err.Error()))
	}
	b.WriteString(components.Wrap(formContent.String(), w) + "\n")

	content := b.String()
	lines := helper.CountLines(content)
	for i := 0; i < m.Height-lines-3; i++ {
		content += "\n"
	}

	content += "\n" + styles.Line(w) + "\n"
	content += components.Help([][]string{{"enter", "create"}, {"esc", "cancel"}})

	return content
}

func (m AgentsModel) viewResult() string {
	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.Header("AGENT CREATED", w) + "\n\n")

	b.WriteString(components.MsgSuccess("Agent created successfully", w) + "\n\n")

	b.WriteString(components.Section("AGENT DETAILS", w) + "\n\n")
	b.WriteString(components.Card(m.Result.Name, []components.CardLine{
		{Label: "ID", Value: styles.MutedStyle.Render(m.Result.ID)},
	}, false, w) + "\n\n")

	b.WriteString(components.Section("TOKEN", w) + "\n\n")
	b.WriteString(components.Token(m.Result.Token, w) + "\n\n")
	b.WriteString(components.MsgWarning("Save this token! It won't be shown again.", w) + "\n\n")

	b.WriteString(components.Section("NEXT STEPS", w) + "\n\n")
	var stepsContent strings.Builder
	stepsContent.WriteString("  1. Install agent on your server:\n")
	stepsContent.WriteString("     " + styles.PrimaryStyle.Render("curl -sSL https://uruflow.io/install | sh") + "\n\n")
	stepsContent.WriteString("  2. Connect with:\n")
	tok := m.Result.Token
	if len(tok) > 16 {
		tok = tok[:16] + "..."
	}
	stepsContent.WriteString("     " + styles.PrimaryStyle.Render(fmt.Sprintf("uruflow-server-agent connect --token %s", tok)))
	b.WriteString(components.Wrap(stepsContent.String(), w) + "\n")

	content := b.String()
	lines := helper.CountLines(content)
	for i := 0; i < m.Height-lines-3; i++ {
		content += "\n"
	}

	content += "\n" + styles.Line(w) + "\n"
	content += components.Help([][]string{{"enter", "done"}, {"esc", "back"}})

	return content
}
