package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
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
		ExecutionModel:   "direct_after_runtime_validation; use --dry-run/--preview-only for no-write preview",
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
		if nextStep := requestString(guide[semantic.FieldNextStep]); nextStep != "" {
			explanation.NextStep = nextStep
		}
	}
	return explanation
}

func intentRequestSchema(intent string, guide map[string]any) map[string]any {
	parameters := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			semantic.FieldHouseID: map[string]any{
				"type":        "string",
				"description": "House id. Required for house-scoped intents unless supplied by selected profile, --house-id, or homeRef.",
			},
		},
	}
	if guide != nil {
		if shape, ok := guide[semantic.FieldPayloadShape].(map[string]any); ok {
			parameters = payloadShapeToSchema(shape)
		}
	}
	parameters = ensureHouseIDParameter(parameters)
	parameters = ensureIntentTargetParameters(intent, parameters)
	parameters = ensureIntentParameterRequiredFields(intent, parameters)
	schema := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"title":                "Yeelight Home SkillRequest for " + intent,
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{semantic.FieldContractVersion, semantic.FieldRequestID, semantic.FieldLocale, semantic.FieldUtterance, semantic.FieldIntent},
		"properties": map[string]any{
			semantic.FieldContractVersion: map[string]any{"const": contract.Version},
			semantic.FieldRequestID:       map[string]any{"type": "string", "minLength": 1},
			semantic.FieldLocale:          map[string]any{"type": "string", "minLength": 1},
			semantic.FieldUtterance:       map[string]any{"type": "string", "minLength": 1},
			semantic.FieldIntent:          map[string]any{"const": intent},
			semantic.FieldHomeRef: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					semantic.FieldID:         map[string]any{"type": "string"},
					semantic.FieldHouseID:    map[string]any{"type": "string"},
					semantic.FieldName:       map[string]any{"type": "string"},
					semantic.FieldUseCurrent: map[string]any{"type": "boolean"},
				},
			},
			semantic.FieldTargets: map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						semantic.FieldID:         map[string]any{"type": "string"},
						semantic.FieldName:       map[string]any{"type": "string"},
						semantic.FieldEntityType: map[string]any{"type": "string"},
					},
				},
			},
			semantic.FieldParameters: parameters,
			semantic.FieldOptions: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					semantic.FieldDryRun:      map[string]any{"type": "boolean"},
					semantic.FieldPreviewOnly: map[string]any{"type": "boolean"},
				},
			},
		},
	}
	if guide != nil {
		if examples, ok := guide[semantic.FieldExamples]; ok {
			schema[semantic.FieldExamples] = examples
		}
		if nextStep := requestString(guide[semantic.FieldNextStep]); nextStep != "" {
			schema[semantic.FieldNextStep] = nextStep
		}
	}
	return schema
}

func ensureIntentTargetParameters(intent string, schema map[string]any) map[string]any {
	fields := intentTargetParameterFields(intent)
	if len(fields) == 0 {
		return schema
	}
	properties, _ := schema["properties"].(map[string]any)
	if properties == nil {
		properties = map[string]any{}
		schema["properties"] = properties
	}
	for _, field := range fields {
		if _, ok := properties[field]; ok {
			continue
		}
		properties[field] = map[string]any{
			"type":        "string",
			"description": "Natural target field accepted by Runtime target resolution.",
		}
	}
	return schema
}

