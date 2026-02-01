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

package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/urustack/uruflow/internal/tui/styles"
)

type DialogButton struct {
	Label    string
	Selected bool
	Primary  bool
}

type Dialog struct {
	Title    string
	Message  string
	Warning  string
	Selected int
	Visible  bool
}

func NewDialog(title, message, warning string) Dialog {
	return Dialog{
		Title:    title,
		Message:  message,
		Warning:  warning,
		Selected: 0,
		Visible:  true,
	}
}

func (d *Dialog) ToggleSelection() {
	if d.Selected == 0 {
		d.Selected = 1
	} else {
		d.Selected = 0
	}
}

func (d Dialog) IsConfirmed() bool {
	return d.Selected == 1
}

func ConfirmDialog(d Dialog, screenWidth, screenHeight int) string {
	if !d.Visible {
		return ""
	}

	dialogWidth := 44
	if dialogWidth > screenWidth-4 {
		dialogWidth = screenWidth - 4
	}

	var content strings.Builder

	content.WriteString(styles.TitleStyle.Render(d.Title) + "\n\n")

	content.WriteString(d.Message + "\n")

	if d.Warning != "" {
		content.WriteString(styles.MutedStyle.Render(d.Warning) + "\n")
	}

	content.WriteString("\n")

	noBtn := styles.MutedStyle.Render("[No]")
	yesBtn := styles.MutedStyle.Render("[Yes]")

	if d.Selected == 0 {
		noBtn = styles.PrimaryStyle.Bold(true).Render("[No]")
	} else {
		yesBtn = styles.ErrorStyle.Bold(true).Render("[Yes]")
	}

	content.WriteString(noBtn + "    " + yesBtn)

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(dialogWidth)

	dialogBox := dialogStyle.Render(content.String())

	dialogLines := strings.Split(dialogBox, "\n")
	dialogHeight := len(dialogLines)

	topPad := (screenHeight - dialogHeight) / 2
	if topPad < 0 {
		topPad = 0
	}

	leftPad := (screenWidth - lipgloss.Width(dialogLines[0])) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	var b strings.Builder

	for i := 0; i < topPad; i++ {
		b.WriteString("\n")
	}

	for _, line := range dialogLines {
		b.WriteString(strings.Repeat(" ", leftPad) + line + "\n")
	}

	return b.String()
}

func DeleteAgentDialog(agentName string) Dialog {
	return NewDialog(
		"Delete Agent",
		"Remove agent '"+agentName+"'?",
		"This cannot be undone.",
	)
}

func DeleteRepoDialog(repoName string) Dialog {
	return NewDialog(
		"Delete Repository",
		"Remove repository '"+repoName+"'?",
		"This cannot be undone.",
	)
}

func ResolveAlertDialog(alertType string) Dialog {
	return NewDialog(
		"Resolve Alert",
		"Mark this "+alertType+" alert as resolved?",
		"",
	)
}
