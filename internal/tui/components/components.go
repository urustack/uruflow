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
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/urustack/uruflow/internal/tui/styles"
)

func Wrap(content string, w int) string {
	return styles.Box.Width(w - 4).Render(content)
}

func WrapSelected(content string, w int) string {
	return styles.BoxSelected.Width(w - 4).Render(content)
}

func WrapSuccess(content string, w int) string {
	return styles.BoxSuccess.Width(w - 4).Render(content)
}

func WrapError(content string, w int) string {
	return styles.BoxError.Width(w - 4).Render(content)
}

func WrapWarning(content string, w int) string {
	return styles.BoxWarning.Width(w - 4).Render(content)
}

func Header(title string, w int) string {
	left := styles.LogoCompact()
	right := styles.MutedStyle.Render("v2.0.0")
	if title != "" {
		right = styles.HeaderStyle.Render(title)
	}
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	gap := w - lw - rw - 4
	if gap < 1 {
		gap = 1
	}
	return "  " + left + strings.Repeat(" ", gap) + right + "  "
}

func Section(title string, w int) string {
	t := styles.MutedStyle.Bold(true).Render(strings.ToUpper(title))
	tw := lipgloss.Width(t)
	lw := w - tw - 6
	if lw < 0 {
		lw = 0
	}
	return "  " + t + " " + styles.Line(lw)
}

func Help(items [][]string) string {
	var p []string
	for _, i := range items {
		if len(i) >= 2 {
			p = append(p, styles.KeyStyle.Render(i[0])+" "+styles.DescStyle.Render(i[1]))
		}
	}
	return "  " + strings.Join(p, "   ")
}

func Badge(s string) string {
	switch s {
	case "online":
		return styles.BadgeOnline.Render("ONLINE")
	case "offline":
		return styles.BadgeOffline.Render("OFFLINE")
	case "success":
		return styles.BadgeSuccess.Render("SUCCESS")
	case "failed", "error":
		return styles.BadgeError.Render("FAILED")
	case "running":
		return styles.BadgePrimary.Render("RUNNING")
	case "pending":
		return styles.BadgeWarning.Render("PENDING")
	case "auto":
		return styles.BadgeSuccess.Render("AUTO")
	case "manual":
		return styles.BadgeMuted.Render("MANUAL")
	case "healthy":
		return styles.BadgeSuccess.Render("HEALTHY")
	case "unhealthy":
		return styles.BadgeError.Render("UNHEALTHY")
	case "critical":
		return styles.BadgeError.Render("CRITICAL")
	case "warning":
		return styles.BadgeWarning.Render("WARNING")
	case "compose":
		return styles.BadgePrimary.Render("COMPOSE")
	case "dockerfile":
		return styles.BadgePrimary.Render("DOCKER")
	case "makefile":
		return styles.BadgePrimary.Render("MAKE")
	default:
		return styles.BadgeMuted.Render(strings.ToUpper(s))
	}
}

func CenteredLogo(w int) string {
	logo := styles.Logo()
	lines := strings.Split(logo, "\n")
	lw := 0
	for _, l := range lines {
		if lipgloss.Width(l) > lw {
			lw = lipgloss.Width(l)
		}
	}
	var b strings.Builder
	for _, l := range lines {
		pad := (w - lw) / 2
		if pad < 0 {
			pad = 0
		}
		b.WriteString(strings.Repeat(" ", pad) + l + "\n")
	}
	tag := styles.Tagline()
	tagW := lipgloss.Width(tag)
	tagPad := (w - tagW) / 2
	if tagPad < 0 {
		tagPad = 0
	}
	b.WriteString(strings.Repeat(" ", tagPad) + tag)
	return b.String()
}

func AgentRow(name string, online bool, cpu, mem, disk float64, lastSeen string, selected bool, w int) string {
	ptr := "   "
	if selected {
		ptr = " " + styles.Pointer() + " "
	}

	dot := styles.Offline()
	if online {
		dot = styles.Online()
	}

	nameStyle := styles.BrightStyle
	if selected {
		nameStyle = styles.PrimaryStyle
	}

	status := Badge("offline")
	if online {
		status = Badge("online")
	}

	var metrics string
	if online {
		metrics = fmt.Sprintf("%5.1f%%    %5.1f%%    %5.1f%%", cpu, mem, disk)
	} else {
		if lastSeen == "0001-01-01 00:00" || lastSeen == "0001-01-01" {
			lastSeen = "never"
		}
		metrics = styles.MutedStyle.Render(styles.Pad(lastSeen, 24))
	}

	return fmt.Sprintf("%s%s  %s  %s  %s",
		ptr,
		dot,
		nameStyle.Render(styles.Pad(styles.Trunc(name, 22), 22)),
		styles.Pad(status, 10),
		metrics)
}

