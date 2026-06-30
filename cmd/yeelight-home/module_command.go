package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/contract"
)

type moduleCommandSpec struct {
	Intent       string
	Utterance    string
	EntityType   string
	TargetIDKeys []string
	TargetName   bool
}

func (app *app) runModuleCommand(resource string, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintf(stderr, "usage: yeelight-home %s <%s> [flags]\n", resource, strings.Join(moduleCommandNames(resource), "|"))
		return exitInvalidInput
	}
	action := args[0]
	spec, ok := moduleCommands[resource][action]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "unsupported %s command %q\n", resource, action)
		return exitInvalidInput
	}
	flags, err := parseFlags(args[1:])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s %s: %v\n", resource, action, err)
		return exitInvalidInput
	}
	request, err := buildModuleRequest(resource, action, spec, flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s %s: %v\n", resource, action, err)
		return exitInvalidInput
	}
	response, err := app.invokeWithFlags(context.Background(), request, flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s %s: %v\n", resource, action, err)
		return exitInternalError
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, response)
	}
	return writeModuleText(stdout, stderr, response)
}

func buildModuleRequest(resource string, action string, spec moduleCommandSpec, flags cliFlags) (contract.Request, error) {
	parameters, err := moduleParameters(flags)
	if err != nil {
		return contract.Request{}, err
	}
	if spec.Intent == "light.power.set" && resource == "light" {
		parameters["power"] = action == "on"
	}
	if value := firstFlagValue(flags, spec.TargetIDKeys...); value != "" {
		parameters[targetParameterName(spec.TargetIDKeys)] = value
	}
	if spec.TargetName {
		if value := firstFlagValue(flags, "name", resource+"-name", "target-name", "entity-name"); value != "" {
			parameters["name"] = value
		}
	}
	targets := moduleTargets(spec, parameters)
	return contract.Request{
		ContractVersion: contract.Version,
		RequestID:       moduleRequestID(resource, action),
		Locale:          "zh-CN",
		Utterance:       flags.string("utterance", spec.Utterance),
		Intent:          spec.Intent,
		Targets:         targets,
		Parameters:      parameters,
	}, nil
}

func moduleParameters(flags cliFlags) (map[string]any, error) {
	parameters := map[string]any{}
	if raw := flags.string("params-json", ""); raw != "" {
		if err := json.Unmarshal([]byte(raw), &parameters); err != nil {
			return nil, fmt.Errorf("invalid --params-json: %w", err)
		}
	}
	if raw := flags.string("set", ""); raw != "" {
		for _, item := range strings.Split(raw, ",") {
			key, value, ok := strings.Cut(item, "=")
			if !ok || strings.TrimSpace(key) == "" {
				return nil, fmt.Errorf("invalid --set item %q, expected key=value", item)
			}
			parameters[strings.TrimSpace(key)] = parseModuleScalar(value)
		}
	}
	for flagName, parameterName := range moduleParameterFlags() {
		if value := flags.string(flagName, ""); value != "" {
			parameters[parameterName] = parseModuleScalar(value)
		}
	}
	return parameters, nil
}

func moduleParameterFlags() map[string]string {
	return map[string]string{
		"area-code":          "areaCode",
		"area-id":            "areaId",
		"area-ids":           "areaIds",
		"automation-id":      "automationId",
		"automation-ids":     "automationIds",
		"button-event-id":    "buttonEventId",
		"brightness":         "brightness",
		"color":              "color",
		"color-temperature":  "colorTemperature",
		"component-id":       "componentId",
		"ct":                 "colorTemperature",
		"delta":              "delta",
		"description":        "description",
		"device-id":          "deviceId",
		"device-ids":         "deviceIds",
		"entity-id":          "entityId",
		"entity-ids":         "entityIds",
		"entity-name":        "entityName",
		"faq-id":             "faqId",
		"favorite-id":        "favoriteId",
		"favorite-ids":       "favoriteIds",
		"gateway-id":         "gatewayId",
		"gateway-ids":        "gatewayIds",
		"group-id":           "groupId",
		"group-ids":          "groupIds",
		"hex":                "hex",
		"id":                 "id",
		"ids":                "ids",
		"keyword":            "keyword",
		"knob-id":            "knobId",
		"limit":              "limit",
		"material-code":      "materialCode",
		"member-id":          "memberId",
		"meshgroup-id":       "meshgroupId",
		"model":              "model",
		"multi-field":        "multiField",
		"name":               "name",
		"node-id":            "nodeId",
		"panel-id":           "panelId",
		"page-no":            "pageNo",
		"page-size":          "pageSize",
		"parent-id":          "parentId",
		"property":           "propertyName",
		"product-id":         "productId",
		"product-ids":        "productIds",
		"product-model":      "productModel",
		"product-name":       "productName",
		"product-short-name": "productShortName",
		"product-sku":        "productSku",
		"product-spu":        "productSpu",
		"progress-id":        "progressId",
		"repeat-type":        "repeatType",
		"res-id":             "resId",
		"res-name":           "resName",
		"room-id":            "roomId",
		"room-ids":           "roomIds",
		"room-name":          "roomName",
		"scene-id":           "sceneId",
		"scene-ids":          "sceneIds",
		"schema-id":          "schemaId",
		"share-id":           "shareId",
		"sku":                "sku",
		"spu":                "spu",
		"status":             "status",
		"target":             "target",
		"target-name":        "name",
		"target-room-id":     "targetRoomId",
		"target-room-name":   "targetRoomName",
		"type":               "type",
		"type-id":            "typeId",
		"user-role":          "userRole",
		"value":              "value",
	}
}

