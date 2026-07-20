package lanruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type targetCapabilities struct {
	actions map[string]bool
	flows   map[string]bool
}

func (adapter *Adapter) capabilitiesForTarget(ctx context.Context, target Target) (Target, targetCapabilities, error) {
	resolved, err := adapter.resolveTarget(ctx, target)
	if err != nil {
		return Target{}, targetCapabilities{}, err
	}
	for _, role := range []toolRole{roleDetail, roleState, roleList} {
		tool, ok := adapter.catalog.tools[role]
		if !ok {
			continue
		}
		arguments, buildErr := buildToolArguments(tool.InputSchema, role, operationValues{target: resolved})
		if buildErr != nil {
			continue
		}
		result, callErr := adapter.call(ctx, tool, arguments, false)
		if callErr != nil {
			continue
		}
		capabilities := capabilitiesFromData(result.Data, resolved)
		if len(capabilities.actions) > 0 || len(capabilities.flows) > 0 {
			return resolved, capabilities, nil
		}
	}
	return resolved, targetCapabilities{}, unsupported("target does not expose action or flow capability evidence")
}

func capabilitiesFromData(data any, target Target) targetCapabilities {
	result := targetCapabilities{actions: map[string]bool{}, flows: map[string]bool{}}
	for _, item := range collectObjectMaps(data) {
		if !targetMapMatches(item, target) {
			continue
		}
		capabilities := capabilitiesFromMap(item)
		for name := range capabilities.actions {
			result.actions[name] = true
		}
		for name := range capabilities.flows {
			result.flows[name] = true
		}
	}
	return result
}

func targetMapMatches(item map[string]any, target Target) bool {
	candidate := targetFromMap(item)
	if target.ID != "" {
		if candidate.ID == "" || candidate.ID != target.ID {
			return false
		}
	} else if candidate.Name == "" || !strings.EqualFold(candidate.Name, target.Name) {
		return false
	}
	if target.Type != "" && candidate.Type != "" && !strings.EqualFold(target.Type, candidate.Type) {
		return false
	}
	itemHouseID := firstMapString(item, "houseId", "homeId")
	return target.HouseID == "" || itemHouseID == "" || itemHouseID == target.HouseID
}

func capabilitiesFromMap(item map[string]any) targetCapabilities {
	result := targetCapabilities{actions: map[string]bool{}, flows: map[string]bool{}}
	sources := []map[string]any{item, asMap(item["capabilities"]), asMap(item["capability"]), asMap(item["detail"])}
	for _, source := range sources {
		for _, key := range []string{"supportActions", "support_actions", "actions"} {
			collectCapabilityNames(result.actions, source[key], "actionName", "action", "name", "id")
		}
		for _, key := range []string{"supportFlows", "support_flows", "flows"} {
			collectCapabilityNames(result.flows, source[key], "flowName", "flow", "mode", "name", "id")
		}
	}
	return result
}

func collectCapabilityNames(target map[string]bool, value any, keys ...string) {
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return
		}
		var decoded any
		if (strings.HasPrefix(text, "[") || strings.HasPrefix(text, "{")) && json.Unmarshal([]byte(text), &decoded) == nil {
			collectCapabilityNames(target, decoded, keys...)
			return
		}
		for _, name := range strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == ';' || r == '|' }) {
			addCapabilityName(target, name)
		}
	case []any:
		for _, item := range typed {
			collectCapabilityNames(target, item, keys...)
		}
	case map[string]any:
		for _, key := range keys {
			if name := strings.TrimSpace(fmt.Sprint(typed[key])); name != "" && name != "<nil>" {
				addCapabilityName(target, name)
				return
			}
		}
		for name, enabled := range typed {
			if allowed, ok := enabled.(bool); ok && allowed {
				addCapabilityName(target, name)
			}
		}
	}
}

func addCapabilityName(target map[string]bool, name string) {
	if normalized := strings.ToLower(strings.TrimSpace(name)); normalized != "" {
		target[normalized] = true
	}
}

func (capabilities targetCapabilities) supportsAction(name string) bool {
	return capabilities.actions[strings.ToLower(strings.TrimSpace(name))]
}

func (capabilities targetCapabilities) supportsFlow(name string) bool {
	return capabilities.flows[strings.ToLower(strings.TrimSpace(name))]
}
