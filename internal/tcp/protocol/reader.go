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
	"io"
	"net"
	"time"
)

type Reader struct {
	conn   net.Conn
	reader *bufio.Reader
}

func NewReader(conn net.Conn) *Reader {
	return &Reader{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

func (r *Reader) Read() (*Message, error) {
	header := make([]byte, HeaderSize)
	_, err := io.ReadFull(r.reader, header)
	if err != nil {
		return nil, err
	}

	msgType, payloadLen, err := DecodeHeader(header)
	if err != nil {
		return nil, err
	}

	var payload []byte
	if payloadLen > 0 {
		payload = make([]byte, payloadLen)
		_, err = io.ReadFull(r.reader, payload)
		if err != nil {
			return nil, err
		}
	}

	return &Message{
		Type:    msgType,
		Payload: payload,
	}, nil
}

func (r *Reader) ReadWithTimeout(timeout time.Duration) (*Message, error) {
	r.conn.SetReadDeadline(time.Now().Add(timeout))
	defer r.conn.SetReadDeadline(time.Time{})
	return r.Read()
}
