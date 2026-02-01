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
	"github.com/urustack/uruflow/internal/api"
	"github.com/urustack/uruflow/internal/tcp/protocol"
	"github.com/urustack/uruflow/internal/tui/components"
	"github.com/urustack/uruflow/internal/tui/styles"
	"github.com/urustack/uruflow/pkg/helper"
)

type ContainerLogsMsg protocol.ContainerLogsDataPayload

type ContainerLogsModel struct {
	Server        *api.Server
	Width         int
	Height        int
	AgentID       string
	AgentName     string
	ContainerID   string
	ContainerName string
	Logs          []LogData
	Offset        int
	AutoFollow    bool
	Mode          int
	Containers    []ContainerData
	Cursor        int
}

func NewContainerLogsModel(server *api.Server) ContainerLogsModel {
	return ContainerLogsModel{
		Server:     server,
		AutoFollow: true,
		Logs:       []LogData{},
		Mode:       0,
	}
}

func (m ContainerLogsModel) Init() tea.Cmd {
	return nil
}

func (m *ContainerLogsModel) SetAgent(agentData AgentData) {
	m.AgentID = agentData.ID
	m.AgentName = agentData.Name
	m.Containers = agentData.Containers
	m.Mode = 0
	m.Cursor = 0

	if len(m.Containers) == 1 {
		m.SetContainer(m.Containers[0].Name, m.Containers[0].Name)
	}
}

func (m *ContainerLogsModel) SetContainer(id, name string) {
	m.ContainerID = id
	m.ContainerName = name
	m.Logs = []LogData{}
	m.Offset = 0
	m.AutoFollow = true
	m.Mode = 1

	m.Server.GetTCPServer().StreamContainerLogs(m.AgentID, m.ContainerID, 100, true)
}

func (m *ContainerLogsModel) StopStream() {
	if m.AgentID != "" && m.ContainerID != "" {
		m.Server.GetTCPServer().StopContainerLogs(m.AgentID, m.ContainerID)
	}
}

func (m ContainerLogsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.Mode == 0 {
			switch msg.String() {
			case "up", "k":
				if m.Cursor > 0 {
					m.Cursor--
				}
			case "down", "j":
				if m.Cursor < len(m.Containers)-1 {
					m.Cursor++
				}
			case "enter":
				if len(m.Containers) > 0 {
					c := m.Containers[m.Cursor]
					m.SetContainer(c.Name, c.Name)
					return m, nil
				}
			}
		} else {
			switch msg.String() {
			case "esc":
				m.StopStream()
				m.Mode = 0
				return m, nil
			case "up", "k":
				if m.Offset > 0 {
					m.Offset--
					m.AutoFollow = false
				}
			case "down", "j":
				visibleLines := m.Height - 12
				if visibleLines < 1 {
					visibleLines = 1
				}
				maxOffset := len(m.Logs) - visibleLines
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
				visibleLines := m.Height - 12
				maxOffset := len(m.Logs) - visibleLines
				if maxOffset < 0 {
					maxOffset = 0
				}
				m.Offset = maxOffset
				m.AutoFollow = true
			case "f":
				m.AutoFollow = !m.AutoFollow
				if m.AutoFollow {
					visibleLines := m.Height - 12
					maxOffset := len(m.Logs) - visibleLines
					if maxOffset < 0 {
						maxOffset = 0
					}
					m.Offset = maxOffset
				}
			case "c":
				m.Logs = []LogData{}
				m.Offset = 0
			}
		}

	case ContainerLogsMsg:
		if m.Mode == 1 && msg.ContainerID == m.ContainerID {
			timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")

			newLog := LogData{
				Time:    timestamp,
				Content: msg.Line,
				Stream:  msg.Stream,
			}
			m.Logs = append(m.Logs, newLog)
			if len(m.Logs) > 2000 {
				m.Logs = m.Logs[1:]
				if m.Offset > 0 {
					m.Offset--
				}
			}

			if m.AutoFollow {
				visibleLines := m.Height - 12
				maxOffset := len(m.Logs) - visibleLines
				if maxOffset < 0 {
					maxOffset = 0
				}
				m.Offset = maxOffset
			}
		}
	}

	return m, cmd
}

func (m ContainerLogsModel) View() string {
	if m.Width == 0 {
		return ""
	}

	if m.Mode == 0 {
		return m.viewSelect()
	}
	return m.viewLogs()
}

func (m ContainerLogsModel) viewSelect() string {
	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.ViewHeader(w, "Dashboard", "Agents", m.AgentName, "Containers") + "\n\n")
	b.WriteString(components.Section(m.AgentName, w) + "\n\n")

	var listContent strings.Builder
	if len(m.Containers) == 0 {
		listContent.WriteString("  " + styles.MutedStyle.Render("No containers found"))
	} else {
		for i, c := range m.Containers {
			ptr := "   "
			if i == m.Cursor {
				ptr = " " + styles.Pointer() + " "
			}

			status := styles.MutedStyle.Render("stopped")
			if c.Running {
				status = styles.SuccessStyle.Render("running")
			}

			name := styles.BrightStyle.Render(c.Name)
			if i == m.Cursor {
				name = styles.PrimaryStyle.Render(c.Name)
			}

			listContent.WriteString(fmt.Sprintf("%s%s  %s\n", ptr, styles.Pad(name, 30), status))
		}
	}
	b.WriteString(components.Wrap(listContent.String(), w) + "\n")

	content := b.String()
	lines := helper.CountLines(content)
	for i := 0; i < m.Height-lines-3; i++ {
		content += "\n"
	}
	content += "\n" + styles.Line(w) + "\n"
	content += components.Help([][]string{{"↑↓", "select"}, {"enter", "view logs"}, {"esc", "back"}})
	return content
}

func (m ContainerLogsModel) viewLogs() string {
	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.ViewHeader(w, "Dashboard", "Agents", m.AgentName, m.ContainerName) + "\n\n")

	headerText := fmt.Sprintf("%s / %s", m.AgentName, m.ContainerName)
	b.WriteString(components.Section(headerText, w) + "\n\n")

	visibleLines := m.Height - 12
	if visibleLines < 1 {
		visibleLines = 1
	}

	var logContent strings.Builder
	if len(m.Logs) == 0 {
		logContent.WriteString("  " + styles.MutedStyle.Render("No logs available") + "\n")
		logContent.WriteString("  " + styles.SubtleStyle.Render("Waiting for container output..."))
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
		{"↑↓", "scroll"}, {"f", "toggle follow"}, {"c", "clear"}, {"esc", "back"},
	})
	content += "   " + followStatus

	return content
}
