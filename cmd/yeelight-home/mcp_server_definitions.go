package main

func localMCPToolDefinitions() []any {
	return []any{
		mcpToolDefinition(
			"yeelight_home_get_home",
			"Get Yeelight home overview",
			"Read the selected Yeelight home's rooms, devices, groups, scenes, and summary without changing anything.",
			mcpScopeInputSchema(),
			mcpReadOnlyAnnotations(),
		),
		mcpToolDefinition(
			"yeelight_home_list_entities",
			"List Yeelight home entities",
			"List rooms, areas, devices, groups, scenes, and automations in the selected Yeelight home.",
			mcpScopeInputSchema(),
			mcpReadOnlyAnnotations(),
		),
		mcpToolDefinition(
			"yeelight_home_get_state",
			"Get Yeelight device state",
			"Read the current state of one Yeelight device. Use roomName when names are duplicated.",
			mcpSchemaWithAnyOf(mcpObjectSchema(map[string]any{
				"locale":     mcpLocaleSchema(),
				"houseId":    mcpStringSchema("Optional home id; the selected home is used when omitted."),
				"deviceId":   mcpStringSchema("Exact device id when already known."),
				"deviceName": mcpStringSchema("Natural device name when the id is not known."),
				"roomName":   mcpStringSchema("Room qualifier for duplicate device names."),
			}, nil), "deviceId", "deviceName"),
			mcpReadOnlyAnnotations(),
		),
		mcpToolDefinition(
			"yeelight_home_control_light",
			"Control Yeelight lighting",
			"Turn a Yeelight light target on or off, or set brightness, color temperature, or color. The target may be a home, room, area, group, or device.",
			mcpSchemaWithAnyOf(mcpObjectSchema(map[string]any{
				"locale":     mcpLocaleSchema(),
				"houseId":    mcpStringSchema("Optional home id; the selected home is used when omitted."),
				"action":     map[string]any{"type": "string", "enum": []string{"power", "brightness", "color_temperature", "color"}, "description": "Lighting operation to perform."},
				"value":      map[string]any{"description": "power: boolean; brightness: 1-100; color_temperature: 2700-6500; color: RGB integer, hex string, or red/green/blue object."},
				"targetType": map[string]any{"type": "string", "enum": []string{"home", "room", "area", "group", "device"}, "description": "Optional target scope type."},
				"targetId":   mcpStringSchema("Exact target id when already known."),
				"targetName": mcpStringSchema("Natural target name when the id is not known."),
				"roomName":   mcpStringSchema("Room qualifier for duplicate device names."),
			}, []string{"action", "value"}), "targetId", "targetName"),
			map[string]any{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": true, "openWorldHint": true},
		),
		mcpToolDefinition(
			"yeelight_home_run_scene",
			"Run Yeelight scene",
			"Run one existing Yeelight scene by id or natural name in the selected home.",
			mcpSchemaWithAnyOf(mcpObjectSchema(map[string]any{
				"locale":    mcpLocaleSchema(),
				"houseId":   mcpStringSchema("Optional home id; the selected home is used when omitted."),
				"sceneId":   mcpStringSchema("Exact scene id when already known."),
				"sceneName": mcpStringSchema("Natural scene name when the id is not known."),
			}, nil), "sceneId", "sceneName"),
			map[string]any{"readOnlyHint": false, "destructiveHint": false, "idempotentHint": false, "openWorldHint": true},
		),
		mcpToolDefinition(
			"yeelight_home_explain",
			"Explain Yeelight capability",
			"Return the accepted fields, payload shape, examples, and next step for one Yeelight Home Runtime intent.",
			mcpObjectSchema(map[string]any{
				"locale": mcpLocaleSchema(),
				"intent": mcpStringSchema("Runtime intent to explain, such as scene.update or automation.create."),
			}, []string{"intent"}),
			mcpReadOnlyAnnotations(),
		),
		mcpToolDefinition(
			"yeelight_home_invoke",
			"Use advanced Yeelight capability",
			"Invoke any supported Yeelight Home Runtime intent. Prefer the focused tools for common home, state, light, and scene operations.",
			mcpObjectSchema(map[string]any{
				"locale":     mcpLocaleSchema(),
				"utterance":  mcpStringSchema("Short natural-language description of the user's request."),
				"intent":     mcpStringSchema("Supported Yeelight Home Runtime intent."),
				"homeRef":    map[string]any{"type": "object", "description": "Optional home reference by id, houseId, name, or useCurrent."},
				"targets":    map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "Optional structured Runtime targets."},
				"parameters": map[string]any{"type": "object", "description": "Intent-specific parameters. Call yeelight_home_explain first for complex writes."},
				"options":    map[string]any{"type": "object", "description": "Optional dryRun or previewOnly execution options."},
			}, []string{"intent"}),
			map[string]any{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": false, "openWorldHint": true},
		),
	}
}

func mcpToolDefinition(name, title, description string, inputSchema, annotations map[string]any) map[string]any {
	return map[string]any{
		"name": name, "title": title, "description": description,
		"inputSchema": inputSchema, "outputSchema": mcpRuntimeResponseSchema(),
		"annotations": annotations,
	}
}

func mcpScopeInputSchema() map[string]any {
	return mcpObjectSchema(map[string]any{
		"locale":  mcpLocaleSchema(),
		"houseId": mcpStringSchema("Optional home id; the selected home is used when omitted."),
	}, nil)
}

func mcpObjectSchema(properties map[string]any, required []string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func mcpSchemaWithAnyOf(schema map[string]any, names ...string) map[string]any {
	choices := make([]any, 0, len(names))
	for _, name := range names {
		choices = append(choices, map[string]any{"required": []string{name}})
	}
	schema["anyOf"] = choices
	return schema
}

func mcpStringSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func mcpLocaleSchema() map[string]any {
	return map[string]any{"type": "string", "enum": []string{"zh-CN", "en-US"}, "description": "Language for user-facing messages."}
}

func mcpRuntimeResponseSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"contractVersion": map[string]any{"type": "string"},
			"requestId":       map[string]any{"type": "string"},
			"status":          map[string]any{"type": "string"},
			"userMessage":     map[string]any{"type": "string"},
			"result":          map[string]any{"type": "object"},
			"clarification":   map[string]any{"type": "object"},
			"warnings":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"error":           map[string]any{"type": "object"},
		},
		"required": []string{"contractVersion", "requestId", "status", "userMessage", "warnings"},
	}
}

func mcpReadOnlyAnnotations() map[string]any {
	return map[string]any{"readOnlyHint": true, "destructiveHint": false, "idempotentHint": true, "openWorldHint": true}
}
