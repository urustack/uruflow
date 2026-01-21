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

package metrics

import (
	"time"
)

type System struct {
	CPUPercent    float64
	MemoryPercent float64
	MemoryUsed    uint64
	MemoryTotal   uint64
	DiskPercent   float64
	DiskUsed      uint64
	DiskTotal     uint64
	LoadAvg       []float64
	Uptime        int64
}

type Collector struct {
	prevCPUIdle  uint64
	prevCPUTotal uint64
	bootTime     time.Time
}

func NewCollector() *Collector {
	return &Collector{
		bootTime: time.Now(),
	}
}

func (c *Collector) Collect() (*System, error) {
	m := &System{
		LoadAvg: []float64{0, 0, 0},
		Uptime:  int64(time.Since(c.bootTime).Seconds()),
	}

	if cpu, err := c.getCPUPercent(); err == nil {
		m.CPUPercent = cpu
	}

	memUsed, memTotal, _ := c.getMemoryInfo()
	m.MemoryUsed = memUsed
	m.MemoryTotal = memTotal
	if memTotal > 0 {
		m.MemoryPercent = float64(memUsed) / float64(memTotal) * 100
	}

	diskUsed, diskTotal, _ := c.getDiskInfo("/")
	m.DiskUsed = diskUsed
	m.DiskTotal = diskTotal
	if diskTotal > 0 {
		m.DiskPercent = float64(diskUsed) / float64(diskTotal) * 100
	}

	m.LoadAvg = c.getLoadAvg()

	return m, nil
}
