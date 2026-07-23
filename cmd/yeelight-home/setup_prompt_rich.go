package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/yeelight/yeelight-home/internal/i18n"
	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

var errSetupCancelled = errors.New("setup cancelled")

func (prompt *setupPrompt) chooseLanguageRich() (string, error) {
	selected := i18n.Chinese
	field := huh.NewSelect[string]().
		Title(i18n.Text(i18n.Chinese, i18n.SetupChooseLanguageRich)).
		Options(
			huh.NewOption("中文", i18n.Chinese),
			huh.NewOption("English", i18n.English),
		).
		Value(&selected)
	if err := prompt.runRichForm(i18n.Chinese, field); err != nil {
		return "", err
	}
	return selected, nil
}

func (prompt *setupPrompt) chooseModeRich(locale string) (string, error) {
	selected := string(setupdomain.ModeSkill)
	field := huh.NewSelect[string]().
		Title(i18n.Text(locale, i18n.SetupChooseModeRich)).
		Options(
			huh.NewOption(i18n.Text(locale, i18n.SetupModeSkill), string(setupdomain.ModeSkill)),
			huh.NewOption(i18n.Text(locale, i18n.SetupModeMCP), string(setupdomain.ModeMCP)),
			huh.NewOption(i18n.Text(locale, i18n.SetupModeLAN), string(setupdomain.ModeLAN)),
		).
		Value(&selected)
	if err := prompt.runRichForm(locale, field); err != nil {
		return "", err
	}
	return selected, nil
}

func (prompt *setupPrompt) chooseMCPClientRich(locale string, clients []setupdomain.Client) (string, error) {
	if len(clients) == 0 {
		return "", fmt.Errorf("no supported MCP client is available")
	}
	selected := clients[0].ID
	options := make([]huh.Option[string], 0, len(clients))
	for _, client := range clients {
		options = append(options, huh.NewOption(clientLabel(client), client.ID))
	}
	field := huh.NewSelect[string]().
		Title(i18n.Text(locale, i18n.SetupChooseClientRich)).
		Description(i18n.Text(locale, i18n.SetupMCPClientHint)).
		Options(options...).
		Value(&selected).
		Height(promptHeight(len(options)))
	if err := prompt.runRichForm(locale, field); err != nil {
		return "", err
	}
	return selected, nil
}

func (prompt *setupPrompt) chooseHomeRich(locale string, homes []setupHomeChoice) (string, error) {
	selected := homes[0].ID
	options := make([]huh.Option[string], 0, len(homes))
	for _, home := range homes {
		options = append(options, huh.NewOption(home.Name, home.ID))
	}
	field := huh.NewSelect[string]().
		Title(i18n.Text(locale, i18n.SetupChooseHomeRich)).
		Description(i18n.Text(locale, i18n.SetupHomeHint)).
		Options(options...).
		Value(&selected).
		Height(promptHeight(len(options)))
	if err := prompt.runRichForm(locale, field); err != nil {
		return "", err
	}
	return selected, nil
}

func (prompt *setupPrompt) reuseCurrentAccountRich(locale string) (bool, error) {
	selected := "keep"
	field := huh.NewSelect[string]().
		Title(i18n.Text(locale, i18n.SetupChooseAccountRich)).
		Options(
			huh.NewOption(i18n.Text(locale, i18n.SetupKeepCurrentAccount), "keep"),
			huh.NewOption(i18n.Text(locale, i18n.SetupSwitchAccount), "switch"),
		).
		Value(&selected)
	if err := prompt.runRichForm(locale, field); err != nil {
		return false, err
	}
	return selected == "keep", nil
}

func (prompt *setupPrompt) confirmRich(locale string) (bool, error) {
	confirmed := true
	field := huh.NewConfirm().
		Title(i18n.Text(locale, i18n.SetupConfirmRich)).
		Affirmative(i18n.Text(locale, i18n.SetupConfirmYes)).
		Negative(i18n.Text(locale, i18n.SetupConfirmNo)).
		Value(&confirmed)
	if err := prompt.runRichForm(locale, field); err != nil {
		return false, err
	}
	return confirmed, nil
}

