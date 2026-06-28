package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

var nativeCompletionCommands = map[string][]string{
	"api":        {"smoke"},
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
	commands := make([]string, 0, len(nativeCompletionCommands)+len(moduleCommands))
	for command := range nativeCompletionCommands {
		if !strings.Contains(command, " ") {
			commands = append(commands, command)
		}
	}
	for _, command := range moduleResourceNames() {
		if _, exists := nativeCompletionCommands[command]; !exists {
			commands = append(commands, command)
		}
	}
	sort.Strings(commands)
	return commands
}

func completionSubcommands(command string) []string {
	values := append([]string{}, nativeCompletionCommands[command]...)
	if subcommands, ok := moduleCommands[command]; ok {
		for subcommand := range subcommands {
			values = append(values, subcommand)
		}
	}
	values = uniqueStrings(values)
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
    token) COMPREPLY=( $(compgen -W "%s" -- "$cur") ); return 0 ;;
%s
  esac
}
complete -F _yeelight_home_completion yeelight-home
`, strings.Join(rootCommandNames(), " "), strings.Join(completionSubcommands("auth token"), " "), bashResourceCompletionCases())
}

func zshCompletionScript() string {
	return fmt.Sprintf(`#compdef yeelight-home
# zsh completion for yeelight-home
local -a commands
commands=(%s)

if (( CURRENT == 2 )); then
  _describe 'command' commands
  return
fi

case "$words[2]" in
%s
esac
`, strings.Join(shellWords(rootCommandNames()), " "), zshResourceCompletionCases())
}

func fishCompletionScript() string {
	lines := []string{"# fish completion for yeelight-home"}
	for _, command := range rootCommandNames() {
		lines = append(lines, fmt.Sprintf("complete -c yeelight-home -f -n '__fish_use_subcommand' -a %s", command))
	}
	for _, command := range rootCommandNames() {
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

func bashResourceCompletionCases() string {
	lines := []string{}
	for _, command := range rootCommandNames() {
		subcommands := completionSubcommands(command)
		if len(subcommands) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("    %s) COMPREPLY=( $(compgen -W \"%s\" -- \"$cur\") ); return 0 ;;", command, strings.Join(subcommands, " ")))
	}
	return strings.Join(lines, "\n")
}

func zshResourceCompletionCases() string {
	lines := []string{}
	for _, command := range rootCommandNames() {
		subcommands := completionSubcommands(command)
		if len(subcommands) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %s) local -a actions; actions=(%s); _describe 'action' actions ;;", command, strings.Join(shellWords(subcommands), " ")))
	}
	return strings.Join(lines, "\n")
}