func intentTargetParameterFields(intent string) []string {
	switch intent {
	case "panel.get", "panel.button.type.get", "panel.button.configure", "panel.button_event.update", "panel.button_event.batch_update", "panel.button_event.reset":
		return []string{
			semantic.FieldPanelID,
			semantic.FieldDeviceID,
			semantic.FieldEntityID,
			semantic.FieldID,
			semantic.FieldPanelName,
			semantic.FieldDeviceName,
			semantic.FieldEntityName,
			semantic.FieldName,
			semantic.FieldRoomID,
			semantic.FieldRoomName,
			semantic.FieldTargetRoomName,
		}
	case "knob.get", "knob.configure", "knob.reset":
		return []string{
			semantic.FieldKnobID,
			semantic.FieldDeviceID,
			semantic.FieldEntityID,
			semantic.FieldID,
			semantic.FieldKnobName,
			semantic.FieldDeviceName,
			semantic.FieldEntityName,
			semantic.FieldName,
			semantic.FieldRoomID,
			semantic.FieldRoomName,
			semantic.FieldTargetRoomName,
		}
	}
	switch entityTypeFromIntent(intent) {
	case "device":
		return []string{
			semantic.FieldDeviceID,
			semantic.FieldEntityID,
			semantic.FieldID,
			semantic.FieldDeviceName,
			semantic.FieldEntityName,
			semantic.FieldName,
			semantic.FieldRoomID,
			semantic.FieldRoomName,
			semantic.FieldTargetRoomName,
		}
	case "gateway":
		return []string{
			semantic.FieldGatewayID,
			semantic.FieldDeviceID,
			semantic.FieldEntityID,
			semantic.FieldID,
			semantic.FieldGatewayName,
			semantic.FieldDeviceName,
			semantic.FieldEntityName,
			semantic.FieldName,
			semantic.FieldRoomID,
			semantic.FieldRoomName,
			semantic.FieldTargetRoomName,
		}
	case "room":
		return []string{
			semantic.FieldRoomID,
			semantic.FieldEntityID,
			semantic.FieldID,
			semantic.FieldRoomName,
			semantic.FieldTargetRoomName,
			semantic.FieldEntityName,
			semantic.FieldName,
		}
	case "area":
		return []string{
			semantic.FieldAreaID,
			semantic.FieldEntityID,
			semantic.FieldID,
			semantic.FieldAreaName,
			semantic.FieldEntityName,
			semantic.FieldName,
		}
	case "group":
		return []string{
			semantic.FieldGroupID,
			semantic.FieldEntityID,
			semantic.FieldID,
			semantic.FieldGroupName,
			semantic.FieldEntityName,
			semantic.FieldName,
			semantic.FieldRoomID,
			semantic.FieldRoomName,
			semantic.FieldTargetRoomName,
		}
	case "scene":
		return []string{
			semantic.FieldSceneID,
			semantic.FieldEntityID,
			semantic.FieldID,
			semantic.FieldSceneName,
			semantic.FieldCurrentName,
			semantic.FieldEntityName,
			semantic.FieldTargetName,
			semantic.FieldName,
		}
	case "automation":
		return []string{
			semantic.FieldAutomationID,
			semantic.FieldEntityID,
			semantic.FieldID,
			semantic.FieldAutomationName,
			semantic.FieldCurrentName,
			semantic.FieldEntityName,
			semantic.FieldTargetName,
			semantic.FieldName,
		}
	default:
		return nil
	}
}

func ensureHouseIDParameter(schema map[string]any) map[string]any {
	properties, _ := schema["properties"].(map[string]any)
	if properties == nil {
		properties = map[string]any{}
		schema["properties"] = properties
	}
	if _, ok := properties[semantic.FieldHouseID]; !ok {
		properties[semantic.FieldHouseID] = map[string]any{
			"type":        "string",
			"description": "House id. Required for house-scoped intents unless supplied by selected profile, --house-id, or homeRef.",
		}
	}
	return schema
}

