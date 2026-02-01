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
	"fmt"
	"strings"

	"github.com/urustack/uruflow/internal/tui/styles"
)

type StepperStep struct {
	Label string
	Value string
}

func FormStepper(steps []StepperStep, currentStep int, w int) string {
	var b strings.Builder

	totalSteps := len(steps)

	progressWidth := w - 8
	if progressWidth < 20 {
		progressWidth = 20
	}

	filledWidth := 0
	if totalSteps > 1 {
		filledWidth = (progressWidth * currentStep) / (totalSteps - 1)
	}
	if currentStep >= totalSteps-1 {
		filledWidth = progressWidth
	}

	emptyWidth := progressWidth - filledWidth
	if emptyWidth < 0 {
		emptyWidth = 0
	}

	progressFilled := strings.Repeat("━", filledWidth)
	progressEmpty := strings.Repeat("─", emptyWidth)

	b.WriteString("  " + styles.PrimaryStyle.Render(progressFilled) + styles.DimStyle.Render(progressEmpty) + "\n")

	b.WriteString("  " + styles.MutedStyle.Render(fmt.Sprintf("Step %d of %d", currentStep+1, totalSteps)) + "\n\n")

	for i, step := range steps {
		var icon string
		var labelStyle, valueStyle func(string) string

		if i < currentStep {
			icon = styles.SuccessStyle.Render(styles.IconStepDone)
			labelStyle = func(s string) string { return styles.MutedStyle.Render(s) }
			valueStyle = func(s string) string { return styles.SubtleStyle.Render(s) }
		} else if i == currentStep {
			icon = styles.PrimaryStyle.Render(styles.IconStepCurr)
			labelStyle = func(s string) string { return styles.PrimaryStyle.Bold(true).Render(s) }
			valueStyle = func(s string) string { return s }
		} else {
			icon = styles.DimStyle.Render(styles.IconStepTodo)
			labelStyle = func(s string) string { return styles.DimStyle.Render(s) }
			valueStyle = func(s string) string { return styles.DimStyle.Render(s) }
		}

		line := "  " + icon + "  " + labelStyle(step.Label)
		if step.Value != "" && i < currentStep {
			truncValue := step.Value
			if len(truncValue) > 30 {
				truncValue = truncValue[:27] + "..."
			}
			line += "  " + styles.SubtleStyle.Render("→") + "  " + valueStyle(truncValue)
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

func FormStepperCompact(totalSteps, currentStep int) string {
	var parts []string

	for i := 0; i < totalSteps; i++ {
		if i < currentStep {
			parts = append(parts, styles.SuccessStyle.Render(styles.IconStepDone))
		} else if i == currentStep {
			parts = append(parts, styles.PrimaryStyle.Render(styles.IconStepCurr))
		} else {
			parts = append(parts, styles.DimStyle.Render(styles.IconStepTodo))
		}
	}

	return strings.Join(parts, " ")
}
