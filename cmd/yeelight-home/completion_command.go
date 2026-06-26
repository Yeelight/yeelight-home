package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

var completionCommands = map[string][]string{
	"api":        {"smoke"},
	"approve":    {},
	"auth":       {"login", "qr-check", "status", "token"},
	"auth token": {"delete", "set"},
	"completion": {"bash", "fish", "powershell", "zsh"},
	"config":     {"get", "list", "set", "unset"},
	"doctor":     {},
	"home":       {"list", "select"},
	"invoke":     {},
	"profile":    {"delete", "list", "show", "use"},
	"version":    {},
}

func (app *app) runCompletion(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home completion <bash|zsh|fish|powershell>")
		return exitInvalidInput
	}
	switch args[0] {
	case "bash":
		_, _ = fmt.Fprint(stdout, bashCompletionScript())
	case "zsh":
		_, _ = fmt.Fprint(stdout, zshCompletionScript())
	case "fish":
		_, _ = fmt.Fprint(stdout, fishCompletionScript())
	case "powershell":
		_, _ = fmt.Fprint(stdout, powershellCompletionScript())
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported completion shell %q\n", args[0])
		return exitInvalidInput
	}
	return exitOK
}

func rootCommandNames() []string {
	commands := make([]string, 0, len(completionCommands))
	for command := range completionCommands {
		if !strings.Contains(command, " ") {
			commands = append(commands, command)
		}
	}
	sort.Strings(commands)
	return commands
}

func completionSubcommands(command string) []string {
	values := append([]string{}, completionCommands[command]...)
	sort.Strings(values)
	return values
}

func bashCompletionScript() string {
	return fmt.Sprintf(`# bash completion for yeelight-home
_yeelight_home_completion() {
  local cur prev
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  case "$prev" in
    yeelight-home) COMPREPLY=( $(compgen -W "%s" -- "$cur") ); return 0 ;;
    auth) COMPREPLY=( $(compgen -W "%s" -- "$cur") ); return 0 ;;
    token) COMPREPLY=( $(compgen -W "%s" -- "$cur") ); return 0 ;;
    config) COMPREPLY=( $(compgen -W "%s" -- "$cur") ); return 0 ;;
    home) COMPREPLY=( $(compgen -W "%s" -- "$cur") ); return 0 ;;
    profile) COMPREPLY=( $(compgen -W "%s" -- "$cur") ); return 0 ;;
    completion) COMPREPLY=( $(compgen -W "%s" -- "$cur") ); return 0 ;;
  esac
}
complete -F _yeelight_home_completion yeelight-home
`, strings.Join(rootCommandNames(), " "), strings.Join(completionSubcommands("auth"), " "), strings.Join(completionSubcommands("auth token"), " "), strings.Join(completionSubcommands("config"), " "), strings.Join(completionSubcommands("home"), " "), strings.Join(completionSubcommands("profile"), " "), strings.Join(completionSubcommands("completion"), " "))
}

func zshCompletionScript() string {
	return fmt.Sprintf(`#compdef yeelight-home
# zsh completion for yeelight-home
local -a commands
commands=(%s)
_describe 'command' commands
`, strings.Join(shellWords(rootCommandNames()), " "))
}

func fishCompletionScript() string {
	lines := []string{"# fish completion for yeelight-home"}
	for _, command := range rootCommandNames() {
		lines = append(lines, fmt.Sprintf("complete -c yeelight-home -f -n '__fish_use_subcommand' -a %s", command))
	}
	for _, command := range []string{"auth", "config", "home", "profile", "completion"} {
		for _, subcommand := range completionSubcommands(command) {
			lines = append(lines, fmt.Sprintf("complete -c yeelight-home -f -n '__fish_seen_subcommand_from %s' -a %s", command, subcommand))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func powershellCompletionScript() string {
	return fmt.Sprintf(`# PowerShell completion for yeelight-home
Register-ArgumentCompleter -Native -CommandName yeelight-home -ScriptBlock {
  param($wordToComplete, $commandAst, $cursorPosition)
  $commands = @(%s)
  $commands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
  }
}
`, strings.Join(shellWords(rootCommandNames()), ", "))
}

func shellWords(values []string) []string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "'"+strings.ReplaceAll(value, "'", "''")+"'")
	}
	return quoted
}