func AgentHeader(w int) string {
	return fmt.Sprintf("       %s  %s  %s    %s    %s",
		styles.MutedStyle.Render(styles.Pad("NAME", 22)),
		styles.MutedStyle.Render(styles.Pad("STATUS", 10)),
		styles.MutedStyle.Render(styles.Pad("CPU", 5)),
		styles.MutedStyle.Render(styles.Pad("MEM", 5)),
		styles.MutedStyle.Render(styles.Pad("DISK", 5)))
}

func RepoRow(name, branch, agent string, auto bool, status, lastTime string, selected bool, w int) string {
	ptr := "   "
	if selected {
		ptr = " " + styles.Pointer() + " "
	}

	nameStyle := styles.BrightStyle
	if selected {
		nameStyle = styles.PrimaryStyle
	}

	mode := Badge("manual")
	if auto {
		mode = Badge("auto")
	}

	st := Badge("pending")
	if status == "success" {
		st = Badge("success")
	} else if status == "failed" {
		st = Badge("failed")
	} else if status == "running" {
		st = Badge("running")
	}

	return fmt.Sprintf("%s%s  %s  %s  %s  %s  %s",
		ptr,
		nameStyle.Render(styles.Pad(styles.Trunc(name, 16), 16)),
		styles.MutedStyle.Render(styles.Pad(styles.Trunc(branch, 10), 10)),
		styles.Pad(styles.Trunc(agent, 14), 14),
		styles.Pad(mode, 8),
		styles.Pad(st, 10),
		styles.MutedStyle.Render(styles.Pad(lastTime, 14)))
}

func RepoHeader(w int) string {
	return fmt.Sprintf("    %s  %s  %s  %s  %s  %s",
		styles.MutedStyle.Render(styles.Pad("NAME", 16)),
		styles.MutedStyle.Render(styles.Pad("BRANCH", 10)),
		styles.MutedStyle.Render(styles.Pad("AGENT", 14)),
		styles.MutedStyle.Render(styles.Pad("MODE", 8)),
		styles.MutedStyle.Render(styles.Pad("STATUS", 10)),
		styles.MutedStyle.Render(styles.Pad("LAST DEPLOY", 14)))
}

func DeployRow(icon, repo, branch, commit, agent, time string, w int) string {
	return fmt.Sprintf("  %s  %s  %s  %s  %s  %s",
		icon,
		styles.Pad(styles.Trunc(repo, 14), 14),
		styles.MutedStyle.Render(styles.Pad(styles.Trunc(branch, 10), 10)),
		styles.MutedStyle.Render(styles.Pad(commit, 8)),
		styles.Pad(styles.Trunc(agent, 12), 12),
		styles.MutedStyle.Render(time))
}

func AlertRow(icon, typ, agent, msg, time string, selected bool, w int) string {
	ptr := "   "
	if selected {
		ptr = " " + styles.Pointer() + " "
	}
	return fmt.Sprintf("%s%s  %s  %s  %s  %s",
		ptr,
		icon,
		styles.Pad(styles.Trunc(typ, 12), 12),
		styles.Pad(styles.Trunc(agent, 14), 14),
		styles.MutedStyle.Render(styles.Pad(styles.Trunc(msg, 24), 24)),
		styles.MutedStyle.Render(time))
}

type ProgressStep struct {
	Label    string
	Status   string
	Duration string
}

func Progress(steps []ProgressStep, w int) string {
	var b strings.Builder
	for i, s := range steps {
		var icon, label string
		switch s.Status {
		case "done":
			icon = styles.SuccessStyle.Render(styles.IconSuccess)
			label = s.Label
		case "running":
			icon = styles.PrimaryStyle.Render(styles.IconSpin)
			label = styles.PrimaryStyle.Render(s.Label)
		case "failed":
			icon = styles.ErrorStyle.Render(styles.IconError)
			label = styles.ErrorStyle.Render(s.Label)
		default:
			icon = styles.MutedStyle.Render(styles.IconUncheck)
			label = styles.MutedStyle.Render(s.Label)
		}
		line := fmt.Sprintf("  %s  %s", icon, styles.Pad(label, 28))
		if s.Duration != "" {
			line += styles.MutedStyle.Render(s.Duration)
		}
		b.WriteString(line + "\n")
		if i < len(steps)-1 {
			b.WriteString("  " + styles.DimStyle.Render(styles.IconBar) + "\n")
		}
	}
	return b.String()
}

