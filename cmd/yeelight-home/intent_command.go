package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/yeelight/yeelight-home/internal/contract"
)

func runIntent(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home intent <explain|schema> --intent <intent> [--json]")
		return exitInvalidInput
	}
	action := args[0]
	switch action {
	case "explain":
		return runIntentExplain(args[1:], stdout, stderr)
	case "schema":
		return runIntentSchema(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported intent command %q\n", action)
		return exitInvalidInput
	}
}

func runExplainAlias(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home explain <intent> [--json]")
		return exitInvalidInput
	}
	intent := strings.TrimSpace(args[0])
	if strings.HasPrefix(intent, "--") || intent == "" {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home explain <intent> [--json]")
		return exitInvalidInput
	}
	forwarded := append([]string{"--intent", intent}, args[1:]...)
	return runIntentSchema(forwarded, stdout, stderr)
}

func runIntentExplain(args []string, stdout io.Writer, stderr io.Writer) int {
	return runIntentExplainLike(args, stdout, stderr, false)
}

func runIntentSchema(args []string, stdout io.Writer, stderr io.Writer) int {
	return runIntentExplainLike(args, stdout, stderr, true)
}

func runIntentExplainLike(args []string, stdout io.Writer, stderr io.Writer, schemaOnly bool) int {
	flags, err := parseFlags(args)
	if err != nil || !intentExplainFlagsAllowed(flags) {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home intent <explain|schema> --intent <intent> [--json]")
		return exitInvalidInput
	}
	intent := flags.string("intent", "")
	if intent == "" {
		_, _ = fmt.Fprintln(stderr, "intent explain/schema: --intent is required")
		return exitInvalidInput
	}
	explanation := explainIntent(intent)
	if !explanation.Supported {
		_, _ = fmt.Fprintf(stderr, "unsupported intent %q\n", intent)
		return exitInvalidInput
	}
	if schemaOnly {
		schema := explanation.RequestSchema
		if schema == nil {
			schema = intentRequestSchema(intent, explanation.PayloadGuide)
		}
		if flags.bool("json") {
			return writeJSON(stdout, stderr, schema)
		}
		return writeIntentSchema(stdout, explanation, schema)
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, explanation)
	}
	return writeIntentExplanation(stdout, explanation)
}

func intentExplainFlagsAllowed(flags cliFlags) bool {
	for name := range flags.values {
		switch name {
		case "intent", "json":
		default:
			return false
		}
	}
	return true
}

type intentExplanation struct {
	Intent           string         `json:"intent"`
	Supported        bool           `json:"supported"`
	Implemented      bool           `json:"implemented"`
	LocalOnly        bool           `json:"localOnly"`
	HouseIndependent bool           `json:"houseIndependent"`
	ExecutionModel   string         `json:"executionModel"`
	Resource         string         `json:"resource,omitempty"`
	Action           string         `json:"action,omitempty"`
	Utterance        string         `json:"utterance,omitempty"`
	TargetEntityType string         `json:"targetEntityType,omitempty"`
	TargetIDFlags    []string       `json:"targetIdFlags,omitempty"`
	TargetName       bool           `json:"targetName,omitempty"`
	AcceptedFields   []string       `json:"acceptedFields,omitempty"`
	RequestSchema    map[string]any `json:"requestSchema,omitempty"`
	PayloadGuide     map[string]any `json:"payloadGuide,omitempty"`
	ExampleCommand   string         `json:"exampleCommand,omitempty"`
	NextStep         string         `json:"nextStep,omitempty"`
}

func explainIntent(intent string) intentExplanation {
	resource, action, spec, found := moduleSpecForIntent(intent)
	guide := payloadGuideForIntent(intent)
	explanation := intentExplanation{
		Intent:           intent,
		Supported:        found || isImplementedInvokeIntent(intent),
		Implemented:      isImplementedInvokeIntent(intent),
		LocalOnly:        isLocalOnlyInvokeIntent(intent),
		HouseIndependent: isHouseIndependentInvokeIntent(intent),
		ExecutionModel:   "direct_after_adapter_validation; use --dry-run/--preview-only for no-write preview",
		AcceptedFields:   intentAcceptedFields(intent),
		PayloadGuide:     guide,
	}
	explanation.RequestSchema = intentRequestSchema(intent, guide)
	if found {
		explanation.Resource = resource
		explanation.Action = action
		explanation.Utterance = spec.Utterance
		explanation.TargetEntityType = spec.EntityType
		explanation.TargetIDFlags = append([]string{}, spec.TargetIDKeys...)
		explanation.TargetName = spec.TargetName
		explanation.ExampleCommand = strings.TrimSpace(moduleActionExamples(resource, action, spec))
	}
	if guide != nil {
		if nextStep := requestString(guide["nextStep"]); nextStep != "" {
			explanation.NextStep = nextStep
		}
	}
	return explanation
}

