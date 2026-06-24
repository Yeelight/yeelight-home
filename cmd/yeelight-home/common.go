package main

import (
	"encoding/json"
	"fmt"
	"io"
)

func writeJSON(stdout io.Writer, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	if err := encoder.Encode(value); err != nil {
		_, _ = fmt.Fprintf(stderr, "write json: %v\n", err)
		return exitInternalError
	}
	return exitOK
}
