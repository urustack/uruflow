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
	err           error
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
	return ReposModel{
		store: store, cfg: cfg, cfgPath: cfgPath, deployService: deployService,
		Mode: RepoModeList, NewRepo: NewRepoData{Branch: "main", AutoDeploy: true, BuildSystem: "compose"},
	}
}

func (m ReposModel) Init() tea.Cmd {
	return tea.Batch(m.fetchRepos, m.fetchAgents)
}

func (m ReposModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.Mode {
		case RepoModeList:
			return m.updateList(msg)
		case RepoModeAdd:
			return m.updateAdd(msg)
		case RepoModeSelectAgent:
			return m.updateSelectAgent(msg)
		}
	case RepoResultMsg:
		if msg.Success {
			m.Mode = RepoModeList
			m.err = nil
		} else {
			m.err = msg.Error
		}
		return m, m.fetchRepos
	case []RepoData:
		m.Repos = msg
		return m, nil
	case []AgentData:
		m.Agents = msg
		return m, nil
	case error:
		m.err = msg
		return m, nil
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
	case "-", "delete", "backspace":
		if len(m.Repos) > 0 {
			return m, m.deleteRepo(m.Repos[m.Cursor].Name)
		}
	case "r":
		return m, m.fetchRepos
	case "e":
		m.Expanded = !m.Expanded
	}
	return m, nil
}

func (m ReposModel) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.AddStep > 0 {
			m.AddStep--
		} else {
			m.Mode = RepoModeList
		}
	case "enter":
		switch m.AddStep {
		case 0:
			if m.NewRepo.Name != "" {
				m.AddStep = 1
			}
		case 1:
			if m.NewRepo.URL != "" {
				m.AddStep = 2
			}
		case 2:
			if m.NewRepo.Branch != "" {
				m.AddStep = 3
			}
		case 3:
			m.NewRepo.BuildSystem = buildSystems[m.BuildCursor]
			m.AddStep = 4
		case 4:
			m.AddStep = 5
		case 5:
			m.AddStep = 6
		case 6:
			m.Mode = RepoModeSelectAgent
			m.AgentCursor = 0
		}
	case "backspace":
		switch m.AddStep {
		case 0:
			if len(m.NewRepo.Name) > 0 {
				m.NewRepo.Name = m.NewRepo.Name[:len(m.NewRepo.Name)-1]
			}
		case 1:
			if len(m.NewRepo.URL) > 0 {
				m.NewRepo.URL = m.NewRepo.URL[:len(m.NewRepo.URL)-1]
			}
		case 2:
			if len(m.NewRepo.Branch) > 0 {
				m.NewRepo.Branch = m.NewRepo.Branch[:len(m.NewRepo.Branch)-1]
			}
		case 4:
			if len(m.NewRepo.BuildFile) > 0 {
				m.NewRepo.BuildFile = m.NewRepo.BuildFile[:len(m.NewRepo.BuildFile)-1]
			}
		case 5:
			if len(m.NewRepo.Path) > 0 {
				m.NewRepo.Path = m.NewRepo.Path[:len(m.NewRepo.Path)-1]
			}
		}
	case "tab", " ":
		if m.AddStep == 6 {
			m.NewRepo.AutoDeploy = !m.NewRepo.AutoDeploy
		}
	case "left":
		if m.AddStep == 3 && m.BuildCursor > 0 {
			m.BuildCursor--
		}
	case "right":
		if m.AddStep == 3 && m.BuildCursor < len(buildSystems)-1 {
			m.BuildCursor++
		}
	default:
		inputStr := msg.String()
		if m.AddStep == 3 && (inputStr == "h" || inputStr == "l") {
			if inputStr == "h" && m.BuildCursor > 0 {
				m.BuildCursor--
			} else if inputStr == "l" && m.BuildCursor < len(buildSystems)-1 {
				m.BuildCursor++
			}
			return m, nil
		}

		if msg.Type == tea.KeyRunes {
			if len(inputStr) > 0 {
				switch m.AddStep {
				case 0:
					if len(m.NewRepo.Name)+len(inputStr) <= 30 {
						m.NewRepo.Name += inputStr
					}
				case 1:
					if len(m.NewRepo.URL)+len(inputStr) <= 150 {
						m.NewRepo.URL += inputStr
					}
				case 2:
					if len(m.NewRepo.Branch)+len(inputStr) <= 30 {
						m.NewRepo.Branch += inputStr
					}
				case 4:
					if len(m.NewRepo.BuildFile)+len(inputStr) <= 50 {
						m.NewRepo.BuildFile += inputStr
					}
				case 5:
					if len(m.NewRepo.Path)+len(inputStr) <= 100 {
						m.NewRepo.Path += inputStr
					}
				}
			}
		}
	}
	return m, nil
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
	default:
		return m.viewList()
	}
}

