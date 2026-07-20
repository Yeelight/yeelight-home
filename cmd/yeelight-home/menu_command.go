package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/yeelight/yeelight-home/internal/i18n"
)

type menuSession struct {
	app     *app
	reader  *bufio.Reader
	stdout  io.Writer
	stderr  io.Writer
	locale  string
	request uint64
}

func (app *app) runMenu(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	locale := app.menuLocale()
	menu := &menuSession{app: app, reader: bufio.NewReader(stdin), stdout: stdout, stderr: stderr, locale: locale}
	for {
		menu.printMain()
		choice, err := menu.readLine()
		if err != nil {
			if err == io.EOF {
				return exitOK
			}
			_, _ = fmt.Fprintf(stderr, "menu: %v\n", err)
			return exitInternalError
		}
		switch strings.ToLower(choice) {
		case "0", "q", "quit", "exit":
			return exitOK
		case "1", "home", "家庭":
			menu.chooseHome()
		case "2", "room", "rooms", "房间":
			menu.chooseRoom()
		case "3", "device", "devices", "设备":
			menu.chooseDevice()
		case "4", "scene", "scenes", "情景", "场景":
			menu.chooseScene()
		case "5", "light", "lights", "灯光":
			menu.chooseQuickLight()
		case "6", "doctor", "检查":
			_ = menu.app.runDoctor(nil, menu.stdout, menu.stderr)
		default:
			_, _ = fmt.Fprintln(menu.stdout, menu.text("请输入菜单中的数字。", "Enter a number from the menu."))
		}
	}
}

func (app *app) menuLocale() string {
	if contextInfo, err := app.resolveRuntimeContext(cliFlags{values: map[string]string{}}); err == nil && contextInfo.Language != "" {
		return contextInfo.Language
	}
	if locale, ok := i18n.Detect(os.LookupEnv); ok {
		return locale
	}
	return i18n.Chinese
}

func (menu *menuSession) printMain() {
	_, _ = fmt.Fprintln(menu.stdout, menu.text("\nYeelight Home 家庭工作台", "\nYeelight Home console"))
	_, _ = fmt.Fprintln(menu.stdout, menu.text("1. 选择家庭", "1. Choose home"))
	_, _ = fmt.Fprintln(menu.stdout, menu.text("2. 房间", "2. Rooms"))
	_, _ = fmt.Fprintln(menu.stdout, menu.text("3. 设备", "3. Devices"))
	_, _ = fmt.Fprintln(menu.stdout, menu.text("4. 情景", "4. Scenes"))
	_, _ = fmt.Fprintln(menu.stdout, menu.text("5. 常用灯光", "5. Common lighting"))
	_, _ = fmt.Fprintln(menu.stdout, menu.text("6. 安装与连接检查", "6. Installation and connection check"))
	_, _ = fmt.Fprint(menu.stdout, menu.text("0. 退出\n请选择：", "0. Exit\nChoose: "))
}

func (menu *menuSession) readLine() (string, error) {
	line, err := menu.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	if err == io.EOF && line == "" {
		return "", io.EOF
	}
	return strings.TrimSpace(line), nil
}

func (menu *menuSession) chooseIndex(prompt string, count int) (int, bool) {
	_, _ = fmt.Fprint(menu.stdout, prompt)
	value, err := menu.readLine()
	if err != nil {
		return 0, false
	}
	index, err := strconv.Atoi(value)
	if err != nil || index < 0 || index > count {
		_, _ = fmt.Fprintln(menu.stdout, menu.text("选择无效。", "Invalid choice."))
		return 0, false
	}
	return index, index > 0
}

func (menu *menuSession) confirm() bool {
	_, _ = fmt.Fprint(menu.stdout, menu.text("确认执行？[Y/n]：", "Run this operation? [Y/n]: "))
	value, err := menu.readLine()
	if err != nil {
		return false
	}
	value = strings.ToLower(value)
	return value == "" || value == "y" || value == "yes" || value == "是" || value == "确认"
}

func (menu *menuSession) text(chinese, english string) string {
	if menu.locale == i18n.English {
		return english
	}
	return chinese
}