func intentRequestSchema(intent string, guide map[string]any) map[string]any {
	parameters := map[string]any{
		"type":                 "object",
		"additionalProperties": true,
		"properties": map[string]any{
			"houseId": map[string]any{
				"type":        "string",
				"description": "House id. Required for house-scoped intents unless supplied by selected profile, --house-id, or homeRef.",
			},
		},
	}
	if guide != nil {
		if shape, ok := guide["payloadShape"].(map[string]any); ok {
			parameters = payloadShapeToSchema(shape)
		}
	}
	schema := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"title":                "Yeelight Home SkillRequest for " + intent,
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"contractVersion", "requestId", "locale", "utterance", "intent"},
		"properties": map[string]any{
			"contractVersion": map[string]any{"const": contract.Version},
			"requestId":       map[string]any{"type": "string", "minLength": 1},
			"locale":          map[string]any{"type": "string", "minLength": 1},
			"utterance":       map[string]any{"type": "string", "minLength": 1},
			"intent":          map[string]any{"const": intent},
			"homeRef": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
				"properties": map[string]any{
					"name":       map[string]any{"type": "string"},
					"useCurrent": map[string]any{"type": "boolean"},
				},
			},
			"targets": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
					"properties": map[string]any{
						"id":         map[string]any{"type": "string"},
						"name":       map[string]any{"type": "string"},
						"entityType": map[string]any{"type": "string"},
					},
				},
			},
			"parameters": parameters,
			"options": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
				"properties": map[string]any{
					"dryRun":      map[string]any{"type": "boolean"},
					"previewOnly": map[string]any{"type": "boolean"},
				},
			},
		},
	}
	if guide != nil {
		if examples, ok := guide["examples"]; ok {
			schema["examples"] = examples
		}
		if nextStep := requestString(guide["nextStep"]); nextStep != "" {
			schema["nextStep"] = nextStep
		}
	}
	return schema
}

func payloadShapeToSchema(shape map[string]any) map[string]any {
	properties := map[string]any{}
	required := []string{}
	for _, key := range sortedMapKeys(shape) {
		if !payloadShapeAcceptedKey(key) {
			continue
		}
		properties[key] = payloadShapeValueToSchema(shape[key])
		if payloadShapeDescriptionContains(shape[key], "required") {
			required = append(required, key)
		}
	}
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": true,
		"properties":           properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func payloadShapeValueToSchema(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return payloadShapeToSchema(typed)
	case []any:
		itemSchema := map[string]any{"type": "object", "additionalProperties": true}
		if len(typed) > 0 {
			itemSchema = payloadShapeValueToSchema(typed[0])
		}
		return map[string]any{"type": "array", "items": itemSchema}
	case string:
		return map[string]any{
			"type":        inferSchemaType(typed),
			"description": typed,
		}
	default:
		return map[string]any{"description": fmt.Sprintf("%v", typed)}
	}
}

func payloadShapeDescriptionContains(value any, marker string) bool {
	switch typed := value.(type) {
	case string:
		return strings.Contains(strings.ToLower(typed), marker)
	case map[string]any:
		for _, child := range typed {
			if payloadShapeDescriptionContains(child, marker) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if payloadShapeDescriptionContains(child, marker) {
				return true
			}
		}
	}
	return false
}

func inferSchemaType(description string) any {
	lower := strings.ToLower(description)
	switch {
	case strings.Contains(lower, "list") || strings.Contains(lower, "array"):
		return "array"
	case strings.Contains(lower, "boolean") || strings.Contains(lower, "bool"):
		return "boolean"
	case strings.Contains(lower, "integer") || strings.Contains(lower, "number") || strings.Contains(lower, "1..") || strings.Contains(lower, "0.."):
		return []string{"integer", "number", "string"}
	case strings.Contains(lower, "object") || strings.Contains(lower, "map"):
		return "object"
	default:
		return "string"
	}
}

func moduleSpecForIntent(intent string) (string, string, moduleCommandSpec, bool) {
	resources := moduleResourceNames()
	for _, resource := range resources {
		actions := moduleCommandNames(resource)
		for _, action := range actions {
			spec := moduleCommands[resource][action]
			if spec.Intent == intent {
				return resource, action, spec, true
			}
		}
	}
	return "", "", moduleCommandSpec{}, false
}