func (m ReposModel) viewList() string {
	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.Header("REPOSITORIES", w) + "\n\n")

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

	var listContent strings.Builder
	if len(m.Repos) == 0 {
		listContent.WriteString("  " + styles.MutedStyle.Render("No repositories configured") + "\n")
		listContent.WriteString("  " + styles.SubtleStyle.Render("Press '+' to add your first repository"))
	} else {
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
				listContent.WriteString(components.RepoRow(r.Name, r.Branch, r.Agent, r.AutoDeploy, r.LastStatus, r.LastTime, selected, w) + "\n")
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
		{"↑↓", "navigate"}, {"enter", "deploy"}, {"+", "add"}, {"-", "remove"}, {"e", "expand"}, {"esc", "back"},
	})

	return content
}

func (m ReposModel) viewAdd() string {
	var b strings.Builder
	w := m.Width

	b.WriteString("\n")
	b.WriteString(components.Header("ADD REPOSITORY", w) + "\n\n")

	b.WriteString(components.Section(fmt.Sprintf("STEP %d OF 7", m.AddStep+1), w) + "\n\n")

	steps := []struct {
		label string
		value string
	}{
		{"Repository Name", m.NewRepo.Name},
		{"Git URL", m.NewRepo.URL},
		{"Branch", m.NewRepo.Branch},
		{"Build System", m.NewRepo.BuildSystem},
		{"Build File (optional)", m.NewRepo.BuildFile},
		{"Deploy Path (optional)", m.NewRepo.Path},
		{"Auto Deploy", fmt.Sprintf("%v", m.NewRepo.AutoDeploy)},
	}

	var formContent strings.Builder
	for i, s := range steps {
		if i < m.AddStep {
			formContent.WriteString("  " + styles.SuccessStyle.Render(styles.IconSuccess) + "  " +
				styles.SubtleStyle.Render(s.label) + "  " + s.value + "\n")
		} else if i == m.AddStep {
			formContent.WriteString("\n")
			if i == 3 {
				formContent.WriteString("  " + styles.SubtleStyle.Render(s.label) + "\n\n")
				for j, bs := range buildSystems {
					if j == m.BuildCursor {
						formContent.WriteString("  " + styles.PrimaryStyle.Render("["+bs+"]"))
					} else {
						formContent.WriteString("  " + styles.MutedStyle.Render("["+bs+"]"))
					}
				}
				formContent.WriteString("\n\n  " + styles.MutedStyle.Render("← → to select, enter to confirm"))
			} else if i == 6 {
				formContent.WriteString(components.Toggle(s.label, m.NewRepo.AutoDeploy, true) + "\n")
				formContent.WriteString("\n  " + styles.MutedStyle.Render("Press space to toggle"))
			} else if i == 4 {
				formContent.WriteString(components.InputWithHint(s.label, s.value, "e.g. docker-compose.prod.yml", true, w-8))
			} else {
				formContent.WriteString(components.Input(s.label, s.value, true, w-8))
			}
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
	b.WriteString(components.Header("SELECT AGENT", w) + "\n\n")

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
