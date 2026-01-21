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

type AlertsModel struct {
	store    storage.Store
	Width    int
	Height   int
	Active   []AlertData
	Recent   []AlertData
	Cursor   int
	Expanded bool
	err      error
}

func NewAlertsModel(store storage.Store) AlertsModel {
	return AlertsModel{store: store}
}

func (m AlertsModel) Init() tea.Cmd {
	return m.fetchAlerts
}

func (m AlertsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			total := len(m.Active) + len(m.Recent)
			if m.Cursor < total-1 {
				m.Cursor++
			}
		case "x":
			if m.Cursor < len(m.Active) {
				return m, m.resolveAlert(m.Active[m.Cursor].ID)
			}
		case "e":
			m.Expanded = !m.Expanded
		case "r":
			return m, m.fetchAlerts
		}
	case alertsMsg:
		m.Active = msg.Active
		m.Recent = msg.Recent
		return m, nil
	case error:
		m.err = msg
		return m, nil
	}
	return m, nil
}

type alertsMsg struct {
	Active []AlertData
	Recent []AlertData
}

func (m AlertsModel) resolveAlert(id string) tea.Cmd {
	return func() tea.Msg {
		if err := m.store.ResolveAlert(id); err != nil {
			return err
		}
		return m.fetchAlerts()
	}
}

func (m AlertsModel) fetchAlerts() tea.Msg {
	active, err := m.store.GetActiveAlerts()
	if err != nil {
		return err
	}
	recent, err := m.store.GetRecentAlerts(24)
	if err != nil {
		return err
	}

	var activeData []AlertData
	for _, a := range active {
		activeData = append(activeData, AlertData{
			ID: a.ID, Type: a.Type, Agent: a.AgentName, Message: a.Message,
			Time:   time.Since(a.CreatedAt).Round(time.Second).String() + " ago",
			Active: true, Severity: string(a.Severity),
		})
	}

	var recentData []AlertData
	for _, a := range recent {
		if a.Resolved {
			recentData = append(recentData, AlertData{
				ID: a.ID, Type: a.Type, Agent: a.AgentName, Message: a.Message,
				Time: a.CreatedAt.Format("15:04"), Active: false, Severity: string(a.Severity),
			})
		}
	}

	return alertsMsg{Active: activeData, Recent: recentData}
}

func (m AlertsModel) View() string {
	if m.Width == 0 {
		return ""
	}

	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.Header("ALERTS", w) + "\n\n")

	b.WriteString(components.Section("STATUS", w) + "\n\n")
	var statusContent strings.Builder
	if len(m.Active) > 0 {
		statusContent.WriteString(fmt.Sprintf("  %s %s",
			styles.WarningStyle.Render(fmt.Sprintf("%d", len(m.Active))),
			styles.WarningStyle.Render("active alerts")))
	} else {
		statusContent.WriteString("  " + styles.SuccessStyle.Render(styles.IconSuccess) + "  " +
			styles.SuccessStyle.Render("All systems operational"))
	}
	b.WriteString(components.Wrap(statusContent.String(), w) + "\n\n")

	if m.err != nil {
		b.WriteString(components.MsgError(m.err.Error(), w) + "\n\n")
	}

	b.WriteString(components.Section("ACTIVE ALERTS", w) + "\n\n")
	var activeContent strings.Builder
	if len(m.Active) == 0 {
		activeContent.WriteString("  " + styles.MutedStyle.Render("No active alerts"))
	} else {
		for i, a := range m.Active {
			selected := i == m.Cursor
			if selected && m.Expanded {
				card := components.AlertCardData{
					Type: a.Type, Agent: a.Agent, Message: a.Message,
					Time: a.Time, Severity: a.Severity, Selected: true,
				}
				activeContent.WriteString(components.AlertCard(card, w-8) + "\n")
			} else {
				ptr := "   "
				if selected {
					ptr = " " + styles.Pointer() + " "
				}
				icon := styles.WarningStyle.Render(styles.IconWarning)
				if a.Severity == "critical" {
					icon = styles.ErrorStyle.Render(styles.IconError)
				}

				typeStyle := styles.BrightStyle
				if selected {
					typeStyle = styles.PrimaryStyle
				}

				activeContent.WriteString(fmt.Sprintf("%s%s  %s  %s  %s  %s\n",
					ptr,
					icon,
					typeStyle.Render(styles.Pad(styles.Trunc(a.Type, 12), 12)),
					styles.Pad(styles.Trunc(a.Agent, 14), 14),
					styles.MutedStyle.Render(styles.Pad(styles.Trunc(a.Message, 24), 24)),
					styles.MutedStyle.Render(a.Time)))
			}
		}
	}
	b.WriteString(components.Wrap(activeContent.String(), w) + "\n\n")

	b.WriteString(components.Section("RESOLVED (LAST 24H)", w) + "\n\n")
	var recentContent strings.Builder
	if len(m.Recent) == 0 {
		recentContent.WriteString("  " + styles.MutedStyle.Render("No recent alerts"))
	} else {
		for i, a := range m.Recent {
			selected := (i + len(m.Active)) == m.Cursor
			ptr := "   "
			if selected {
				ptr = " " + styles.Pointer() + " "
			}
			icon := styles.SuccessStyle.Render(styles.IconSuccess)

			typeStyle := styles.MutedStyle
			if selected {
				typeStyle = styles.PrimaryStyle
			}

			recentContent.WriteString(fmt.Sprintf("%s%s  %s  %s  %s  %s\n",
				ptr,
				icon,
				typeStyle.Render(styles.Pad(styles.Trunc(a.Type, 12), 12)),
				styles.MutedStyle.Render(styles.Pad(styles.Trunc(a.Agent, 14), 14)),
				styles.MutedStyle.Render(styles.Pad(styles.Trunc(a.Message, 24), 24)),
				styles.MutedStyle.Render(a.Time)))
		}
	}
	b.WriteString(components.Wrap(recentContent.String(), w) + "\n")

	content := b.String()
	lines := helper.CountLines(content)
	for i := 0; i < m.Height-lines-3; i++ {
		content += "\n"
	}

	content += "\n" + styles.Line(w) + "\n"
	content += components.Help([][]string{
		{"↑↓", "navigate"}, {"e", "expand"}, {"x", "resolve"}, {"r", "refresh"}, {"esc", "back"},
	})

	return content
}
