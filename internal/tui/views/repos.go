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

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/urustack/uruflow/internal/config"
	"github.com/urustack/uruflow/internal/models"
	"github.com/urustack/uruflow/internal/services"
	"github.com/urustack/uruflow/internal/storage"
	"github.com/urustack/uruflow/internal/tui/components"
	"github.com/urustack/uruflow/internal/tui/styles"
	"github.com/urustack/uruflow/pkg/helper"
)

type RepoMode int

const (
	RepoModeList RepoMode = iota
	RepoModeAdd
	RepoModeSelectAgent
	RepoModeConfirmDelete
)

const (
	RepoStepName       = 0
	RepoStepURL        = 1
	RepoStepBranch     = 2
	RepoStepBuild      = 3
	RepoStepBuildFile  = 4
	RepoStepPath       = 5
	RepoStepAutoDeploy = 6
	RepoStepTotal      = 7
)

var buildSystems = []string{"compose", "dockerfile", "makefile"}

type RepoResultMsg struct {
	Success bool
	Name    string
	Error   error
}

type ReposModel struct {
	store         storage.Store
	cfg           *config.Config
	cfgPath       string
	deployService *services.DeploymentService
	Width         int
	Height        int
	Repos         []RepoData
	Agents        []AgentData
	Cursor        int
	Expanded      bool
	Mode          RepoMode
	AddStep       int
	NewRepo       NewRepoData
	AgentCursor   int
	BuildCursor   int
	Dialog        components.Dialog
	Loading       bool
	SpinnerFrame  int
	input         textinput.Model

	err error
}

type NewRepoData struct {
	Name        string
	URL         string
	Branch      string
	Path        string
	AgentID     string
	AgentName   string
	AutoDeploy  bool
	BuildSystem string
	BuildFile   string
}

func NewReposModel(store storage.Store, cfg *config.Config, cfgPath string, deployService *services.DeploymentService) ReposModel {
	ti := textinput.New()
	ti.Cursor.Style = styles.PrimaryStyle
	ti.CharLimit = 150
	ti.Focus()

	return ReposModel{
		store: store, cfg: cfg, cfgPath: cfgPath, deployService: deployService,
		Mode:    RepoModeList,
		NewRepo: NewRepoData{Branch: "main", AutoDeploy: true, BuildSystem: "compose"},
		input:   ti,
	}
}

func (m ReposModel) Init() tea.Cmd {
	return tea.Batch(m.fetchRepos, m.fetchAgents, textinput.Blink)
}

func (m ReposModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.Mode {
		case RepoModeList:
			return m.updateList(msg)
		case RepoModeAdd:
			return m.updateAdd(msg)
		case RepoModeSelectAgent:
			return m.updateSelectAgent(msg)
		case RepoModeConfirmDelete:
			return m.updateConfirmDelete(msg)
		}
	case SpinnerTickMsg:
		m.SpinnerFrame++
		if m.Loading {
			return m, m.spinnerTick
		}
	case RepoResultMsg:
		if msg.Success {
			m.Mode = RepoModeList
			m.err = nil
		} else {
			m.err = msg.Error
		}
		m.Loading = false
		return m, m.fetchRepos
	case []RepoData:
		m.Repos = msg
		m.Loading = false
		return m, nil
	case []AgentData:
		m.Agents = msg
		return m, nil
	case error:
		m.err = msg
		m.Loading = false
		return m, nil
	}

	if m.Mode == RepoModeAdd {
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m ReposModel) spinnerTick() tea.Msg {
	time.Sleep(80 * time.Millisecond)
	return SpinnerTickMsg{}
}

func (m ReposModel) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.Mode = RepoModeList
		m.Dialog.Visible = false
	case "left", "right", "h", "l", "tab":
		m.Dialog.ToggleSelection()
	case "enter":
		if m.Dialog.IsConfirmed() {
			m.Dialog.Visible = false
			m.Mode = RepoModeList
			m.Loading = true
			return m, tea.Batch(m.deleteRepo(m.Repos[m.Cursor].Name), m.spinnerTick)
		} else {
			m.Mode = RepoModeList
			m.Dialog.Visible = false
		}
	case "y":
		m.Dialog.Visible = false
		m.Mode = RepoModeList
		m.Loading = true
		return m, tea.Batch(m.deleteRepo(m.Repos[m.Cursor].Name), m.spinnerTick)
	}
	return m, nil
}

