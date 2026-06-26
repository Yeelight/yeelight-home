package main

import (
	"fmt"
	"io"
	"runtime"
)

var (
	commit = "unknown"
	date   = "unknown"
)

func printVersion(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "version: %v\n", err)
		return exitInvalidInput
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, versionInfo())
	}
	_, _ = fmt.Fprintf(stdout, "yeelight-home %s\n", version)
	return exitOK
}

func versionInfo() map[string]any {
	return map[string]any{
		"cli":     "yeelight-home",
		"version": version,
		"commit":  commit,
		"date":    date,
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
	}
}
