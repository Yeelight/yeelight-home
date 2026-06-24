package main

import (
	"fmt"
	"io"
	"strings"
)

func (app *app) runApprove(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid approve flags: %v\n", err)
		return exitInvalidInput
	}
	if !flags.bool("json") {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home approve --json --plan-id <id> --challenge <text>")
		return exitInvalidInput
	}
	planID := flags.string("plan-id", "")
	challenge := flags.string("challenge", "")
	if strings.TrimSpace(planID) == "" || strings.TrimSpace(challenge) == "" {
		_, _ = fmt.Fprintln(stderr, "approve requires --plan-id and --challenge")
		return exitInvalidInput
	}
	record, err := app.planStore.MarkApproved(planID, challenge)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "approve plan: %v\n", err)
		return exitInvalidInput
	}
	return writeJSON(stdout, stderr, map[string]any{
		"status":      "approved",
		"planId":      record.ID,
		"risk":        record.Risk,
		"intent":      record.Intent,
		"approvedAt":  record.ApprovedAt,
		"nextIntent":  "plan.commit",
		"commitShape": map[string]any{"parameters": map[string]any{"planId": record.ID}},
	})
}