func (m ReposModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < len(m.Repos)-1 {
			m.Cursor++
		}
	case "enter":
		if len(m.Repos) > 0 {
			return m, m.triggerDeploy(m.Cursor)
		}
	case "+", "n":
		m.Mode = RepoModeAdd
		m.AddStep = 0
		m.NewRepo = NewRepoData{Branch: "main", AutoDeploy: true, BuildSystem: "compose"}
		m.BuildCursor = 0
		m.err = nil
		m.input.SetValue("")
		m.input.Placeholder = "my-service"
		m.input.Focus()
		return m, textinput.Blink

	case "-", "delete", "backspace":
		if len(m.Repos) > 0 {
			m.Dialog = components.DeleteRepoDialog(m.Repos[m.Cursor].Name)
			m.Mode = RepoModeConfirmDelete
		}
	case "r":
		m.Loading = true
		return m, tea.Batch(m.fetchRepos, m.spinnerTick)
	case "e":
		m.Expanded = !m.Expanded
	}
	return m, nil
}

func (m ReposModel) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	isSelectionStep := (m.AddStep == 3 || m.AddStep == 6)

	switch msg.String() {
	case "esc":
		if m.AddStep > 0 {
			m.AddStep--
			switch m.AddStep {
			case 0:
				m.input.SetValue(m.NewRepo.Name)
			case 1:
				m.input.SetValue(m.NewRepo.URL)
			case 2:
				m.input.SetValue(m.NewRepo.Branch)
			case 4:
				m.input.SetValue(m.NewRepo.BuildFile)
			case 5:
				m.input.SetValue(m.NewRepo.Path)
			}
		} else {
			m.Mode = RepoModeList
		}
		return m, nil

	case "enter":
		val := m.input.Value()
		switch m.AddStep {
		case RepoStepName:
			if val == "" {
				return m, nil
			}
			m.NewRepo.Name = val
		case RepoStepURL:
			if val == "" {
				return m, nil
			}
			m.NewRepo.URL = val
		case RepoStepBranch:
			m.NewRepo.Branch = val
		case RepoStepBuild:
			m.NewRepo.BuildSystem = buildSystems[m.BuildCursor]
		case RepoStepBuildFile:
			m.NewRepo.BuildFile = val
		case RepoStepPath:
			m.NewRepo.Path = val
		}

		if m.AddStep < RepoStepAutoDeploy {
			m.AddStep++
			m.input.SetValue("")
			switch m.AddStep {
			case RepoStepURL:
				m.input.Placeholder = "https://github.com/user/repo.git"
			case RepoStepBranch:
				m.input.SetValue("main")
			case RepoStepBuildFile:
				m.input.Placeholder = "docker-compose.yml"
			case RepoStepPath:
				m.input.Placeholder = "./"
			}
		} else {
			m.Mode = RepoModeSelectAgent
			m.AgentCursor = 0
		}
		return m, nil

	case "left", "h":
		if m.AddStep == RepoStepBuild && m.BuildCursor > 0 {
			m.BuildCursor--
		}
	case "right", "l":
		if m.AddStep == RepoStepBuild && m.BuildCursor < len(buildSystems)-1 {
			m.BuildCursor++
		}
	case "tab", " ":
		if m.AddStep == RepoStepAutoDeploy {
			m.NewRepo.AutoDeploy = !m.NewRepo.AutoDeploy
		}
	}

	var cmd tea.Cmd
	if !isSelectionStep {
		m.input, cmd = m.input.Update(msg)
	}
	return m, cmd
}

func (m ReposModel) updateSelectAgent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Mode = RepoModeAdd
		m.AddStep = 6
	case "up", "k":
		if m.AgentCursor > 0 {
			m.AgentCursor--
		}
	case "down", "j":
		if m.AgentCursor < len(m.Agents)-1 {
			m.AgentCursor++
		}
	case "enter":
		if len(m.Agents) > 0 {
			m.NewRepo.AgentID = m.Agents[m.AgentCursor].ID
			m.NewRepo.AgentName = m.Agents[m.AgentCursor].Name
			return m, m.addRepo()
		}
	}
	return m, nil
}

func (m ReposModel) triggerDeploy(index int) tea.Cmd {
	return func() tea.Msg {
		if index >= len(m.Repos) {
			return nil
		}
		repo := m.Repos[index]
		_, err := m.deployService.TriggerDeploy(repo.AgentID, repo.Name, repo.Branch, "HEAD", "manual")
		if err != nil {
			return err
		}
		return m.fetchRepos()
	}
}

