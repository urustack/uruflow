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

func (c *Collector) getDiskInfo(path string) (uint64, uint64, error) {
	return 0, 0, nil
}

func (c *Collector) getCPUPercent() (float64, error) {
	return 0, nil
}

func (c *Collector) getMemoryInfo() (uint64, uint64, error) {
	return 0, 0, nil
}

func (c *Collector) getLoadAvg() []float64 {
	return []float64{0, 0, 0}
}
