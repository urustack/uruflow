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

package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urustack/uruflow/internal/api"
	"github.com/urustack/uruflow/internal/config"
	"github.com/urustack/uruflow/internal/storage"
	"github.com/urustack/uruflow/internal/tui/views"
)

type ViewState int
type TickMsg time.Time

const (
	ViewDashboard ViewState = iota
	ViewAgents
	ViewRepos
	ViewAlerts
	ViewDeploy
	ViewLogs
	ViewInit
)

type Model struct {
	ActiveView ViewState
	Width      int
	Height     int
	Ready      bool
	Store      storage.Store
	Config     *config.Config
	CfgPath    string
	Server     *api.Server
	Dashboard  views.DashboardModel
	Agents     views.AgentsModel
	Repos      views.ReposModel
	Alerts     views.AlertsModel
	Deploy     views.DeployModel
	Logs       views.LogsModel
	InitState  views.InitModel
}

func NewModel(store storage.Store, cfg *config.Config, cfgPath string, server *api.Server) Model {
	deployService := server.GetDeployService()
	return Model{
		ActiveView: ViewDashboard,
		Store:      store,
		Config:     cfg,
		CfgPath:    cfgPath,
		Server:     server,
		Dashboard:  views.NewDashboardModel(store),
		Agents:     views.NewAgentsModel(store, cfg, cfgPath),
		Repos:      views.NewReposModel(store, cfg, cfgPath, deployService),
		Alerts:     views.NewAlertsModel(store),
		Deploy:     views.NewDeployModel(store),
		Logs:       views.NewLogsModel(store),
		InitState:  views.NewInitModel(),
	}
}

func NewInitModel() Model {
	return Model{ActiveView: ViewInit, InitState: views.NewInitModel(), Ready: true}
}

func (m *Model) Init() tea.Cmd {
	if m.ActiveView == ViewInit {
		return m.InitState.Init()
	}
	return m.Dashboard.Init()
}

func (m Model) isInputActive() bool {
	if m.ActiveView == ViewAgents && m.Agents.Mode == views.AgentModeAdd {
		return true
	}
	if m.ActiveView == ViewRepos && (m.Repos.Mode == views.RepoModeAdd || m.Repos.Mode == views.RepoModeSelectAgent) {
		return true
	}
	if m.ActiveView == ViewInit {
		return true
	}
	return false
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.ActiveView != ViewDashboard && m.ActiveView != ViewInit {
				if m.ActiveView == ViewAgents && m.Agents.Mode != views.AgentModeList {
					break
				}
				if m.ActiveView == ViewRepos && m.Repos.Mode != views.RepoModeList {
					break
				}
				if m.ActiveView == ViewLogs && m.Logs.Mode == views.LogsModeView {
					break
				}
				m.ActiveView = ViewDashboard
				m.Dashboard.ClearMessage()
				return m, m.Dashboard.Init()
			}
		}

		if !m.isInputActive() {
			switch msg.String() {
			case "q":
				if m.ActiveView != ViewInit && m.ActiveView != ViewDeploy {
					return m, tea.Quit
				}
			case "tab":
				if m.ActiveView != ViewInit {
					switch m.ActiveView {
					case ViewDashboard:
						m.ActiveView = ViewAgents
						cmd = m.Agents.Init()
					case ViewAgents:
						m.ActiveView = ViewRepos
						cmd = m.Repos.Init()
					case ViewRepos:
						m.ActiveView = ViewAlerts
						cmd = m.Alerts.Init()
					case ViewAlerts:
						m.ActiveView = ViewLogs
						cmd = m.Logs.Init()
					case ViewLogs:
						m.ActiveView = ViewDashboard
						m.Dashboard.ClearMessage()
						cmd = m.Dashboard.Init()
					default:
						m.ActiveView = ViewDashboard
						m.Dashboard.ClearMessage()
						cmd = m.Dashboard.Init()
					}
					return m, cmd
				}
			case "a":
				if m.ActiveView == ViewDashboard {
					m.ActiveView = ViewAgents
					return m, m.Agents.Init()
				}
			case "r":
				if m.ActiveView == ViewDashboard {
					m.ActiveView = ViewRepos
					return m, m.Repos.Init()
				}
			case "x":
				if m.ActiveView == ViewDashboard {
					m.ActiveView = ViewAlerts
					return m, m.Alerts.Init()
				}
			case "d":
				if m.ActiveView == ViewDashboard || m.ActiveView == ViewRepos {
					m.ActiveView = ViewDeploy
					return m, m.Deploy.Init()
				}
			case "l":
				if m.ActiveView == ViewDashboard || m.ActiveView == ViewDeploy || m.ActiveView == ViewRepos {
					m.ActiveView = ViewLogs
					return m, m.Logs.Init()
				}
			}
		}

	case views.AgentResultMsg:
		if msg.Success {
			m.Dashboard.SetMessage("Agent '"+msg.Name+"' created successfully", "success")
		}

	case views.RepoResultMsg:
		if msg.Success {
			m.Dashboard.SetMessage("Repository '"+msg.Name+"' added successfully", "success")
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Ready = true
		m.Dashboard.Width = msg.Width
		m.Dashboard.Height = msg.Height
		m.Agents.Width = msg.Width
		m.Agents.Height = msg.Height
		m.Repos.Width = msg.Width
		m.Repos.Height = msg.Height
		m.Alerts.Width = msg.Width
		m.Alerts.Height = msg.Height
		m.Deploy.Width = msg.Width
		m.Deploy.Height = msg.Height
		m.Logs.Width = msg.Width
		m.Logs.Height = msg.Height
		m.InitState.Width = msg.Width
		m.InitState.Height = msg.Height
	}

	switch m.ActiveView {
	case ViewDashboard:
		var newModel tea.Model
		newModel, cmd = m.Dashboard.Update(msg)
		m.Dashboard = newModel.(views.DashboardModel)
		cmds = append(cmds, cmd)
	case ViewAgents:
		var newModel tea.Model
		newModel, cmd = m.Agents.Update(msg)
		m.Agents = newModel.(views.AgentsModel)
		cmds = append(cmds, cmd)
	case ViewRepos:
		var newModel tea.Model
		newModel, cmd = m.Repos.Update(msg)
		m.Repos = newModel.(views.ReposModel)
		cmds = append(cmds, cmd)
	case ViewAlerts:
		var newModel tea.Model
		newModel, cmd = m.Alerts.Update(msg)
		m.Alerts = newModel.(views.AlertsModel)
		cmds = append(cmds, cmd)
	case ViewDeploy:
		var newModel tea.Model
		newModel, cmd = m.Deploy.Update(msg)
		m.Deploy = newModel.(views.DeployModel)
		cmds = append(cmds, cmd)
	case ViewLogs:
		var newModel tea.Model
		newModel, cmd = m.Logs.Update(msg)
		m.Logs = newModel.(views.LogsModel)
		cmds = append(cmds, cmd)
	case ViewInit:
		var newModel tea.Model
		newModel, cmd = m.InitState.Update(msg)
		m.InitState = newModel.(views.InitModel)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if !m.Ready {
		return ""
	}
	switch m.ActiveView {
	case ViewDashboard:
		return m.Dashboard.View()
	case ViewAgents:
		return m.Agents.View()
	case ViewRepos:
		return m.Repos.View()
	case ViewAlerts:
		return m.Alerts.View()
	case ViewDeploy:
		return m.Deploy.View()
	case ViewLogs:
		return m.Logs.View()
	case ViewInit:
		return m.InitState.View()
	default:
		return m.Dashboard.View()
	}
}
