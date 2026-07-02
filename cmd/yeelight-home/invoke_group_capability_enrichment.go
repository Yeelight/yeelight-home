package main

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func enrichGroupCreatePayload(
	ctx context.Context,
	endpoint api.Endpoint,
	houseID string,
	authorization string,
	clientID string,
	groupCapability string,
	payload map[string]any,
) (int, []string, string) {
	deviceIDs := valueIDList(payload[semantic.FieldDeviceIDs])
	if len(deviceIDs) == 0 {
		return 0, nil, ""
	}
	if strings.TrimSpace(groupCapability) == "" {
		return 0, nil, ""
	}
	componentID, apiCalls, warnings, reason := deriveGroupComponentID(ctx, endpoint, houseID, authorization, clientID, groupCapability, deviceIDs)
	if reason != "" {
		return apiCalls, warnings, reason
	}
	if componentID > 0 {
		payload[semantic.InternalCloudComponentIDField()] = float64(componentID)
	}
	return apiCalls, warnings, ""
}

func deriveGroupComponentID(
	ctx context.Context,
	endpoint api.Endpoint,
	houseID string,
	authorization string,
	clientID string,
	groupCapability string,
	deviceIDs []string,
) (int, int, []string, string) {
	common := map[int]int{}
	apiCalls := 0
	warnings := []string{}
	for index, deviceID := range deviceIDs {
		capability, ok, warning := readDeviceCapability(ctx, endpoint, houseID, deviceID, authorization, clientID)
		apiCalls++
		if !ok {
			warnings = appendWarning(warnings, warning)
			return 0, apiCalls, warnings, ""
		}
		scores := groupComponentScores(capability.Device, groupCapability)
		if len(scores) == 0 {
			return 0, apiCalls, warnings, "group_capability_not_supported_by_selected_devices"
		}
		if index == 0 {
			common = scores
			continue
		}
		for componentID, score := range common {
			if nextScore, ok := scores[componentID]; ok {
				common[componentID] = score + nextScore
			} else {
				delete(common, componentID)
			}
		}
		if len(common) == 0 {
			return 0, apiCalls, warnings, "group_capability_not_shared_by_selected_devices"
		}
	}
	componentID, ok := bestGroupComponent(common)
	if !ok {
		return 0, apiCalls, warnings, "ambiguous_group_capability_component"
	}
	return componentID, apiCalls, warnings, ""
}

func groupComponentScores(device api.DeviceCapability, groupCapability string) map[int]int {
	result := map[int]int{}
	for _, component := range device.Components {
		componentID, ok := parseComponentID(component.ID)
		if !ok {
			continue
		}
		score := groupComponentScore(component, groupCapability)
		if score > 0 {
			result[componentID] = score
		}
	}
	return result
}

func groupComponentScore(component api.ComponentCapability, groupCapability string) int {
	normalized := strings.ToLower(strings.TrimSpace(groupCapability))
	props := map[string]bool{}
	for _, property := range component.Properties {
		props[property.ID] = true
	}
	score := 0
	if strings.EqualFold(component.Category, "light") {
		score += 4
	}
	if props[semantic.FieldPower] {
		score += 3
	}
	if props[semantic.FieldBrightness] {
		score += 2
	}
	switch normalized {
	case "color", "rgb", "彩光", "颜色":
		if props[semantic.FieldColor] {
			score += 8
		} else {
			return 0
		}
	case "colortemperature", "color_temperature", "color-temperature", "ct", "色温":
		if props[semantic.FieldColorTemperature] {
			score += 8
		} else {
			return 0
		}
	case "dimming", "brightness", "调光", "亮度":
		if props[semantic.FieldBrightness] {
			score += 8
		} else {
			return 0
		}
	default:
		if !props[semantic.FieldPower] && !props[semantic.FieldBrightness] && !props[semantic.FieldColorTemperature] && !props[semantic.FieldColor] {
			return 0
		}
	}
	return score
}

func parseComponentID(value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func bestGroupComponent(scores map[int]int) (int, bool) {
	if len(scores) == 0 {
		return 0, false
	}
	ids := make([]int, 0, len(scores))
	for id := range scores {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i int, j int) bool {
		if scores[ids[i]] != scores[ids[j]] {
			return scores[ids[i]] > scores[ids[j]]
		}
		return ids[i] < ids[j]
	})
	if len(ids) > 1 && scores[ids[0]] == scores[ids[1]] {
		return 0, false
	}
	return ids[0], true
}
