package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

type setupStepReporter struct {
	output io.Writer
	rich   bool
}

func newSetupStepReporter(options setupExecutionOptions) setupStepReporter {
	rich := options.Prompt != nil && options.Prompt.rich
	return setupStepReporter{output: options.Stdout, rich: rich && !options.Quiet}
}

func (reporter setupStepReporter) start(step setupdomain.Step) {
	if reporter.output == nil {
		return
	}
	reporter.line("◇", strings.TrimSpace(step.Title), "active")
}

func (reporter setupStepReporter) finish(step setupdomain.Step, result setupdomain.StepResult, err error) {
	if reporter.output == nil {
		return
	}
	marker := "◆"
	state := "ok"
	if err != nil {
		marker, state = "■", "failed"
	} else {
		switch result.Status {
		case "warning":
			marker, state = "▲", "warning"
		case "skipped":
			marker, state = "─", "skipped"
		}
	}
	label := strings.TrimSpace(step.Title)
	if result.Message != "" && result.Message != "running" && result.Message != "ready" {
		label += "  " + result.Message
	}
	reporter.line(marker, label, state)
}

func (reporter setupStepReporter) line(marker string, label string, state string) {
	if reporter.rich {
		color := lipgloss.AdaptiveColor{Light: "#8A6500", Dark: "#FFD21C"}
		switch state {
		case "ok":
			color = lipgloss.AdaptiveColor{Light: "#087A55", Dark: "#48D597"}
		case "warning":
			color = lipgloss.AdaptiveColor{Light: "#9A5B00", Dark: "#FFB454"}
		case "failed":
			color = lipgloss.AdaptiveColor{Light: "#B42318", Dark: "#FF6B6B"}
		case "skipped":
			color = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#8B93A1"}
		}
		marker = lipgloss.NewStyle().Foreground(color).Bold(true).Render(marker)
	}
	_, _ = fmt.Fprintf(reporter.output, "%s  %s\n", marker, label)
}

func writeSetupPlanRich(plan setupdomain.Plan, prompt *setupPrompt) {
	accent := lipgloss.AdaptiveColor{Light: "#8A6500", Dark: "#FFD21C"}
	muted := lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#8B93A1"}
	title := lipgloss.NewStyle().Bold(true).Render("Yeelight AI")
	mode := lipgloss.NewStyle().Foreground(muted).Render(fmt.Sprintf("%s · %s", plan.Client.Name, plan.Mode))
	_, _ = fmt.Fprintf(prompt.stdout, "\n%s  %s\n%s  %s\n", lipgloss.NewStyle().Foreground(accent).Bold(true).Render("┌"), title, lipgloss.NewStyle().Foreground(accent).Render("│"), mode)
	for _, step := range plan.Steps {
		_, _ = fmt.Fprintf(prompt.stdout, "%s  %s\n", lipgloss.NewStyle().Foreground(accent).Render("◇"), strings.TrimSpace(step.Title))
	}
	_, _ = fmt.Fprintln(prompt.stdout, lipgloss.NewStyle().Foreground(accent).Render("└"))
}
