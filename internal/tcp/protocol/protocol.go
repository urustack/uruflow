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
	"encoding/binary"
	"errors"
)

const (
	MagicByte1 byte = 0x55
	MagicByte2 byte = 0x46
	Version    byte = 0x01

	HeaderSize     = 8
	MaxPayloadSize = 16 * 1024 * 1024
)

type MessageType byte

const (
	TypeAuth     MessageType = 0x01
	TypeAuthOK   MessageType = 0x02
	TypeAuthFail MessageType = 0x03

	TypeMetrics    MessageType = 0x10
	TypeMetricsAck MessageType = 0x11

	TypeCommand      MessageType = 0x20
	TypeCommandAck   MessageType = 0x21
	TypeCommandStart MessageType = 0x22
	TypeCommandLog   MessageType = 0x23
	TypeCommandDone  MessageType = 0x24

	TypePing MessageType = 0x30
	TypePong MessageType = 0x31

	TypeDisconnect MessageType = 0x40
	TypeError      MessageType = 0x41
)

var (
	ErrInvalidMagic    = errors.New("invalid magic bytes")
	ErrInvalidVersion  = errors.New("unsupported protocol version")
	ErrPayloadTooLarge = errors.New("payload exceeds maximum size")
	ErrInvalidHeader   = errors.New("invalid header")
)

func (t MessageType) String() string {
	switch t {
	case TypeAuth:
		return "AUTH"
	case TypeAuthOK:
		return "AUTH_OK"
	case TypeAuthFail:
		return "AUTH_FAIL"
	case TypeMetrics:
		return "METRICS"
	case TypeMetricsAck:
		return "METRICS_ACK"
	case TypeCommand:
		return "COMMAND"
	case TypeCommandAck:
		return "COMMAND_ACK"
	case TypeCommandStart:
		return "COMMAND_START"
	case TypeCommandLog:
		return "COMMAND_LOG"
	case TypeCommandDone:
		return "COMMAND_DONE"
	case TypePing:
		return "PING"
	case TypePong:
		return "PONG"
	case TypeDisconnect:
		return "DISCONNECT"
	case TypeError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func EncodeHeader(msgType MessageType, payloadLen uint32) []byte {
	header := make([]byte, HeaderSize)
	header[0] = MagicByte1
	header[1] = MagicByte2
	header[2] = Version
	header[3] = byte(msgType)
	binary.BigEndian.PutUint32(header[4:], payloadLen)
	return header
}

func DecodeHeader(header []byte) (MessageType, uint32, error) {
	if len(header) < HeaderSize {
		return 0, 0, ErrInvalidHeader
	}

	if header[0] != MagicByte1 || header[1] != MagicByte2 {
		return 0, 0, ErrInvalidMagic
	}

	if header[2] != Version {
		return 0, 0, ErrInvalidVersion
	}

	msgType := MessageType(header[3])
	payloadLen := binary.BigEndian.Uint32(header[4:])

	if payloadLen > MaxPayloadSize {
		return 0, 0, ErrPayloadTooLarge
	}

	return msgType, payloadLen, nil
}