func LogLine(time, content, stream string, w int) string {
	c := content
	if stream == "stderr" {
		c = styles.ErrorStyle.Render(content)
	}
	return fmt.Sprintf("  %s  %s", styles.SubtleStyle.Render(time), c)
}

func Select(options []string, cursor int) string {
	var b strings.Builder
	for i, o := range options {
		ptr := "   "
		label := styles.MutedStyle.Render(o)
		if i == cursor {
			ptr = " " + styles.Pointer() + " "
			label = styles.PrimaryStyle.Render(o)
		}
		b.WriteString(ptr + label + "\n")
	}
	return b.String()
}

func Input(label, value string, focused bool, w int) string {
	iw := w - 8
	if iw < 20 {
		iw = 20
	}
	ist := styles.InputBox.Width(iw)
	if focused {
		ist = styles.InputBoxFocused.Width(iw)
	}
	disp := value
	if focused {
		disp = value + styles.PrimaryStyle.Render("▎")
	}
	return fmt.Sprintf("  %s\n  %s", styles.SubtleStyle.Render(label), ist.Render(disp))
}

func InputWithHint(label, value, hint string, focused bool, w int) string {
	iw := w - 8
	if iw < 20 {
		iw = 20
	}
	ist := styles.InputBox.Width(iw)
	if focused {
		ist = styles.InputBoxFocused.Width(iw)
	}
	disp := value
	if focused {
		disp = value + styles.PrimaryStyle.Render("▎")
	}
	result := fmt.Sprintf("  %s\n  %s", styles.SubtleStyle.Render(label), ist.Render(disp))
	if hint != "" {
		result += "\n  " + styles.MutedStyle.Render(hint)
	}
	return result
}

func Toggle(label string, value bool, focused bool) string {
	var tog string
	if value {
		tog = styles.SuccessStyle.Render("[ON]") + "  " + styles.MutedStyle.Render("[OFF]")
	} else {
		tog = styles.MutedStyle.Render("[ON]") + "  " + styles.PrimaryStyle.Render("[OFF]")
	}
	ptr := "   "
	ls := styles.MutedStyle
	if focused {
		ptr = " " + styles.Pointer() + " "
		ls = styles.BrightStyle
	}
	return ptr + ls.Render(styles.Pad(label, 20)) + "  " + tog
}

func MsgSuccess(msg string, w int) string {
	return WrapSuccess(styles.SuccessStyle.Render(styles.IconSuccess)+"  "+styles.SuccessStyle.Render(msg), w)
}

func MsgError(msg string, w int) string {
	return WrapError(styles.ErrorStyle.Render(styles.IconError)+"  "+styles.ErrorStyle.Render(msg), w)
}

func MsgWarning(msg string, w int) string {
	return WrapWarning(styles.WarningStyle.Render(styles.IconWarning)+"  "+styles.WarningStyle.Render(msg), w)
}

func MsgInfo(msg string, w int) string {
	return Wrap(styles.PrimaryStyle.Render("●")+"  "+msg, w)
}

func Empty(title, sub string, w int) string {
	c := styles.MutedStyle.Render(title)
	if sub != "" {
		c += "\n" + styles.SubtleStyle.Render(sub)
	}
	return Wrap(c, w)
}

func Token(tok string, w int) string {
	return WrapWarning(styles.WarningStyle.Render(tok), w)
}

func Stats(online, offline, total int) string {
	return fmt.Sprintf("  %s %s    %s %s    %s %s",
		styles.SuccessStyle.Render(fmt.Sprintf("%d", online)),
		styles.MutedStyle.Render("online"),
		styles.MutedStyle.Render(fmt.Sprintf("%d", offline)),
		styles.MutedStyle.Render("offline"),
		styles.BrightStyle.Render(fmt.Sprintf("%d", total)),
		styles.MutedStyle.Render("total"))
}

type CardLine struct {
	Label string
	Value string
}

func Card(title string, lines []CardLine, selected bool, w int) string {
	var b strings.Builder
	b.WriteString(styles.TitleStyle.Render(title) + "\n")
	for _, l := range lines {
		b.WriteString("\n" + styles.SubtleStyle.Render(styles.Pad(l.Label, 10)) + " " + l.Value)
	}
	if selected {
		return WrapSelected(b.String(), w)
	}
	return Wrap(b.String(), w)
}

