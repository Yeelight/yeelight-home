package main

import (
	"bytes"
	"strings"
	"testing"

	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

func newAccessibleRichPrompt(input string, output *bytes.Buffer) *setupPrompt {
	prompt := newSetupPrompt(strings.NewReader(input), output, true)
	prompt.accessible = true
	return prompt
}

func TestRichPromptUsesConsistentSelections(t *testing.T) {
	t.Run("mode", func(t *testing.T) {
		var output bytes.Buffer
		got, err := newAccessibleRichPrompt("2\n", &output).chooseModeRich("zh-CN")
		if err != nil || got != string(setupdomain.ModeMCP) || !strings.Contains(output.String(), "选择适合你的接入方式") {
			t.Fatalf("got=%q err=%v output=%q", got, err, output.String())
		}
	})

	t.Run("account", func(t *testing.T) {
		var output bytes.Buffer
		got, err := newAccessibleRichPrompt("2\n", &output).reuseCurrentAccountRich("zh-CN")
		if err != nil || got || !strings.Contains(output.String(), "重新扫码并切换账号") {
			t.Fatalf("got=%t err=%v output=%q", got, err, output.String())
		}
	})

	t.Run("home", func(t *testing.T) {
		var output bytes.Buffer
		homes := []setupHomeChoice{{ID: "1", Name: "我的家"}, {ID: "2", Name: "父母家"}}
		got, err := newAccessibleRichPrompt("2\n", &output).chooseHomeRich("zh-CN", homes)
		if err != nil || got != "2" || !strings.Contains(output.String(), "选择默认家庭") {
			t.Fatalf("got=%q err=%v output=%q", got, err, output.String())
		}
	})

	t.Run("confirm", func(t *testing.T) {
		var output bytes.Buffer
		got, err := newAccessibleRichPrompt("n\n", &output).confirmRich("en-US")
		if err != nil || got || !strings.Contains(output.String(), "Proceed with this setup plan?") {
			t.Fatalf("got=%t err=%v output=%q", got, err, output.String())
		}
	})
}

func TestSetupStepReporterDistinguishesEveryState(t *testing.T) {
	var output bytes.Buffer
	reporter := setupStepReporter{output: &output}
	step := setupdomain.Step{Title: "Install Skill"}
	reporter.start(step)
	reporter.finish(step, setupdomain.StepResult{Status: "ok"}, nil)
	reporter.finish(step, setupdomain.StepResult{Status: "skipped", Message: "already authenticated"}, nil)
	reporter.finish(step, setupdomain.StepResult{Status: "warning", Message: "partial"}, nil)
	reporter.finish(step, setupdomain.StepResult{Status: "failed"}, errSetupCancelled)
	for _, marker := range []string{"◇", "◆", "─", "▲", "■"} {
		if !strings.Contains(output.String(), marker) {
			t.Fatalf("missing marker %q in %q", marker, output.String())
		}
	}
}
