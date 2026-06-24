package main

import (
	"fmt"
	"strconv"
	"strings"
)

type cliFlags struct {
	values map[string]string
}

func parseFlags(args []string) (cliFlags, error) {
	flags := cliFlags{values: map[string]string{}}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if !strings.HasPrefix(arg, "--") {
			return cliFlags{}, fmt.Errorf("unexpected argument %q", arg)
		}
		name := strings.TrimPrefix(arg, "--")
		if name == "" {
			return cliFlags{}, fmt.Errorf("empty flag")
		}
		if strings.Contains(name, "=") {
			parts := strings.SplitN(name, "=", 2)
			flags.values[parts[0]] = parts[1]
			continue
		}
		if index+1 < len(args) && !strings.HasPrefix(args[index+1], "--") {
			flags.values[name] = args[index+1]
			index++
			continue
		}
		flags.values[name] = "true"
	}
	return flags, nil
}

func (flags cliFlags) bool(name string) bool {
	value, ok := flags.values[name]
	return ok && (value == "true" || value == "1")
}

func (flags cliFlags) string(name string, fallback string) string {
	if value := strings.TrimSpace(flags.values[name]); value != "" {
		return value
	}
	return fallback
}

func (flags cliFlags) int(name string, fallback int) int {
	value := strings.TrimSpace(flags.values[name])
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