type AgentCardData struct {
	Name       string
	Host       string
	Version    string
	Online     bool
	CPU        float64
	Memory     float64
	Disk       float64
	Containers []ContainerInfo
	Selected   bool
}

type ContainerInfo struct {
	Name    string
	Running bool
	Healthy bool
	CPU     float64
	Memory  string
}

func AgentCard(d AgentCardData, w int) string {
	var b strings.Builder
	st := "offline"
	if d.Online {
		st = "online"
	}
	b.WriteString(styles.TitleStyle.Render(d.Name) + "  " + Badge(st) + "\n")
	if d.Online {
		b.WriteString("\n" + styles.SubtleStyle.Render("Host    ") + d.Host)
		if d.Version != "" {
			b.WriteString("\n" + styles.SubtleStyle.Render("Version ") + d.Version)
		}
		b.WriteString(fmt.Sprintf("\n\n%s %5.1f%%    %s %5.1f%%    %s %5.1f%%",
			styles.SubtleStyle.Render("CPU"), d.CPU,
			styles.SubtleStyle.Render("MEM"), d.Memory,
			styles.SubtleStyle.Render("DISK"), d.Disk))
		if len(d.Containers) > 0 {
			b.WriteString("\n\n" + styles.SubtleStyle.Render("Containers:"))
			for _, c := range d.Containers {
				dot := styles.Offline()
				if c.Running {
					dot = styles.Online()
				}
				h := "unhealthy"
				if c.Healthy {
					h = "healthy"
				}
				b.WriteString(fmt.Sprintf("\n  %s %s  %s  %5.1f%%  %s",
					dot, styles.Pad(styles.Trunc(c.Name, 14), 14), Badge(h), c.CPU, c.Memory))
			}
		}
	} else {
		b.WriteString("\n" + styles.MutedStyle.Render("Agent is currently offline"))
	}
	if d.Selected {
		return WrapSelected(b.String(), w)
	}
	return Wrap(b.String(), w)
}

type RepoCardData struct {
	Name        string
	URL         string
	Branch      string
	Agent       string
	AutoDeploy  bool
	BuildSystem string
	BuildFile   string
	LastStatus  string
	LastCommit  string
	LastTime    string
	Selected    bool
}

func RepoCard(d RepoCardData, w int) string {
	var b strings.Builder
	mode := "manual"
	if d.AutoDeploy {
		mode = "auto"
	}
	b.WriteString(styles.TitleStyle.Render(d.Name) + "  " + Badge(mode) + "\n")
	b.WriteString("\n" + styles.SubtleStyle.Render(d.URL))
	b.WriteString("\n\n" + styles.SubtleStyle.Render("Branch ") + d.Branch)
	b.WriteString("\n" + styles.SubtleStyle.Render("Agent  ") + d.Agent)

	buildInfo := d.BuildSystem
	if d.BuildFile != "" {
		buildInfo += " → " + d.BuildFile
	}
	b.WriteString("\n" + styles.SubtleStyle.Render("Build  ") + buildInfo)

	if d.LastCommit != "" {
		st := "success"
		if d.LastStatus == "failed" {
			st = "failed"
		} else if d.LastStatus == "running" {
			st = "running"
		}
		b.WriteString("\n\n" + Badge(st) + "  " + d.LastCommit + "  " + styles.MutedStyle.Render(d.LastTime))
	} else {
		b.WriteString("\n\n" + styles.MutedStyle.Render("No deployments yet"))
	}
	if d.Selected {
		return WrapSelected(b.String(), w)
	}
	return Wrap(b.String(), w)
}

type AlertCardData struct {
	Type     string
	Agent    string
	Message  string
	Time     string
	Severity string
	Selected bool
}

func AlertCard(d AlertCardData, w int) string {
	var b strings.Builder
	icon := styles.WarningStyle.Render(styles.IconWarning)
	if d.Severity == "critical" {
		icon = styles.ErrorStyle.Render(styles.IconError)
	}
	b.WriteString(icon + "  " + styles.TitleStyle.Render(strings.ToUpper(d.Type)) + "\n")
	b.WriteString("\n" + styles.SubtleStyle.Render("Agent   ") + d.Agent)
	b.WriteString("\n" + styles.SubtleStyle.Render("Message ") + d.Message)
	b.WriteString("\n\n" + styles.MutedStyle.Render(d.Time))
	if d.Severity == "critical" {
		return WrapError(b.String(), w)
	}
	return WrapWarning(b.String(), w)
}
