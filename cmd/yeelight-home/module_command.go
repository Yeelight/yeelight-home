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
	"github.com/yeelight/yeelight-home/internal/semantic"
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
		parameters[semantic.FieldPower] = action == "on"
	}
	if value := firstFlagValue(flags, spec.TargetIDKeys...); value != "" {
		parameters[targetParameterName(spec.TargetIDKeys)] = value
	}
	if spec.TargetName {
		if value := firstFlagValue(flags, "name", resource+"-name", "target-name", "entity-name"); value != "" {
			parameters[semantic.FieldName] = value
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
		"area-code":          semantic.FieldAreaCode,
		"area-id":            semantic.FieldAreaID,
		"area-ids":           semantic.FieldAreaIDs,
		"automation-id":      semantic.FieldAutomationID,
		"automation-ids":     semantic.FieldAutomationIDs,
		"button-event-id":    semantic.FieldButtonEventID,
		"brightness":         semantic.FieldBrightness,
		"color":              semantic.FieldColor,
		"color-temperature":  semantic.FieldColorTemperature,
		"confirmed":          semantic.FieldConfirmed,
		"delta":              semantic.FieldDelta,
		"description":        semantic.FieldDescription,
		"device-id":          semantic.FieldDeviceID,
		"device-ids":         semantic.FieldDeviceIDs,
		"entity-id":          semantic.FieldEntityID,
		"entity-ids":         semantic.FieldEntityIDs,
		"entity-name":        semantic.FieldEntityName,
		"faq-id":             semantic.FieldFAQID,
		"favorite-id":        semantic.FieldFavoriteID,
		"favorite-ids":       semantic.FieldFavoriteIDs,
		"gateway-id":         semantic.FieldGatewayID,
		"gateway-ids":        semantic.FieldGatewayIDs,
		"group-capability":   semantic.FieldGroupCapability,
		"group-category":     semantic.FieldGroupCategory,
		"group-id":           semantic.FieldGroupID,
		"group-ids":          semantic.FieldGroupIDs,
		"hex":                semantic.FieldHex,
		"id":                 semantic.FieldID,
		"ids":                semantic.FieldIDs,
		"keyword":            semantic.FieldKeyword,
		"knob-id":            semantic.FieldKnobID,
		"limit":              semantic.FieldLimit,
		"member-id":          semantic.FieldMemberID,
		"member-name":        semantic.FieldMemberName,
		"meshgroup-id":       semantic.FieldMeshGroupID,
		"model":              semantic.FieldModel,
		"multi-field":        semantic.FieldMultiField,
		"name":               semantic.FieldName,
		"node-id":            semantic.FieldNodeID,
		"panel-id":           semantic.FieldPanelID,
		"page-no":            semantic.FieldPageNo,
		"page-size":          semantic.FieldPageSize,
		"parent-id":          semantic.FieldParentID,
		"property":           semantic.FieldProperty,
		"sku-code":           semantic.FieldSKUCode,
		"capability-pid":     semantic.FieldCapabilityPID,
		"capability-pids":    semantic.FieldCapabilityPIDs,
		"product-model":      semantic.FieldProductModel,
		"product-name":       semantic.FieldProductName,
		"product-short-name": semantic.FieldProductShortName,
		"product-sku":        semantic.FieldProductSKU,
		"product-spu":        semantic.FieldProductSPU,
		"progress-id":        semantic.FieldProgressID,
		"repeat":             semantic.FieldRepeat,
		"room-id":            semantic.FieldRoomID,
		"room-ids":           semantic.FieldRoomIDs,
		"room-name":          semantic.FieldRoomName,
		"scene-id":           semantic.FieldSceneID,
		"scene-ids":          semantic.FieldSceneIDs,
		"schema-id":          semantic.FieldSchemaID,
		"share-id":           semantic.FieldShareID,
		"sku":                semantic.FieldSKU,
		"spu":                semantic.FieldSPU,
		"status":             semantic.FieldStatus,
		"target":             semantic.FieldTarget,
		"target-id":          semantic.FieldTargetID,
		"target-name":        semantic.FieldTargetName,
		"target-room-id":     semantic.FieldTargetRoomID,
		"target-room-name":   semantic.FieldTargetRoomName,
		"target-type":        semantic.FieldTargetType,
		"type":               semantic.FieldType,
		"user-role":          semantic.FieldUserRole,
		"value":              semantic.FieldValue,
	}
}

func moduleTargets(spec moduleCommandSpec, parameters map[string]any) []map[string]any {
	if spec.EntityType == "" {
		return nil
	}
	target := map[string]any{semantic.FieldEntityType: spec.EntityType}
	for _, key := range moduleTargetIDParameterKeys(spec.EntityType) {
		if value := requestString(parameters[key]); value != "" {
			target[semantic.FieldID] = value
			break
		}
	}
	if value := requestString(parameters[semantic.FieldName]); value != "" {
		target[semantic.FieldName] = value
	}
	for _, key := range []string{semantic.FieldRoomID, semantic.FieldTargetRoomID, semantic.FieldRoomName, semantic.FieldTargetRoomName} {
		if value := requestString(parameters[key]); value != "" {
			target[key] = value
		}
	}
	return []map[string]any{target}
}

func moduleTargetIDParameterKeys(entityType string) []string {
	switch entityType {
	case "device":
		return []string{semantic.FieldDeviceID, semantic.FieldGatewayID, semantic.FieldPanelID, semantic.FieldKnobID, semantic.FieldSensorID, semantic.FieldMeshGroupID, semantic.FieldEntityID, semantic.FieldID}
	case "room":
		return []string{semantic.FieldRoomID, semantic.FieldEntityID, semantic.FieldID}
	case "scene":
		return []string{semantic.FieldSceneID, semantic.FieldEntityID, semantic.FieldID}
	case "automation":
		return []string{semantic.FieldAutomationID, semantic.FieldEntityID, semantic.FieldID}
	case "group":
		return []string{semantic.FieldGroupID, semantic.FieldMeshGroupID, semantic.FieldEntityID, semantic.FieldID}
	case "area":
		return []string{semantic.FieldAreaID, semantic.FieldEntityID, semantic.FieldID}
	default:
		return []string{semantic.FieldEntityID, semantic.FieldID}
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
	return semantic.FieldID
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
		if preview := requestMap(response.Result[semantic.FieldPreview]); preview != nil {
			if summary := requestString(preview[semantic.FieldSummary]); summary != "" {
				_, _ = fmt.Fprintf(stdout, "preview: %s\n", summary)
			}
		} else if summary := requestString(response.Result[semantic.FieldSummary]); summary != "" {
			_, _ = fmt.Fprintf(stdout, "preview: %s\n", summary)
		}
	}
	if response.Clarification != nil {
		if reason := requestString(response.Clarification[semantic.FieldReason]); reason != "" {
			_, _ = fmt.Fprintf(stdout, "reason: %s\n", reason)
		}
	}
	if response.Error != nil {
		_, _ = fmt.Fprintf(stderr, "error: %s\n", response.Error.Code)
	}
	return exitOK
}
