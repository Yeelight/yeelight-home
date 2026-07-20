package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/lanmcp"
	"github.com/yeelight/yeelight-home/internal/lanruntime"
)

type menuEntity struct {
	Type     string
	ID       string
	Name     string
	HouseID  string
	RoomID   string
	RoomName string
}

func (menu *menuSession) loadEntities() ([]menuEntity, error) {
	contextInfo, err := menu.app.resolveRuntimeContext(cliFlags{values: map[string]string{}})
	if err != nil {
		return nil, err
	}
	if !contextInfo.TokenPresent && contextInfo.ControlMode != controlModeCloud && contextInfo.LANEndpoint != "" {
		client, err := lanmcp.NewClient(contextInfo.LANEndpoint, lanmcp.Options{})
		if err != nil {
			return nil, err
		}
		adapter, err := lanruntime.Connect(context.Background(), lanruntime.Options{Client: client})
		if err != nil {
			return nil, err
		}
		targets, err := adapter.ListTargets(context.Background(), contextInfo.HouseID)
		if err != nil {
			return nil, err
		}
		entities := make([]menuEntity, 0, len(targets))
		for _, target := range targets {
			entities = append(entities, menuEntity{Type: target.Type, ID: target.ID, Name: target.Name, HouseID: target.HouseID, RoomName: target.Room})
		}
		return entities, nil
	}
	response, err := menu.invoke("entity.list", map[string]any{"houseId": contextInfo.HouseID})
	if err != nil {
		return nil, err
	}
	rows, _ := response.Result["entities"].([]any)
	entities := make([]menuEntity, 0, len(rows))
	roomNames := map[string]string{}
	for _, row := range rows {
		item, _ := row.(map[string]any)
		entity := menuEntity{
			Type: stringValue(item["type"]), ID: stringValue(item["id"]), Name: stringValue(item["name"]),
			HouseID: stringValue(item["houseId"]), RoomID: stringValue(item["roomId"]), RoomName: stringValue(item["roomName"]),
		}
		if entity.ID == "" || entity.Name == "" {
			continue
		}
		entities = append(entities, entity)
		if entity.Type == "room" {
			roomNames[entity.ID] = entity.Name
		}
	}
	for index := range entities {
		if entities[index].RoomName == "" {
			entities[index].RoomName = roomNames[entities[index].RoomID]
		}
	}
	return entities, nil
}

func (menu *menuSession) selectEntity(types ...string) (menuEntity, bool) {
	entities, err := menu.loadEntities()
	if err != nil {
		_, _ = fmt.Fprintf(menu.stderr, "%s: %v\n", menu.text("读取家庭失败", "Could not read the home"), err)
		return menuEntity{}, false
	}
	allowed := map[string]bool{}
	for _, entityType := range types {
		allowed[entityType] = true
	}
	filtered := make([]menuEntity, 0, len(entities))
	for _, entity := range entities {
		if allowed[entity.Type] {
			filtered = append(filtered, entity)
		}
	}
	if len(filtered) == 0 {
		_, _ = fmt.Fprintln(menu.stdout, menu.text("没有找到可选项目。", "No matching items were found."))
		return menuEntity{}, false
	}
	for index, entity := range filtered {
		room := ""
		if entity.RoomName != "" {
			room = " · " + entity.RoomName
		}
		_, _ = fmt.Fprintf(menu.stdout, "%d. %s%s\n", index+1, entity.Name, room)
	}
	index, ok := menu.chooseIndex(menu.text("0. 返回\n请选择：", "0. Back\nChoose: "), len(filtered))
	if !ok {
		return menuEntity{}, false
	}
	return filtered[index-1], true
}

func (menu *menuSession) invoke(intent string, parameters map[string]any) (contract.Response, error) {
	menu.request++
	response, err := menu.app.invoke(context.Background(), contract.Request{
		ContractVersion: contract.Version,
		RequestID:       fmt.Sprintf("menu-%d", menu.request),
		Locale:          menu.locale,
		Utterance:       menu.text("在家庭工作台中执行操作", "Run an operation from the home console"),
		Intent:          intent,
		Parameters:      parameters,
	})
	if err != nil {
		return contract.Response{}, err
	}
	if response.Status == "error" || response.Status == "auth_required" || response.Status == "blocked" {
		return response, fmt.Errorf("%s", response.UserMessage)
	}
	return response, nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
