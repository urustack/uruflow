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

package protocol

import (
	"bufio"
	"net"
	"sync"
	"time"
)

type Writer struct {
	conn   net.Conn
	writer *bufio.Writer
	mu     sync.Mutex
}

func NewWriter(conn net.Conn) *Writer {
	return &Writer{
		conn:   conn,
		writer: bufio.NewWriter(conn),
	}
}

func (w *Writer) Write(msg *Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data := msg.Encode()
	_, err := w.writer.Write(data)
	if err != nil {
		return err
	}

	return w.writer.Flush()
}

func (w *Writer) WriteWithTimeout(msg *Message, timeout time.Duration) error {
	w.conn.SetWriteDeadline(time.Now().Add(timeout))
	defer w.conn.SetWriteDeadline(time.Time{})
	return w.Write(msg)
}
