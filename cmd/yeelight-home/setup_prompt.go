package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/yeelight/yeelight-home/internal/i18n"
	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

type setupPrompt struct {
	input  io.Reader
	reader *bufio.Reader
	stdout io.Writer
	rich   bool
	// accessible keeps the rich component API testable without a real TTY.
	accessible bool
}

func newSetupPrompt(input io.Reader, stdout io.Writer, rich bool) *setupPrompt {
	return &setupPrompt{input: input, reader: bufio.NewReader(input), stdout: stdout, rich: rich}
}

func (prompt *setupPrompt) chooseLanguage() (string, error) {
	if prompt.rich {
		return prompt.chooseLanguageRich()
	}
	_, _ = fmt.Fprint(prompt.stdout, i18n.Text(i18n.Chinese, i18n.SetupChooseLanguage))
	value, err := prompt.readLine()
	if err != nil {
		return "", err
	}
	switch strings.ToLower(value) {
	case "1", "zh", "zh-cn", "中文":
		return i18n.Chinese, nil
	case "2", "en", "en-us", "english":
		return i18n.English, nil
	default:
		return "", fmt.Errorf("language must be zh-CN or en-US")
	}
}

func (prompt *setupPrompt) chooseMode(locale string) (string, error) {
	if prompt.rich {
		return prompt.chooseModeRich(locale)
	}
	labels := []string{
		i18n.Text(locale, i18n.SetupModeSkill),
		i18n.Text(locale, i18n.SetupModeMCP),
		i18n.Text(locale, i18n.SetupModeLAN),
	}
	_, _ = fmt.Fprintln(prompt.stdout, i18n.Text(locale, i18n.SetupChooseMode))
	for index, label := range labels {
		_, _ = fmt.Fprintf(prompt.stdout, "  %d. %s\n", index+1, label)
	}
	value, err := prompt.readLine()
	if err != nil {
		return "", err
	}
	switch strings.ToLower(value) {
	case "", "1", "skill":
		return string(setupdomain.ModeSkill), nil
	case "2", "mcp":
		return string(setupdomain.ModeMCP), nil
	case "3", "lan":
		return string(setupdomain.ModeLAN), nil
	default:
		return "", fmt.Errorf("unsupported setup mode %q", value)
	}
}

func (prompt *setupPrompt) chooseMCPClient(locale string, clients []setupdomain.Client) (string, error) {
	if prompt.rich {
		return prompt.chooseMCPClientRich(locale, clients)
	}
	_, _ = fmt.Fprintln(prompt.stdout, i18n.Text(locale, i18n.SetupChooseClient))
	for index, client := range clients {
		_, _ = fmt.Fprintf(prompt.stdout, "  %d. %s\n", index+1, client.Name)
	}
	value, err := prompt.readLine()
	if err != nil {
		return "", err
	}
	if index, parseErr := strconv.Atoi(value); parseErr == nil && index >= 1 && index <= len(clients) {
		return clients[index-1].ID, nil
	}
	for _, client := range clients {
		if strings.EqualFold(value, client.ID) {
			return client.ID, nil
		}
	}
	return "", fmt.Errorf("MCP auto-configuration is not verified for client %q", value)
}

func (prompt *setupPrompt) chooseHome(locale string, homes []setupHomeChoice) (string, error) {
	if len(homes) == 0 {
		return "", fmt.Errorf("no Yeelight Pro home is available for setup")
	}
	if prompt.rich {
		return prompt.chooseHomeRich(locale, homes)
	}
	_, _ = fmt.Fprintln(prompt.stdout, i18n.Text(locale, i18n.SetupChooseHome))
	for index, home := range homes {
		_, _ = fmt.Fprintf(prompt.stdout, "  %d. %s\n", index+1, home.Name)
	}
	value, err := prompt.readLine()
	if err == io.EOF || value == "" {
		return homes[0].ID, nil
	}
	if err != nil {
		return "", err
	}
	if index, parseErr := strconv.Atoi(value); parseErr == nil && index >= 1 && index <= len(homes) {
		return homes[index-1].ID, nil
	}
	for _, home := range homes {
		if strings.EqualFold(value, home.ID) || strings.EqualFold(value, home.Name) {
			return home.ID, nil
		}
	}
	return "", fmt.Errorf("unknown home %q", value)
}

func (prompt *setupPrompt) reuseCurrentAccount(locale string) (bool, error) {
	if prompt.rich {
		return prompt.reuseCurrentAccountRich(locale)
	}
	_, _ = fmt.Fprintln(prompt.stdout, i18n.Text(locale, i18n.SetupChooseAccount))
	_, _ = fmt.Fprintf(prompt.stdout, "  1. %s\n", i18n.Text(locale, i18n.SetupKeepCurrentAccount))
	_, _ = fmt.Fprintf(prompt.stdout, "  2. %s\n", i18n.Text(locale, i18n.SetupSwitchAccount))
	value, err := prompt.readLine()
	if err == io.EOF || value == "" {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	switch strings.ToLower(value) {
	case "1":
		return true, nil
	case "2":
		return false, nil
	default:
		return false, fmt.Errorf("%s", i18n.Text(locale, i18n.SetupInvalidAccountChoice))
	}
}

func (prompt *setupPrompt) confirm(locale string) (bool, error) {
	if prompt.rich {
		return prompt.confirmRich(locale)
	}
	_, _ = fmt.Fprint(prompt.stdout, i18n.Text(locale, i18n.SetupConfirm))
	value, err := prompt.readLine()
	if err != nil {
		return false, err
	}
	value = strings.ToLower(value)
	return value == "" || value == "y" || value == "yes" || value == "是" || value == "确认", nil
}

func (prompt *setupPrompt) readLine() (string, error) {
	line, err := prompt.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	if err == io.EOF && line == "" {
		return "", io.EOF
	}
	return strings.TrimSpace(line), nil
}
