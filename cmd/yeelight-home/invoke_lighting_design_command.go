package main

import (
	"context"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

type runtimeLightingCatalog struct {
	Version            string `json:"version"`
	Status             string `json:"status"`
	LightingExperience struct {
		SceneRecipes []map[string]any `json:"sceneRecipes"`
		MoodRecipes  []map[string]any `json:"moodRecipes"`
	} `json:"lightingExperience"`
}

func newRuntimeLightingCatalog() runtimeLightingCatalog {
	catalog := runtimeLightingCatalog{
		Version: "0.1.0",
		Status:  "runtime_builtin",
	}
	catalog.LightingExperience.SceneRecipes = []map[string]any{
		{"name": "回家模式", "brightness": 90, "colorTemperature": 4500},
		{"name": "离家模式", "mainLight": "off"},
		{"name": "清洁模式", "brightness": 100, "colorTemperature": 5700},
		{"name": "日常模式", "brightnessMax": 80, "colorTemperature": 3800},
		{"name": "会客模式", "brightnessMax": 100, "colorTemperature": 4500},
		{"name": "观影模式", "mainLight": "off_or_low", "backgroundBrightnessMax": 20, "colorTemperature": 3000},
		{"name": "阅读模式", "brightness": 80, "colorTemperature": 4500},
		{"name": "睡眠模式", "brightness": 8, "colorTemperature": 2700},
		{"name": "夜灯模式", "brightness": 8, "colorTemperature": 2700},
	}
	catalog.LightingExperience.MoodRecipes = []map[string]any{
		{"expression": "放松", "brightness": 35, "colorTemperature": 3000},
		{"expression": "睡前", "brightness": 8, "colorTemperature": 2700},
		{"expression": "专注", "brightness": 80, "colorTemperature": 5000},
		{"expression": "阅读", "brightness": 80, "colorTemperature": 4500},
		{"expression": "观影", "brightness": 20, "mainLight": "off_or_low", "colorTemperature": 3000},
		{"expression": "聚餐", "brightness": 65, "colorTemperature": 3500},
	}
	return catalog
}

func (app *app) invokeLightingDesignPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
		HouseID: houseID,
		Credentials: api.EntityListCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	scope, selectedDevices, clarification := lightingDesignScope(request, entities)
	if clarification != nil {
		return *clarification, nil
	}
	domainCatalog := loadRuntimeLightingCatalog()
	recipe := selectLightingRecipe(request, domainCatalog)
	capabilityEvidence, capabilityWarnings, capabilityCalls := lightingDesignCapabilities(ctx, endpoint, entities.HouseID, selectedDevices, authorization, clientID)

	unknowns := []string{}
	warnings := append([]string{}, entities.Warnings...)
	warnings = append(warnings, capabilityWarnings...)
	if len(selectedDevices) == 0 {
		unknowns = append(unknowns, "target_device_evidence_unavailable")
	}
	if len(capabilityWarnings) > 0 {
		unknowns = append(unknowns, "some_device_capabilities_unavailable")
	}

	status := "success"
	message := "已生成本地照明设计计划；不会直接修改设备、情景或自动化。"
	if len(unknowns) > 0 {
		status = "partial"
		message = "已生成本地照明设计计划，但仍缺少部分设备或规则证据。"
	}
	result := map[string]any{
		"region":           entities.Region,
		"houseId":          entities.HouseID,
		"planType":         "local_lighting_design",
		"persistentWrites": false,
		"applyIntent":      "lighting.design.apply",
		"applyBehavior":    "pending_plan_required",
		"scope":            scope,
		"selectedRecipe":   recipe,
		"deviceEvidence":   capabilityEvidence,
		"unknownEvidence":  unknowns,
		"steps": []string{
			"按当前家庭实体和设备能力证据生成建议",
			"只使用可确认支持的灯光属性作为候选",
			"任何持久化应用必须进入 pending plan 确认链路",
		},
	}
	result["runtimeLightingCatalog"] = map[string]any{
		"version":          domainCatalog.Version,
		"status":           domainCatalog.Status,
		"sceneRecipeCount": len(domainCatalog.LightingExperience.SceneRecipes),
		"moodRecipeCount":  len(domainCatalog.LightingExperience.MoodRecipes),
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          status,
		UserMessage:     message,
		Result:          result,
		Warnings:        warnings,
		TraceID:         "lighting-design-plan-local",
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(entities) + capabilityCalls,
			"cacheHits": 0,
		},
	}, nil
}

