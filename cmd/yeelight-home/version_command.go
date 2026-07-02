package main

import (
	"fmt"
	"io"
	"runtime"

	"github.com/yeelight/yeelight-home/internal/semantic"
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
		semantic.FieldCLI:     "yeelight-home",
		semantic.FieldVersion: version,
		semantic.FieldCommit:  commit,
		semantic.FieldDate:    date,
		semantic.FieldOS:      runtime.GOOS,
		semantic.FieldArch:    runtime.GOARCH,
	}
}
