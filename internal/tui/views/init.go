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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urustack/uruflow/internal/config"
	"github.com/urustack/uruflow/internal/tui/components"
	"github.com/urustack/uruflow/internal/tui/styles"
	"github.com/urustack/uruflow/pkg/helper"
)

type InitModel struct {
	Width        int
	Height       int
	Step         int
	TotalSteps   int
	HTTPPort     string
	TCPPort      string
	Secret       string
	SecretOption int
	DataDir      string
	FocusedField int
	Done         bool
	Error        string
}

func NewInitModel() InitModel {
	return InitModel{
		Step: 0, TotalSteps: 4, HTTPPort: "9000", TCPPort: "9001",
		Secret: helper.GenerateSecret(), SecretOption: 0, DataDir: "/var/lib/uruflow-server", FocusedField: 0,
	}
}

func (m InitModel) Init() tea.Cmd {
	return nil
}

func (m InitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.Step == 2 {
				if m.SecretOption > 0 {
					m.SecretOption--
				}
			} else if m.FocusedField > 0 {
				m.FocusedField--
			}
		case "down", "j":
			if m.Step == 2 {
				if m.SecretOption < 1 {
					m.SecretOption++
				}
			} else if m.Step == 1 && m.FocusedField < 1 {
				m.FocusedField++
			}
		case "tab":
			if m.Step == 1 {
				m.FocusedField = (m.FocusedField + 1) % 2
			}
		case "enter":
			if m.Done {
				return m, tea.Quit
			}
			if m.Step < m.TotalSteps {
				m.Step++
				m.FocusedField = 0
			} else {
				m.Done = true
				m.saveConfig()
			}
		case "backspace":
			if m.Step == 1 {
				if m.FocusedField == 0 && len(m.HTTPPort) > 0 {
					m.HTTPPort = m.HTTPPort[:len(m.HTTPPort)-1]
				} else if m.FocusedField == 1 && len(m.TCPPort) > 0 {
					m.TCPPort = m.TCPPort[:len(m.TCPPort)-1]
				}
			} else if m.Step == 3 && len(m.DataDir) > 0 {
				m.DataDir = m.DataDir[:len(m.DataDir)-1]
			}
		default:
			if len(msg.String()) == 1 {
				char := msg.String()
				if m.Step == 1 {
					if char >= "0" && char <= "9" {
						if m.FocusedField == 0 && len(m.HTTPPort) < 5 {
							m.HTTPPort += char
						} else if m.FocusedField == 1 && len(m.TCPPort) < 5 {
							m.TCPPort += char
						}
					}
				} else if m.Step == 3 {
					m.DataDir += char
				}
			}
		}
	}
	return m, nil
}

func (m *InitModel) saveConfig() {
	cfg := config.Default()
	if m.HTTPPort != "" {
		var port int
		fmt.Sscanf(m.HTTPPort, "%d", &port)
		cfg.Server.HTTPPort = port
	}
	if m.TCPPort != "" {
		var port int
		fmt.Sscanf(m.TCPPort, "%d", &port)
		cfg.Server.TCPPort = port
	}
	cfg.Webhook.Secret = m.Secret
	cfg.Server.DataDir = m.DataDir
	if err := cfg.Save(config.DefaultConfigPath); err != nil {
		m.Error = err.Error()
	}
}

func (m InitModel) View() string {
	if m.Width == 0 {
		return ""
	}

	var b strings.Builder
	w := m.Width

	if m.Step == 0 {
		b.WriteString("\n\n")
		b.WriteString(components.CenteredLogo(w))
		b.WriteString("\n\n\n")

		welcome := "Welcome to UruFlow"
		b.WriteString(styles.Center(styles.TitleStyle.Render(welcome), w) + "\n\n")

		desc1 := "This wizard will help you set up your deployment server."
		b.WriteString(styles.Center(styles.MutedStyle.Render(desc1), w) + "\n")
		desc2 := "It takes about 2 minutes."
		b.WriteString(styles.Center(styles.MutedStyle.Render(desc2), w) + "\n")
	} else {
		b.WriteString("\n")
		b.WriteString(components.Header("SETUP", w) + "\n\n")

		b.WriteString(components.Section(fmt.Sprintf("STEP %d OF %d", m.Step, m.TotalSteps), w) + "\n\n")

		switch m.Step {
		case 1:
			b.WriteString(m.viewStep1(w))
		case 2:
			b.WriteString(m.viewStep2(w))
		case 3:
			b.WriteString(m.viewStep3(w))
		case 4:
			b.WriteString(m.viewStep4(w))
		}
	}

	content := b.String()
	lines := helper.CountLines(content)
	for i := 0; i < m.Height-lines-3; i++ {
		content += "\n"
	}

	content += "\n" + styles.Line(w) + "\n"
	if m.Done {
		content += components.Help([][]string{{"enter", "start"}})
	} else if m.Step == 0 {
		content += components.Help([][]string{{"enter", "begin setup"}, {"ctrl+c", "exit"}})
	} else {
		content += components.Help([][]string{{"↑↓", "navigate"}, {"tab", "next field"}, {"enter", "continue"}, {"ctrl+c", "exit"}})
	}

	return content
}