func intentAcceptedFields(intent string) []string {
	fields := []string{"parameters.houseId", "homeRef.name", "homeRef.useCurrent", "targets[].id", "targets[].name", "targets[].entityType"}
	guide := payloadGuideForIntent(intent)
	if guide != nil {
		if shape, ok := guide["payloadShape"].(map[string]any); ok {
			fields = append(fields, payloadShapeAcceptedFields("parameters", shape)...)
		}
	}
	fields = uniqueStrings(fields)
	sort.Strings(fields)
	return fields
}

func payloadShapeAcceptedFields(prefix string, value any) []string {
	result := []string{}
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range sortedMapKeys(typed) {
			if !payloadShapeAcceptedKey(key) {
				continue
			}
			path := prefix + "." + key
			result = append(result, path)
			result = append(result, payloadShapeAcceptedFields(path, typed[key])...)
		}
	case []any:
		arrayPrefix := prefix + "[]"
		result = append(result, arrayPrefix)
		if len(typed) > 0 {
			result = append(result, payloadShapeAcceptedFields(arrayPrefix, typed[0])...)
		}
	}
	return result
}

func payloadShapeAcceptedKey(key string) bool {
	switch key {
	case "sceneActionContract", "automationContract", "shortKeyCompatibility", "keyVocabulary":
		return false
	default:
		return true
	}
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func writeIntentExplanation(stdout io.Writer, explanation intentExplanation) int {
	lines := []string{
		fmt.Sprintf("Intent: %s", explanation.Intent),
		fmt.Sprintf("Implemented: %t", explanation.Implemented),
		fmt.Sprintf("Execution model: %s", explanation.ExecutionModel),
	}
	if explanation.Resource != "" {
		lines = append(lines, fmt.Sprintf("Shortcut: yeelight-home %s %s", explanation.Resource, explanation.Action))
	}
	if explanation.NextStep != "" {
		lines = append(lines, "Next step: "+explanation.NextStep)
	}
	if explanation.ExampleCommand != "" {
		lines = append(lines, "Example: "+explanation.ExampleCommand)
	}
	if len(explanation.AcceptedFields) > 0 {
		lines = append(lines, "Accepted fields: "+strings.Join(explanation.AcceptedFields, ", "))
	}
	if explanation.RequestSchema != nil {
		lines = append(lines, "Machine schema: yeelight-home intent schema --intent "+explanation.Intent+" --json")
	}
	_, _ = fmt.Fprintln(stdout, strings.Join(lines, "\n"))
	return exitOK
}

func writeIntentSchema(stdout io.Writer, explanation intentExplanation, schema map[string]any) int {
	lines := []string{
		fmt.Sprintf("Intent schema: %s", explanation.Intent),
		"JSON: yeelight-home intent schema --intent " + explanation.Intent + " --json",
	}
	if explanation.NextStep != "" {
		lines = append(lines, "Next step: "+explanation.NextStep)
	}
	if properties := requestMap(schema["properties"]); properties != nil {
		if parameters := requestMap(properties["parameters"]); parameters != nil {
			if parameterProperties := requestMap(parameters["properties"]); parameterProperties != nil {
				keys := sortedMapKeys(parameterProperties)
				lines = append(lines, "Parameter keys: "+strings.Join(keys, ", "))
			}
		}
	}
	_, _ = fmt.Fprintln(stdout, strings.Join(lines, "\n"))
	return exitOK
}

func invokeIntentExplain(request contract.Request) contract.Response {
	intent := strings.TrimSpace(requestString(request.Parameters["intent"]))
	if intent == "" {
		intent = strings.TrimSpace(requestString(request.Parameters["targetIntent"]))
	}
	if intent == "" {
		return configureClarificationResponseWithGuide(request, "missing_intent_to_explain", []string{"parameters.intent", "parameters.targetIntent"}, map[string]any{
			"payloadShape": map[string]any{
				"intent":       "required semantic Runtime intent to explain, such as scene.update",
				"targetIntent": "optional alias of intent",
			},
			"examples": []any{map[string]any{"intent": "scene.update"}},
			"nextStep": "Send intent.explain with parameters.intent set to the semantic Runtime intent whose payload contract you need.",
		})
	}
	explanation := explainIntent(intent)
	if !explanation.Supported {
		return executionBlockedResponse(request, "unsupported_intent_to_explain", fmt.Sprintf("Runtime does not expose semantic intent %s.", intent))
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已读取本地 Runtime intent 契约说明。",
		Result: map[string]any{
			"intentExplanation": explanation,
		},
		TraceID: "intent-explain-local",
		Metrics: map[string]any{
			"apiCalls":  0,
			"cacheHits": 1,
		},
	}
}
