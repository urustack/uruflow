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

package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	Primary     = lipgloss.Color("#2563EB")
	PrimaryDark = lipgloss.Color("#1D4ED8")
	Muted       = lipgloss.Color("#888888")
	Subtle      = lipgloss.Color("#666666")
	Dim         = lipgloss.Color("#444444")
	DimBorder   = lipgloss.Color("#333333")
	Surface     = lipgloss.Color("#1a1a1a")
	Success     = lipgloss.Color("#22C55E")
	Error       = lipgloss.Color("#EF4444")
	Warning     = lipgloss.Color("#FBBF24")
)

var (
	PrimaryStyle = lipgloss.NewStyle().Foreground(Primary)
	BrightStyle  = lipgloss.NewStyle()
	MutedStyle   = lipgloss.NewStyle().Foreground(Muted)
	SubtleStyle  = lipgloss.NewStyle().Foreground(Subtle)
	DimStyle     = lipgloss.NewStyle().Foreground(Dim)
	SuccessStyle = lipgloss.NewStyle().Foreground(Success)
	ErrorStyle   = lipgloss.NewStyle().Foreground(Error)
	WarningStyle = lipgloss.NewStyle().Foreground(Warning)
	TitleStyle   = lipgloss.NewStyle().Bold(true)
	HeaderStyle  = lipgloss.NewStyle().Foreground(Primary).Bold(true)

	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DimBorder).
		Padding(1, 2)

	BoxSelected = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	BoxFocused = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	RowSelected = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Padding(0, 1)

	BoxSuccess = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Success).
			Padding(1, 2)

	BoxError = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Error).
			Padding(1, 2)

	BoxWarning = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Warning).
			Padding(1, 2)

	InputBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Dim).
			Padding(0, 1)

	InputBoxFocused = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(0, 1)

	KeyStyle  = lipgloss.NewStyle().Foreground(Primary).Bold(true)
	DescStyle = lipgloss.NewStyle().Foreground(Subtle)

	BadgeOnline  = lipgloss.NewStyle().Background(Success).Padding(0, 1)
	BadgeOffline = lipgloss.NewStyle().Background(Dim).Padding(0, 1)
	BadgeSuccess = lipgloss.NewStyle().Background(Success).Padding(0, 1)
	BadgeError   = lipgloss.NewStyle().Background(Error).Padding(0, 1)
	BadgeWarning = lipgloss.NewStyle().Background(Warning).Padding(0, 1)
	BadgePrimary = lipgloss.NewStyle().Background(Primary).Padding(0, 1)
	BadgeMuted   = lipgloss.NewStyle().Background(Subtle).Padding(0, 1)
)

const (
	IconOnline    = "●"
	IconOffline   = "○"
	IconSuccess   = "✓"
	IconError     = "✗"
	IconWarning   = "⚠"
	IconPointer   = "▸"
	IconDash      = "─"
	IconSpin      = "◐"
	IconUncheck   = "◇"
	IconBar       = "│"
	IconBreadSep  = "›"
	IconStepDone  = "●"
	IconStepCurr  = "◉"
	IconStepTodo  = "○"
)

var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func Online() string  { return SuccessStyle.Render(IconOnline) }
func Offline() string { return MutedStyle.Render(IconOffline) }
func Pointer() string { return PrimaryStyle.Render(IconPointer) }

func Line(w int) string {
	if w < 0 {
		w = 0
	}
	return DimStyle.Render(strings.Repeat(IconDash, w))
}

func Logo() string {
	return PrimaryStyle.Bold(true).Render(`██╗   ██╗██████╗ ██╗   ██╗███████╗██╗      ██████╗ ██╗    ██╗
██║   ██║██╔══██╗██║   ██║██╔════╝██║     ██╔═══██╗██║    ██║
██║   ██║██████╔╝██║   ██║█████╗  ██║     ██║   ██║██║ █╗ ██║
██║   ██║██╔══██╗██║   ██║██╔══╝  ██║     ██║   ██║██║███╗██║
╚██████╔╝██║  ██║╚██████╔╝██║     ███████╗╚██████╔╝╚███╔███╔╝
 ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝`)
}

func LogoCompact() string {
	return PrimaryStyle.Bold(true).Render("◆ URUFLOW")
}

func LogoInline() string {
	return PrimaryStyle.Bold(true).Render("URUFLOW")
}

func BreadcrumbSep() string {
	return SubtleStyle.Render(" " + IconBreadSep + " ")
}

func Spinner(frame int) string {
	idx := frame % len(SpinnerFrames)
	return PrimaryStyle.Render(SpinnerFrames[idx])
}

func Tagline() string {
	return SubtleStyle.Render("Automation Deployment System") + "  " + MutedStyle.Render("v1.1.0")
}

func Pad(s string, w int) string {
	l := lipgloss.Width(s)
	if l >= w {
		return s
	}
	return s + strings.Repeat(" ", w-l)
}

func PadL(s string, w int) string {
	l := lipgloss.Width(s)
	if l >= w {
		return s
	}
	return strings.Repeat(" ", w-l) + s
}

func Center(s string, w int) string {
	l := lipgloss.Width(s)
	if l >= w {
		return s
	}
	pad := (w - l) / 2
	return strings.Repeat(" ", pad) + s
}

func Trunc(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 2 {
		return string(r[:w])
	}
	return string(r[:w-2]) + ".."
}