func (m InitModel) viewStep1(w int) string {
	var b strings.Builder

	var formContent strings.Builder
	formContent.WriteString(styles.SubtleStyle.Render("Configure the ports for your server") + "\n\n")

	httpPtr := "   "
	httpLabel := styles.SubtleStyle.Render("HTTP PORT")
	httpValue := m.HTTPPort
	if m.FocusedField == 0 {
		httpPtr = " " + styles.Pointer() + " "
		httpLabel = styles.PrimaryStyle.Render("HTTP PORT")
		httpValue = m.HTTPPort + styles.PrimaryStyle.Render("▎")
	}
	formContent.WriteString(fmt.Sprintf("%s%s    %s\n\n", httpPtr, styles.Pad(httpLabel, 12), httpValue))

	tcpPtr := "   "
	tcpLabel := styles.SubtleStyle.Render("TCP PORT")
	tcpValue := m.TCPPort
	if m.FocusedField == 1 {
		tcpPtr = " " + styles.Pointer() + " "
		tcpLabel = styles.PrimaryStyle.Render("TCP PORT")
		tcpValue = m.TCPPort + styles.PrimaryStyle.Render("▎")
	}
	formContent.WriteString(fmt.Sprintf("%s%s     %s\n\n", tcpPtr, styles.Pad(tcpLabel, 12), tcpValue))

	formContent.WriteString(styles.MutedStyle.Render("  HTTP: webhooks and API") + "\n")
	formContent.WriteString(styles.MutedStyle.Render("  TCP: agent connections"))

	b.WriteString(components.Wrap(formContent.String(), w) + "\n")
	return b.String()
}

func (m InitModel) viewStep2(w int) string {
	var b strings.Builder

	var formContent strings.Builder
	formContent.WriteString(styles.SubtleStyle.Render("Used to verify GitHub/GitLab webhooks") + "\n\n")

	opts := []string{"Generate random secret", "Enter custom secret"}
	formContent.WriteString(components.Select(opts, m.SecretOption) + "\n")

	formContent.WriteString(styles.SubtleStyle.Render("Generated:") + "\n")
	formContent.WriteString(styles.PrimaryStyle.Render(m.Secret))

	b.WriteString(components.Wrap(formContent.String(), w) + "\n")
	return b.String()
}

func (m InitModel) viewStep3(w int) string {
	var b strings.Builder

	var formContent strings.Builder
	formContent.WriteString(styles.SubtleStyle.Render("Where UruFlow stores data") + "\n\n")
	formContent.WriteString(components.Input("Path", m.DataDir, true, w-8))

	b.WriteString(components.Wrap(formContent.String(), w) + "\n")
	return b.String()
}

func (m InitModel) viewStep4(w int) string {
	var b strings.Builder

	if m.Done {
		if m.Error != "" {
			b.WriteString(components.MsgError(m.Error, w) + "\n\n")
		} else {
			b.WriteString(components.MsgSuccess("Configuration saved successfully", w) + "\n\n")
		}

		b.WriteString(components.Section("NEXT STEPS", w) + "\n\n")

		var stepsContent strings.Builder
		stepsContent.WriteString("  1. Add an agent in the Agents view\n")
		stepsContent.WriteString("  2. Install agent on target servers\n")
		stepsContent.WriteString("  3. Add repositories and deploy!")
		b.WriteString(components.Wrap(stepsContent.String(), w) + "\n")
	} else {
		var reviewContent strings.Builder
		reviewContent.WriteString(styles.SubtleStyle.Render("Review your configuration") + "\n\n")

		reviewContent.WriteString(styles.SubtleStyle.Render(styles.Pad("HTTP Port", 12)) + " " + m.HTTPPort + "\n")
		reviewContent.WriteString(styles.SubtleStyle.Render(styles.Pad("TCP Port", 12)) + " " + m.TCPPort + "\n")
		secret := m.Secret
		if len(secret) > 16 {
			secret = secret[:16] + "..."
		}
		reviewContent.WriteString(styles.SubtleStyle.Render(styles.Pad("Secret", 12)) + " " + styles.MutedStyle.Render(secret) + "\n")
		reviewContent.WriteString(styles.SubtleStyle.Render(styles.Pad("Data Dir", 12)) + " " + m.DataDir + "\n\n")

		reviewContent.WriteString(styles.MutedStyle.Render("Press Enter to save configuration"))

		b.WriteString(components.Wrap(reviewContent.String(), w) + "\n")
	}

	return b.String()
}