func ensureIntentParameterRequiredFields(intent string, schema map[string]any) map[string]any {
	requiredFields := intentAdditionalRequiredParameterFields(intent)
	if len(requiredFields) == 0 {
		return schema
	}
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return schema
	}
	required := schemaStringList(schema["required"])
	for _, field := range requiredFields {
		if _, ok := properties[field]; ok && !schemaStringListContains(required, field) {
			required = append(required, field)
		}
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func intentAdditionalRequiredParameterFields(intent string) []string {
	switch intent {
	case "scene.create", "scene.update":
		return []string{semantic.FieldActions}
	case "automation.create", "automation.update":
		return []string{semantic.FieldTrigger, semantic.FieldActions}
	default:
		return nil
	}
}

func schemaStringList(value any) []string {
	items := []string{}
	switch typed := value.(type) {
	case []string:
		items = append(items, typed...)
	case []any:
		for _, raw := range typed {
			text := strings.TrimSpace(requestString(raw))
			if text != "" {
				items = append(items, text)
			}
		}
	}
	return items
}

func schemaStringListContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func payloadShapeToSchema(shape map[string]any) map[string]any {
	properties := map[string]any{}
	required := []string{}
	for _, key := range sortedMapKeys(shape) {
		if !payloadShapeAcceptedKey(key) {
			continue
		}
		properties[key] = payloadShapeValueToSchema(key, shape[key])
		if payloadShapeValueIsRequired(shape[key]) {
			required = append(required, key)
		}
	}
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func payloadShapeValueToSchema(key string, value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return payloadShapeToSchema(typed)
	case []any:
		itemSchema := map[string]any{"type": "object", "additionalProperties": false}
		if len(typed) > 0 {
			itemSchema = payloadShapeValueToSchema("", typed[0])
		}
		return map[string]any{"type": "array", "items": itemSchema}
	case string:
		return map[string]any{
			"type":        inferSchemaTypeForKey(key, typed),
			"description": typed,
		}
	default:
		return map[string]any{"description": fmt.Sprintf("%v", typed)}
	}
}

func payloadShapeValueIsRequired(value any) bool {
	description, ok := value.(string)
	if !ok {
		return false
	}
	lower := strings.TrimSpace(strings.ToLower(description))
	return lower == "required" || strings.HasPrefix(lower, "required ") || strings.HasPrefix(lower, "required:")
}

func inferSchemaType(description string) any {
	lower := strings.ToLower(description)
	isNumeric := hasSchemaWord(lower, "integer") || hasSchemaWord(lower, "number") || strings.Contains(lower, "1..") || strings.Contains(lower, "0..")
	isObject := hasSchemaWord(lower, "object") || hasSchemaWord(lower, "map")
	switch {
	case hasSchemaWord(lower, "list") || hasSchemaWord(lower, "array"):
		return "array"
	case hasSchemaWord(lower, "boolean") || hasSchemaWord(lower, "bool"):
		return "boolean"
	case isNumeric && isObject:
		return []string{"integer", "number", "string", "object"}
	case isNumeric:
		return []string{"integer", "number", "string"}
	case isObject:
		return "object"
	default:
		return "string"
	}
}

func inferSchemaTypeForKey(key string, description string) any {
	normalizedKey := strings.ToLower(strings.TrimSpace(key))
	if strings.HasSuffix(normalizedKey, "ids") || strings.HasSuffix(normalizedKey, "names") {
		return "array"
	}
	if normalizedKey == "id" || (strings.HasSuffix(normalizedKey, "id") && !strings.HasSuffix(normalizedKey, "ids")) {
		return "string"
	}
	if strings.HasSuffix(normalizedKey, "name") {
		return "string"
	}
	return inferSchemaType(description)
}

func hasSchemaWord(value string, word string) bool {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r < 'a' || r > 'z'
	})
	for _, field := range fields {
		if field == word {
			return true
		}
	}
	return false
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
	fields := []string{
		semantic.ParameterPath(semantic.FieldHouseID),
		semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldName),
		semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldUseCurrent),
		semantic.FieldPath(semantic.ArrayField(semantic.FieldTargets), semantic.FieldID),
		semantic.FieldPath(semantic.ArrayField(semantic.FieldTargets), semantic.FieldName),
		semantic.FieldPath(semantic.ArrayField(semantic.FieldTargets), semantic.FieldEntityType),
	}
	guide := payloadGuideForIntent(intent)
	if guide != nil {
		if shape, ok := guide[semantic.FieldPayloadShape].(map[string]any); ok {
			fields = append(fields, payloadShapeAcceptedFields(semantic.FieldParameters, shape)...)
		}
	}
	for _, field := range intentTargetParameterFields(intent) {
		fields = append(fields, semantic.ParameterPath(field))
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
		if parameters := requestMap(properties[semantic.FieldParameters]); parameters != nil {
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
	intent := strings.TrimSpace(requestString(request.Parameters[semantic.FieldIntent]))
	if intent == "" {
		intent = strings.TrimSpace(requestString(request.Parameters[semantic.FieldTargetIntent]))
	}
	if intent == "" {
		return configureClarificationResponseWithGuide(request, "missing_intent_to_explain", []string{semantic.ParameterPath(semantic.FieldIntent), semantic.ParameterPath(semantic.FieldTargetIntent)}, map[string]any{
			semantic.FieldPayloadShape: map[string]any{
				semantic.FieldIntent:       "required Runtime intent to explain, such as scene.update",
				semantic.FieldTargetIntent: "optional alias of intent",
			},
			semantic.FieldExamples: []any{map[string]any{semantic.FieldIntent: "scene.update"}},
			semantic.FieldNextStep: "Send intent.explain with parameters.intent set to the Runtime intent whose payload contract you need.",
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
			semantic.FieldIntentExplanation: explanation,
		},
		TraceID: "intent-explain-local",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 1,
		},
	}
}
