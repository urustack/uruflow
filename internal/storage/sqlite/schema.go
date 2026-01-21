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

package sqlite

const schema = `
CREATE TABLE IF NOT EXISTS agents (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	token TEXT NOT NULL,
	host TEXT DEFAULT '',
	hostname TEXT DEFAULT '',
	version TEXT DEFAULT '',
	status TEXT DEFAULT 'offline',
	cpu_percent REAL DEFAULT 0,
	memory_percent REAL DEFAULT 0,
	disk_percent REAL DEFAULT 0,
	memory_used INTEGER DEFAULT 0,
	memory_total INTEGER DEFAULT 0,
	disk_used INTEGER DEFAULT 0,
	disk_total INTEGER DEFAULT 0,
	uptime INTEGER DEFAULT 0,
	last_heartbeat DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS containers (
	id TEXT PRIMARY KEY,
	agent_id TEXT NOT NULL,
	name TEXT NOT NULL,
	image TEXT DEFAULT '',
	status TEXT DEFAULT 'unknown',
	health TEXT DEFAULT 'unknown',
	cpu_percent REAL DEFAULT 0,
	memory_usage INTEGER DEFAULT 0,
	memory_limit INTEGER DEFAULT 0,
	network_rx INTEGER DEFAULT 0,
	network_tx INTEGER DEFAULT 0,
	restart_count INTEGER DEFAULT 0,
	started_at DATETIME,
	FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS repositories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	url TEXT NOT NULL,
	branch TEXT DEFAULT 'main',
	agent_id TEXT NOT NULL,
	path TEXT DEFAULT '',
	auto_deploy INTEGER DEFAULT 1,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (agent_id) REFERENCES agents(id)
);

CREATE TABLE IF NOT EXISTS deployments (
	id TEXT PRIMARY KEY,
	repo_name TEXT NOT NULL,
	branch TEXT NOT NULL,
	commit_hash TEXT DEFAULT '',
	agent_id TEXT NOT NULL,
	agent_name TEXT DEFAULT '',
	status TEXT DEFAULT 'pending',
	trigger_type TEXT DEFAULT 'manual',
	output TEXT DEFAULT '',
	started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	finished_at DATETIME,
	duration_ms INTEGER DEFAULT 0,
	FOREIGN KEY (agent_id) REFERENCES agents(id)
);

CREATE TABLE IF NOT EXISTS deployment_logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	deployment_id TEXT NOT NULL,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
	stream TEXT DEFAULT 'stdout',
	content TEXT NOT NULL,
	FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS alerts (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	severity TEXT DEFAULT 'warning',
	agent_id TEXT NOT NULL,
	agent_name TEXT NOT NULL,
	message TEXT NOT NULL,
	resolved INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	resolved_at DATETIME,
	FOREIGN KEY (agent_id) REFERENCES agents(id)
);

CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_agents_token ON agents(token);
CREATE INDEX IF NOT EXISTS idx_containers_agent ON containers(agent_id);
CREATE INDEX IF NOT EXISTS idx_deployments_repo ON deployments(repo_name);
CREATE INDEX IF NOT EXISTS idx_deployments_agent ON deployments(agent_id);
CREATE INDEX IF NOT EXISTS idx_deployments_started ON deployments(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_resolved ON alerts(resolved);
CREATE INDEX IF NOT EXISTS idx_alerts_agent ON alerts(agent_id);
CREATE INDEX IF NOT EXISTS idx_deployment_logs_deployment ON deployment_logs(deployment_id);
`