func (m ReposModel) addRepo() tea.Cmd {
	return func() tea.Msg {
		repo := models.Repository{
			Name: m.NewRepo.Name, URL: m.NewRepo.URL, Branch: m.NewRepo.Branch,
			Path: m.NewRepo.Path, AgentID: m.NewRepo.AgentID, AutoDeploy: m.NewRepo.AutoDeploy,
			BuildSystem: models.BuildSystem(m.NewRepo.BuildSystem), BuildFile: m.NewRepo.BuildFile,
		}
		if err := m.cfg.AddRepository(repo); err != nil {
			return RepoResultMsg{Success: false, Error: err}
		}
		if err := m.cfg.Save(m.cfgPath); err != nil {
			return RepoResultMsg{Success: false, Error: err}
		}
		m.store.CreateRepository(&repo)
		return RepoResultMsg{Success: true, Name: repo.Name}
	}
}

func (m ReposModel) deleteRepo(name string) tea.Cmd {
	return func() tea.Msg {
		m.cfg.RemoveRepository(name)
		m.cfg.Save(m.cfgPath)
		m.store.DeleteRepository(name)
		return m.fetchRepos()
	}
}

func (m ReposModel) fetchRepos() tea.Msg {
	repos, err := m.store.GetAllRepositories()
	if err != nil {
		return err
	}
	var data []RepoData
	for _, r := range repos {
		deployments, _ := m.store.GetDeploymentsByRepo(r.Name, 1)
		lastStatus, lastCommit, lastTime := "", "", ""
		if len(deployments) > 0 {
			d := deployments[0]
			lastStatus = string(d.Status)
			if len(d.Commit) > 7 {
				lastCommit = d.Commit[:7]
			} else {
				lastCommit = d.Commit
			}
			lastTime = time.Since(d.StartedAt).Round(time.Second).String() + " ago"
		}
		agentName := r.AgentID
		agent, _ := m.store.GetAgent(r.AgentID)
		if agent != nil {
			agentName = agent.Name
		}
		data = append(data, RepoData{
			Name: r.Name, URL: r.URL, Branch: r.Branch, Agent: agentName, AgentID: r.AgentID,
			AutoDeploy: r.AutoDeploy, BuildSystem: string(r.BuildSystem), BuildFile: r.BuildFile,
			LastStatus: lastStatus, LastCommit: lastCommit, LastTime: lastTime,
		})
	}
	return data
}

func (m ReposModel) fetchAgents() tea.Msg {
	agents, err := m.store.GetAllAgents()
	if err != nil {
		return err
	}
	var data []AgentData
	for _, a := range agents {
		data = append(data, AgentData{ID: a.ID, Name: a.Name, Online: a.Status == "online"})
	}
	return data
}

func (m ReposModel) View() string {
	if m.Width == 0 {
		return ""
	}
	switch m.Mode {
	case RepoModeAdd:
		return m.viewAdd()
	case RepoModeSelectAgent:
		return m.viewSelectAgent()
	case RepoModeConfirmDelete:
		return m.viewList() + components.ConfirmDialog(m.Dialog, m.Width, m.Height)
	default:
		return m.viewList()
	}
}

func (m ReposModel) viewList() string {
	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.ViewHeader(w, "Dashboard", "Repositories") + "\n\n")

	b.WriteString(components.Section("OVERVIEW", w) + "\n\n")
	var statsContent strings.Builder
	statsContent.WriteString(fmt.Sprintf("  %s %s",
		styles.BrightStyle.Render(fmt.Sprintf("%d", len(m.Repos))),
		styles.MutedStyle.Render("repositories configured")))
	b.WriteString(components.Wrap(statsContent.String(), w) + "\n\n")

	if m.err != nil {
		b.WriteString(components.MsgError(m.err.Error(), w) + "\n\n")
	}

	b.WriteString(components.Section("REPOSITORY LIST", w) + "\n\n")

	if m.Loading {
		b.WriteString(components.Loading(m.SpinnerFrame, "Loading repositories...") + "\n\n")
	}

	var listContent strings.Builder
	if len(m.Repos) == 0 && !m.Loading {
		listContent.WriteString("  " + styles.MutedStyle.Render("No repositories configured") + "\n")
		listContent.WriteString("  " + styles.SubtleStyle.Render("Press '+' to add your first repository"))
	} else if len(m.Repos) > 0 {
		listContent.WriteString(components.RepoHeader(w) + "\n")
		listContent.WriteString("  " + styles.Line(w-8) + "\n")
		for i, r := range m.Repos {
			selected := i == m.Cursor
			if selected && m.Expanded {
				card := components.RepoCardData{
					Name: r.Name, URL: r.URL, Branch: r.Branch, Agent: r.Agent,
					AutoDeploy: r.AutoDeploy, BuildSystem: r.BuildSystem, BuildFile: r.BuildFile,
					LastStatus: r.LastStatus, LastCommit: r.LastCommit, LastTime: r.LastTime, Selected: true,
				}
				listContent.WriteString(components.RepoCard(card, w-8) + "\n")
			} else {
				row := components.RepoRow(r.Name, r.Branch, r.Agent, r.AutoDeploy, r.LastStatus, r.LastTime, selected, w)
				if selected {
					listContent.WriteString(components.SelectedRow(row, true) + "\n")
				} else {
					listContent.WriteString(row + "\n")
				}
			}
		}
	}
	if listContent.Len() > 0 {
		b.WriteString(components.Wrap(listContent.String(), w) + "\n")
	}

	content := b.String()
	lines := helper.CountLines(content)
	for i := 0; i < m.Height-lines-3; i++ {
		content += "\n"
	}

	content += "\n" + styles.Line(w) + "\n"
	content += components.Help([][]string{
		{"↑↓", "navigate"}, {"enter", "deploy"}, {"+", "add"}, {"-", "remove"}, {"e", "expand"}, {"esc", "back"},
	})

	return content
}

