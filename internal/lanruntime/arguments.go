package lanruntime

import (
	"fmt"
	"slices"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type operationValues struct {
	requestID  string
	target     Target
	property   string
	value      any
	valueSet   bool
	properties map[string]any
	action     string
	actionName string
	payload    map[string]any
	flow       any
	duration   any
	delay      any
}

func buildToolArguments(schema map[string]any, role toolRole, values operationValues) (map[string]any, error) {
	properties := schemaProperties(schema)
	if len(properties) == 0 {
		if len(schemaRequired(schema)) == 0 {
			return map[string]any{}, nil
		}
		return nil, unsupported("tool schema does not expose mappable properties")
	}
	arguments := map[string]any{}
	for name, propertySchema := range properties {
		propertyDefinition := asMap(propertySchema)
		if len(schemaProperties(propertyDefinition)) > 0 {
			nested, err := buildToolArguments(asMap(propertySchema), role, values)
			if err == nil && len(nested) > 0 {
				arguments[name] = nested
				continue
			}
		}
		if value, ok := mappedArgumentValue(name, propertyDefinition, role, values); ok {
			if !mappedValueAllowed(propertyDefinition, value) {
				continue
			}
			arguments[name] = value
		}
	}
	for _, required := range schemaRequired(schema) {
		if value, ok := arguments[required]; !ok || emptyArgument(value) {
			return nil, unsupported(fmt.Sprintf("required schema field %s cannot be mapped", required))
		}
	}
	if len(arguments) == 0 && role != roleList {
		return nil, unsupported("tool schema has no compatible fields")
	}
	return arguments, nil
}

func mappedValueAllowed(schema map[string]any, value any) bool {
	allowed := stringEnumValues(schema)
	if len(allowed) == 0 {
		return true
	}
	text, ok := value.(string)
	return ok && slices.Contains(allowed, strings.ToLower(strings.TrimSpace(text)))
}

func mappedArgumentValue(name string, schema map[string]any, role toolRole, values operationValues) (any, bool) {
	normalized := normalizedArgumentName(name)
	switch normalized {
	case "requestid":
		return nonEmptyValue(values.requestID)
	case "houseid", "homeid":
		return nonEmptyValue(values.target.HouseID)
	case "nodeid", "deviceid", "targetid", "entityid":
		return nonEmptyValue(values.target.ID)
	case "id":
		if role == roleDetail {
			return nonEmptyValue(values.target.ID)
		}
	case "nodename", "devicename", "targetname", "entityname", "name":
		if role == roleScene {
			return nonEmptyValue(values.target.Name)
		}
		return nonEmptyValue(values.target.Name)
	case "roomname":
		return nonEmptyValue(values.target.Room)
	case "targettype":
		return targetTypeValue(values.target.Type, schema)
	case "nodetype", "entitytype", "type":
		return nodeTypeValue(values.target.Type, schema)
	case "property", "propertyname", "propname", "propertyid":
		return nonEmptyValue(values.property)
	case "capability":
		if role == roleAction {
			return nonEmptyValue(values.actionName)
		}
		if role == roleFlow {
			return nonEmptyValue(flowCapabilityName(values.flow))
		}
		return nonEmptyValue(gatewayCapabilityName(values.property))
	case "value":
		if role == roleAction {
			return values.payload, true
		}
		if role == roleFlow {
			return values.flow, values.flow != nil
		}
		return values.value, values.valueSet
	case "properties", "propertyset":
		if len(values.properties) > 0 {
			return values.properties, true
		}
		if values.property != "" && values.valueSet {
			return map[string]any{values.property: values.value}, true
		}
	case "actions":
		return controlActions(schema, role, values)
	case "sceneid":
		return nonEmptyValue(values.target.ID)
	case "scenename":
		return nonEmptyValue(values.target.Name)
	case "command", "operation":
		return nonEmptyValue(values.action)
	case "action", "actionname":
		return nonEmptyValue(firstNonEmpty(values.actionName, values.action))
	case "params", "parameters":
		if role == roleAction || role == roleFlow {
			return values.payload, true
		}
		return controlParams(values.properties)
	case "payload":
		if len(values.payload) > 0 {
			return values.payload, true
		}
	case "flow":
		return values.flow, values.flow != nil
	case "duration":
		return values.duration, values.duration != nil
	case "delay":
		return values.delay, values.delay != nil
	case "dryrun":
		return false, true
	case "confirmsideeffect", "confirmed":
		return true, true
	}
	return nil, false
}

func gatewayCapabilityName(property string) string {
	switch name := semantic.PropertyName(property); name {
	case semantic.FieldColorTemperature:
		return "color_temperature"
	case semantic.FieldColor:
		return "color_rgb"
	default:
		return name
	}
}

func flowCapabilityName(flow any) string {
	if name, ok := flow.(string); ok {
		return strings.TrimSpace(name)
	}
	if item, ok := flow.(map[string]any); ok {
		for _, key := range []string{"flowName", "name", "mode", "id"} {
			if name := strings.TrimSpace(fmt.Sprint(item[key])); name != "" && name != "<nil>" {
				return name
			}
		}
	}
	return ""
}

func targetTypeValue(value string, schema map[string]any) (any, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return nil, false
	}
	allowed := stringEnumValues(schema)
	if slices.Contains(allowed, "node") {
		if value == "scene" && slices.Contains(allowed, "scene") {
			return "scene", true
		}
		return "node", true
	}
	return nodeTypeValue(value, schema)
}

func stringEnumValues(schema map[string]any) []string {
	values, _ := schema["enum"].([]any)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			result = append(result, strings.ToLower(strings.TrimSpace(text)))
		}
	}
	return result
}

func controlActions(schema map[string]any, role toolRole, values operationValues) (any, bool) {
	itemSchema := asMap(schema["items"])
	if len(itemSchema) == 0 {
		return nil, false
	}
	action, err := buildToolArguments(itemSchema, role, values)
	if err != nil || len(action) == 0 {
		return nil, false
	}
	return []any{action}, true
}

func nodeTypeValue(value string, schema map[string]any) (any, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}
	if schema["type"] == "integer" || schema["type"] == "number" {
		ids := map[string]int{"room": 1, "device": 2, "area": 3, "group": 4, "home": 5, "scene": 6}
		id, ok := ids[strings.ToLower(value)]
		return id, ok
	}
	return value, true
}

func controlParams(properties map[string]any) (any, bool) {
	if len(properties) == 0 {
		return nil, false
	}
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	params := make([]any, 0, len(keys))
	for _, key := range keys {
		params = append(params, map[string]any{"propName": key, "value": properties[key]})
	}
	return params, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func schemaProperties(schema map[string]any) map[string]any {
	return asMap(schema["properties"])
}

func schemaRequired(schema map[string]any) []string {
	values, _ := schema["required"].([]any)
	if direct, ok := schema["required"].([]string); ok {
		return direct
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func asMap(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func normalizedArgumentName(value string) string {
	return strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.TrimSpace(value)))
}

func nonEmptyValue(value string) (any, bool) {
	value = strings.TrimSpace(value)
	return value, value != ""
}

func emptyArgument(value any) bool {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) == ""
	}
	return value == nil
}

func unsupported(message string) error {
	return &Error{Kind: ErrorUnsupported, Stage: "schema", Message: message}
}
