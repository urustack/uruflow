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

package components

import (
	"strings"

	"github.com/urustack/uruflow/internal/tui/styles"
)

func Breadcrumb(items ...string) string {
	if len(items) == 0 {
		return ""
	}

	var parts []string
	for i, item := range items {
		if i == len(items)-1 {
			parts = append(parts, styles.PrimaryStyle.Render(item))
		} else {
			parts = append(parts, styles.MutedStyle.Render(item))
		}
	}

	return strings.Join(parts, styles.BreadcrumbSep())
}

func ViewHeader(w int, crumbs ...string) string {
	logo := styles.LogoInline()
	breadcrumb := Breadcrumb(crumbs...)

	if breadcrumb != "" {
		return "  " + logo + "   " + breadcrumb
	}
	return "  " + logo
}