func moduleTargets(spec moduleCommandSpec, parameters map[string]any) []map[string]any {
	if spec.EntityType == "" {
		return nil
	}
	target := map[string]any{"entityType": spec.EntityType}
	for _, key := range moduleTargetIDParameterKeys(spec.EntityType) {
		if value := requestString(parameters[key]); value != "" {
			target["id"] = value
			break
		}
	}
	if value := requestString(parameters["name"]); value != "" {
		target["name"] = value
	}
	for _, key := range []string{"roomId", "targetRoomId", "roomName", "targetRoomName"} {
		if value := requestString(parameters[key]); value != "" {
			target[key] = value
		}
	}
	return []map[string]any{target}
}

func moduleTargetIDParameterKeys(entityType string) []string {
	switch entityType {
	case "device":
		return []string{"deviceId", "gatewayId", "panelId", "knobId", "sensorId", "meshgroupId", "entityId", "id"}
	case "room":
		return []string{"roomId", "entityId", "id"}
	case "scene":
		return []string{"sceneId", "entityId", "id"}
	case "automation":
		return []string{"automationId", "entityId", "id"}
	case "group":
		return []string{"groupId", "meshgroupId", "entityId", "id"}
	case "area":
		return []string{"areaId", "entityId", "id"}
	default:
		return []string{"entityId", "id"}
	}
}

func targetParameterName(keys []string) string {
	for _, key := range keys {
		if strings.HasSuffix(key, "Id") {
			return key
		}
		if strings.HasSuffix(key, "-id") {
			prefix := strings.TrimSuffix(key, "-id")
			parts := strings.Split(prefix, "-")
			for index := 1; index < len(parts); index++ {
				parts[index] = strings.ToUpper(parts[index][:1]) + parts[index][1:]
			}
			return strings.Join(parts, "") + "Id"
		}
	}
	return "id"
}

func firstFlagValue(flags cliFlags, names ...string) string {
	for _, name := range names {
		if value := flags.string(name, ""); value != "" {
			return value
		}
	}
	return ""
}

func parseModuleScalar(value string) any {
	trimmed := strings.TrimSpace(value)
	switch strings.ToLower(trimmed) {
	case "true":
		return true
	case "false":
		return false
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
		switch decoded.(type) {
		case string, float64, bool, []any, map[string]any:
			return decoded
		}
	}
	return trimmed
}

func moduleRequestID(resource string, action string) string {
	return fmt.Sprintf("cli-%s-%s-%d", resource, action, time.Now().UnixNano())
}

func moduleCommandNames(resource string) []string {
	commands := moduleCommands[resource]
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func moduleResourceNames() []string {
	names := make([]string, 0, len(moduleCommands))
	for name := range moduleCommands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeModuleText(stdout io.Writer, stderr io.Writer, response contract.Response) int {
	_, _ = fmt.Fprintf(stdout, "%s: %s\n", response.Status, response.UserMessage)
	if response.Result != nil {
		if preview := requestMap(response.Result["preview"]); preview != nil {
			if summary := requestString(preview["summary"]); summary != "" {
				_, _ = fmt.Fprintf(stdout, "preview: %s\n", summary)
			}
		} else if summary := requestString(response.Result["summary"]); summary != "" {
			_, _ = fmt.Fprintf(stdout, "preview: %s\n", summary)
		}
	}
	if response.Clarification != nil {
		if reason := requestString(response.Clarification["reason"]); reason != "" {
			_, _ = fmt.Fprintf(stdout, "reason: %s\n", reason)
		}
	}
	if response.Error != nil {
		_, _ = fmt.Fprintf(stderr, "error: %s\n", response.Error.Code)
	}
	return exitOK
}
