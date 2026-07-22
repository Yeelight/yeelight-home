package main

import (
	"fmt"
	"strconv"

	"github.com/yeelight/yeelight-home/internal/contract"
)

func (menu *menuSession) chooseHome() {
	response, err := menu.invoke("home.list", map[string]any{})
	if err != nil {
		_, _ = fmt.Fprintf(menu.stderr, "%s: %v\n", menu.text("读取家庭失败", "Could not list homes"), err)
		return
	}
	rows, _ := response.Result["houses"].([]any)
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(menu.stdout, menu.text("没有找到易来家庭。", "No Yeelight homes were found."))
		return
	}
	for index, row := range rows {
		item, _ := row.(map[string]any)
		_, _ = fmt.Fprintf(menu.stdout, "%d. %s\n", index+1, stringValue(item["name"]))
	}
	index, ok := menu.chooseIndex(menu.text("0. 返回\n请选择家庭：", "0. Back\nChoose a home: "), len(rows))
	if !ok {
		return
	}
	selected, _ := rows[index-1].(map[string]any)
	if code := menu.app.runHome([]string{"select", "--house-id", stringValue(selected["id"]), "--json"}, menu.reader, menu.stdout, menu.stderr); code == exitOK {
		_, _ = fmt.Fprintln(menu.stdout, menu.text("已切换家庭。", "Home selected."))
	}
}

func (menu *menuSession) chooseRoom() {
	room, ok := menu.selectEntity("room")
	if !ok {
		return
	}
	_, _ = fmt.Fprint(menu.stdout, menu.text("1. 查看设备\n2. 开灯\n3. 关灯\n4. 调亮度\n0. 返回\n请选择：", "1. View devices\n2. Turn lights on\n3. Turn lights off\n4. Set brightness\n0. Back\nChoose: "))
	choice, _ := menu.readLine()
	switch choice {
	case "1":
		entities, err := menu.loadEntities()
		if err != nil {
			_, _ = fmt.Fprintln(menu.stderr, err)
			return
		}
		for _, entity := range entities {
			if entity.Type == "device" && (entity.RoomID == room.ID || entity.RoomName == room.Name) {
				_, _ = fmt.Fprintf(menu.stdout, "- %s\n", entity.Name)
			}
		}
	case "2":
		menu.runLight(room, "light.power.set", map[string]any{"power": true})
	case "3":
		menu.runLight(room, "light.power.set", map[string]any{"power": false})
	case "4":
		menu.promptBrightness(room)
	}
}

func (menu *menuSession) chooseDevice() {
	device, ok := menu.selectEntity("device")
	if !ok {
		return
	}
	_, _ = fmt.Fprint(menu.stdout, menu.text("1. 查看状态\n2. 开灯\n3. 关灯\n4. 调亮度\n0. 返回\n请选择：", "1. View state\n2. Turn on\n3. Turn off\n4. Set brightness\n0. Back\nChoose: "))
	choice, _ := menu.readLine()
	switch choice {
	case "1":
		menu.printResponse(menu.invoke("state.query", menu.targetParameters(device)))
	case "2":
		menu.runLight(device, "light.power.set", map[string]any{"power": true})
	case "3":
		menu.runLight(device, "light.power.set", map[string]any{"power": false})
	case "4":
		menu.promptBrightness(device)
	}
}

func (menu *menuSession) chooseScene() {
	scene, ok := menu.selectEntity("scene")
	if !ok || !menu.confirm() {
		return
	}
	menu.printResponse(menu.invoke("scene.execute", map[string]any{"houseId": scene.HouseID, "sceneId": scene.ID}))
}

func (menu *menuSession) chooseQuickLight() {
	target, ok := menu.selectEntity("room", "group", "device")
	if !ok {
		return
	}
	_, _ = fmt.Fprint(menu.stdout, menu.text("1. 开灯\n2. 关灯\n3. 调亮度\n0. 返回\n请选择：", "1. Turn on\n2. Turn off\n3. Set brightness\n0. Back\nChoose: "))
	choice, _ := menu.readLine()
	switch choice {
	case "1":
		menu.runLight(target, "light.power.set", map[string]any{"power": true})
	case "2":
		menu.runLight(target, "light.power.set", map[string]any{"power": false})
	case "3":
		menu.promptBrightness(target)
	}
}

func (menu *menuSession) promptBrightness(target menuEntity) {
	_, _ = fmt.Fprint(menu.stdout, menu.text("亮度（1-100）：", "Brightness (1-100): "))
	value, err := menu.readLine()
	brightness, parseErr := strconv.Atoi(value)
	if err != nil || parseErr != nil || brightness < 1 || brightness > 100 {
		_, _ = fmt.Fprintln(menu.stdout, menu.text("亮度无效。", "Invalid brightness."))
		return
	}
	menu.runLight(target, "light.brightness.set", map[string]any{"brightness": brightness})
}

func (menu *menuSession) runLight(target menuEntity, intent string, values map[string]any) {
	if !menu.confirm() {
		return
	}
	parameters := menu.targetParameters(target)
	for key, value := range values {
		parameters[key] = value
	}
	menu.printResponse(menu.invoke(intent, parameters))
}

func (menu *menuSession) targetParameters(target menuEntity) map[string]any {
	return map[string]any{
		"houseId": target.HouseID, "targetType": target.Type,
		"targetId": target.ID, "roomName": target.RoomName,
	}
}

func (menu *menuSession) printResponse(response contractResponse, err error) {
	if err != nil {
		_, _ = fmt.Fprintln(menu.stderr, err)
		return
	}
	_, _ = fmt.Fprintln(menu.stdout, response.UserMessage)
}

type contractResponse = contract.Response