func lightingDesignScope(request contract.Request, entities api.EntityListResult) (map[string]any, []api.EntitySummary, *contract.Response) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		devices := firstEntitiesByType(entities.Entities, "device", 0)
		return map[string]any{"type": "home", "target": map[string]any{"houseId": entities.HouseID}}, devices, nil
	}
	match, candidates, _ := findEntity(target, entities.Entities)
	if match.ID == "" {
		response := diagnosticClarificationResponse(request, "entity_not_found", target, candidates, []string{"room", "area", "group", "device"}, entityListAPICalls(entities))
		response.TraceID = "lighting-design-clarification"
		return nil, nil, &response
	}
	scope := map[string]any{
		"type":   match.Type,
		"target": entitySummaryMap(match),
	}
	switch match.Type {
	case "device":
		return scope, []api.EntitySummary{match}, nil
	case "room":
		return scope, devicesInRoom(entities.Entities, match.ID, 0), nil
	default:
		scope["limitations"] = []string{"当前 Runtime 只能直接确认房间和设备范围；区域、设备组成员关系仍需后续只读 adapter。"}
		return scope, []api.EntitySummary{}, nil
	}
}

func lightingDesignCapabilities(ctx context.Context, endpoint api.Endpoint, houseID string, devices []api.EntitySummary, authorization string, clientID string) ([]any, []string, int) {
	evidence := []any{}
	warnings := []string{}
	apiCalls := 0
	for _, device := range devices {
		capability, ok, warning := readDeviceCapability(ctx, endpoint, houseID, device.ID, authorization, clientID)
		if !ok {
			warnings = append(warnings, warning)
			continue
		}
		apiCalls++
		evidence = append(evidence, map[string]any{
			"entity":       entitySummaryMap(device),
			"schemaStatus": capability.SchemaStatus,
			"propertyIds":  stateQueryPropertySet(capability.Device),
		})
	}
	return evidence, warnings, apiCalls
}

func loadRuntimeLightingCatalog() runtimeLightingCatalog {
	return newRuntimeLightingCatalog()
}

func selectLightingRecipe(request contract.Request, catalog runtimeLightingCatalog) map[string]any {
	query := strings.ToLower(strings.TrimSpace(request.Utterance + " " + firstRequestString(request.Parameters, "mood", "scene", "purpose", "recipe")))
	for _, recipe := range catalog.LightingExperience.SceneRecipes {
		if recipeMatches(query, recipe, "name") {
			return compactRecipe(recipe, "scene")
		}
	}
	for _, recipe := range catalog.LightingExperience.MoodRecipes {
		if recipeMatches(query, recipe, "expression") {
			return compactRecipe(recipe, "mood")
		}
	}
	for _, recipe := range catalog.LightingExperience.SceneRecipes {
		if recipe["name"] == "日常模式" {
			return compactRecipe(recipe, "scene")
		}
	}
	return map[string]any{"type": "conservative_default", "name": "日常模式"}
}

func recipeMatches(query string, recipe map[string]any, key string) bool {
	value, ok := recipe[key].(string)
	if !ok || value == "" {
		return false
	}
	normalizedQuery := normalizeRecipeMatchText(query)
	normalizedValue := normalizeRecipeMatchText(value)
	return strings.Contains(normalizedQuery, normalizedValue) || strings.Contains(normalizedValue, normalizedQuery)
}

func normalizeRecipeMatchText(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.TrimSuffix(normalized, "模式")
	return normalized
}

func compactRecipe(recipe map[string]any, recipeType string) map[string]any {
	result := map[string]any{"type": recipeType}
	for _, key := range []string{"name", "expression", "brightness", "brightnessMax", "colorTemperature", "mainLight", "backgroundBrightnessMax"} {
		if value, ok := recipe[key]; ok {
			result[key] = value
		}
	}
	return result
}

func firstEntitiesByType(entities []api.EntitySummary, entityType string, limit int) []api.EntitySummary {
	result := []api.EntitySummary{}
	for _, entity := range entities {
		if entity.Type != entityType {
			continue
		}
		result = append(result, entity)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func devicesInRoom(entities []api.EntitySummary, roomID string, limit int) []api.EntitySummary {
	result := []api.EntitySummary{}
	for _, entity := range entities {
		if entity.Type != "device" || entity.RoomID != roomID {
			continue
		}
		result = append(result, entity)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}