func (prompt *setupPrompt) runRichForm(locale string, field huh.Field) error {
	form := huh.NewForm(huh.NewGroup(field)).
		WithInput(prompt.input).
		WithOutput(prompt.stdout).
		WithTheme(setupPromptTheme()).
		WithKeyMap(setupPromptKeyMap(locale)).
		WithShowHelp(true).
		WithAccessible(prompt.accessible)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return errSetupCancelled
		}
		return err
	}
	return nil
}

func setupPromptTheme() *huh.Theme {
	theme := huh.ThemeBase()
	accent := lipgloss.AdaptiveColor{Light: "#8A6500", Dark: "#FFD21C"}
	muted := lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#8B93A1"}
	green := lipgloss.AdaptiveColor{Light: "#087A55", Dark: "#48D597"}
	normal := lipgloss.AdaptiveColor{Light: "#171717", Dark: "#F4F4F5"}
	theme.Focused.Base = lipgloss.NewStyle().BorderStyle(lipgloss.ThickBorder()).BorderLeft(true).BorderForeground(accent).PaddingLeft(1)
	theme.Focused.Title = lipgloss.NewStyle().Foreground(normal).Bold(true)
	theme.Focused.Description = lipgloss.NewStyle().Foreground(muted)
	theme.Focused.SelectSelector = lipgloss.NewStyle().Foreground(accent).SetString("❯ ")
	theme.Focused.Option = lipgloss.NewStyle().Foreground(normal)
	theme.Focused.MultiSelectSelector = lipgloss.NewStyle().Foreground(accent).SetString("❯ ")
	theme.Focused.SelectedOption = lipgloss.NewStyle().Foreground(green)
	theme.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(green).SetString("◉ ")
	theme.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(normal)
	theme.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(muted).SetString("◯ ")
	theme.Focused.FocusedButton = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(accent).Padding(0, 2)
	theme.Focused.BlurredButton = lipgloss.NewStyle().Foreground(normal).Background(lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#33363D"}).Padding(0, 2)
	theme.Blurred = theme.Focused
	theme.Blurred.Base = lipgloss.NewStyle().PaddingLeft(2)
	theme.Blurred.SelectSelector = lipgloss.NewStyle()
	theme.Blurred.MultiSelectSelector = lipgloss.NewStyle()
	theme.Group.Title = theme.Focused.Title
	theme.Group.Description = theme.Focused.Description
	return theme
}

func setupPromptKeyMap(locale string) *huh.KeyMap {
	keyMap := huh.NewDefaultKeyMap()
	if locale == i18n.Chinese {
		keyMap.Select.Up.SetHelp("↑", "上移")
		keyMap.Select.Down.SetHelp("↓", "下移")
		keyMap.Select.Submit.SetHelp("回车", "确认")
		keyMap.Select.Filter.SetHelp("/", "搜索")
		keyMap.MultiSelect.Up.SetHelp("↑", "上移")
		keyMap.MultiSelect.Down.SetHelp("↓", "下移")
		keyMap.MultiSelect.Toggle.SetHelp("空格", "勾选")
		keyMap.MultiSelect.Submit.SetHelp("回车", "确认")
		keyMap.MultiSelect.Filter.SetHelp("/", "搜索")
		keyMap.MultiSelect.SelectAll.SetHelp("ctrl+a", "全选")
		keyMap.MultiSelect.SelectNone.SetHelp("ctrl+a", "取消全选")
		keyMap.Confirm.Toggle.SetHelp("←/→", "切换")
		keyMap.Confirm.Submit.SetHelp("回车", "确认")
	}
	return keyMap
}

func clientLabel(client setupdomain.Client) string {
	if strings.EqualFold(client.Name, client.ID) {
		return client.Name
	}
	return fmt.Sprintf("%s  ·  %s", client.Name, client.ID)
}

func promptHeight(optionCount int) int {
	if optionCount < 3 {
		return 4
	}
	if optionCount > 10 {
		return 10
	}
	return optionCount + 1
}