func (m ReposModel) viewAdd() string {
	var b strings.Builder
	w := m.Width
	stepNames := []string{"Name", "URL", "Branch", "Build System", "Build File", "Path", "Auto Deploy"}
	currentStepName := stepNames[m.AddStep]
	b.WriteString("\n")
	b.WriteString(components.ViewHeader(w, "Dashboard", "Repositories", "Add Repository", currentStepName) + "\n\n")
	stepperSteps := []components.StepperStep{
		{Label: "Repository Name", Value: m.NewRepo.Name},
		{Label: "Git URL", Value: m.NewRepo.URL},
		{Label: "Branch", Value: m.NewRepo.Branch},
		{Label: "Build System", Value: m.NewRepo.BuildSystem},
		{Label: "Build File", Value: m.NewRepo.BuildFile},
		{Label: "Deploy Path", Value: m.NewRepo.Path},
		{Label: "Auto Deploy", Value: fmt.Sprintf("%v", m.NewRepo.AutoDeploy)},
	}

	b.WriteString(components.FormStepper(stepperSteps, m.AddStep, w) + "\n")
	var formContent strings.Builder
	switch m.AddStep {
	case RepoStepBuild:
		formContent.WriteString("\n")
		for j, bs := range buildSystems {
			if j == m.BuildCursor {
				formContent.WriteString("  " + styles.PrimaryStyle.Render("["+bs+"]"))
			} else {
				formContent.WriteString("  " + styles.MutedStyle.Render("["+bs+"]"))
			}
		}
		formContent.WriteString("\n\n  " + styles.MutedStyle.Render("← → to select, enter to confirm"))

	case RepoStepAutoDeploy:
		formContent.WriteString("\n")
		formContent.WriteString(components.Toggle("Auto Deploy", m.NewRepo.AutoDeploy, true) + "\n")
		formContent.WriteString("\n  " + styles.MutedStyle.Render("Press space to toggle, enter to continue"))

	default:
		inputView := styles.InputBoxFocused.Width(w - 8).Render(m.input.View())
		formContent.WriteString("\n  " + inputView)

		switch m.AddStep {
		case RepoStepBuildFile:
			formContent.WriteString("\n  " + styles.MutedStyle.Render("e.g. docker-compose.prod.yml (optional)"))
		case RepoStepPath:
			formContent.WriteString("\n  " + styles.MutedStyle.Render("Relative path to deploy directory (optional)"))
		}
	}

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
	content += components.Help([][]string{{"enter", "next"}, {"esc", "back"}})

	return content
}

func (m ReposModel) viewSelectAgent() string {
	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.ViewHeader(w, "Dashboard", "Repositories", "Add Repository", "Select Agent") + "\n\n")

	b.WriteString(components.Section("CHOOSE DEPLOYMENT TARGET", w) + "\n\n")

	var listContent strings.Builder
	if len(m.Agents) == 0 {
		listContent.WriteString("  " + styles.MutedStyle.Render("No agents available") + "\n")
		listContent.WriteString("  " + styles.SubtleStyle.Render("Add an agent first"))
	} else {
		for i, a := range m.Agents {
			selected := i == m.AgentCursor
			listContent.WriteString(components.AgentRow(a.Name, a.Online, 0, 0, 0, "", selected, w) + "\n")
		}
	}
	b.WriteString(components.Wrap(listContent.String(), w) + "\n")

	content := b.String()
	lines := helper.CountLines(content)
	for i := 0; i < m.Height-lines-3; i++ {
		content += "\n"
	}

	content += "\n" + styles.Line(w) + "\n"
	content += components.Help([][]string{{"↑↓", "select"}, {"enter", "confirm"}, {"esc", "back"}})

	return content
}
