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

package daemon

import (
	"crypto/tls"
	"net"
	"time"
)

func (d *Daemon) connectTLS(addr string) (net.Conn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: d.cfg.Server.TLSSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	return tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
}
