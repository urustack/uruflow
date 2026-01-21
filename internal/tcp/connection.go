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

package tcp

import (
	"net"
	"sync"
	"time"

	"github.com/urustack/uruflow/internal/tcp/protocol"
)

type Connection struct {
	ID        string
	AgentID   string
	AgentName string
	Conn      net.Conn
	Reader    *protocol.Reader
	Writer    *protocol.Writer
	Connected time.Time
	LastPing  time.Time
	mu        sync.Mutex
	closed    bool
}

func NewConnection(id string, conn net.Conn) *Connection {
	return &Connection{
		ID:        id,
		Conn:      conn,
		Reader:    protocol.NewReader(conn),
		Writer:    protocol.NewWriter(conn),
		Connected: time.Now(),
		LastPing:  time.Now(),
	}
}

func (c *Connection) Send(msg *protocol.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return net.ErrClosed
	}

	return c.Writer.WriteWithTimeout(msg, 10*time.Second)
}

func (c *Connection) Receive() (*protocol.Message, error) {
	return c.Reader.Read()
}

func (c *Connection) ReceiveWithTimeout(timeout time.Duration) (*protocol.Message, error) {
	return c.Reader.ReadWithTimeout(timeout)
}

func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return c.Conn.Close()
}

func (c *Connection) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *Connection) SetAgent(agentID, agentName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AgentID = agentID
	c.AgentName = agentName
}

func (c *Connection) UpdatePing() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastPing = time.Now()
}

func (c *Connection) RemoteAddr() string {
	return c.Conn.RemoteAddr().String()
}
