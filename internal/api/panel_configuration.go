package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type PanelConfigurationKind string

const (
	PanelButtonConfigure        PanelConfigurationKind = "panel.button.configure"
	PanelButtonEventUpdate      PanelConfigurationKind = "panel.button_event.update"
	PanelButtonEventBatchUpdate PanelConfigurationKind = "panel.button_event.batch_update"
	PanelButtonEventReset       PanelConfigurationKind = "panel.button_event.reset"
	KnobConfigure               PanelConfigurationKind = "knob.configure"
	KnobReset                   PanelConfigurationKind = "knob.reset"
)

type PanelConfigurationCredentials struct {
	Authorization string
	ClientID      string
}

type PanelConfigurationRequest struct {
	Kind           PanelConfigurationKind
	HouseID        string
	DeviceID       string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    PanelConfigurationCredentials
}

type PanelConfigurationResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId,omitempty"`
	DeviceID   string `json:"deviceId"`
	Capability string `json:"capability"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
	APICalls   int    `json:"apiCalls"`
}

type PanelConfigurationClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewPanelConfigurationClient(endpoint Endpoint, client *http.Client) PanelConfigurationClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return PanelConfigurationClient{endpoint: endpoint, client: client}
}

func (client PanelConfigurationClient) Run(ctx context.Context, request PanelConfigurationRequest) (PanelConfigurationResult, error) {
	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID == "" {
		return PanelConfigurationResult{}, fmt.Errorf("device id is required")
	}
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return PanelConfigurationResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	current, verifyCalls, err := client.readCurrent(ctx, request.Kind, deviceID, credentials)
	apiCalls += verifyCalls
	if err != nil {
		return PanelConfigurationResult{}, err
	}
	writePayload := request.Payload
	if request.Kind == PanelButtonConfigure {
		merged, err := buildPanelButtonConfigureWritePayload(current, deviceID, request.Payload)
		if err != nil {
			return PanelConfigurationResult{}, err
		}
		writePayload = merged
	}
	if err := client.write(ctx, request.Kind, deviceID, writePayload, credentials); err != nil {
		return PanelConfigurationResult{}, err
	}
	apiCalls++
	ok, verifyCalls, err := client.verifyAfterWrite(ctx, request.Kind, deviceID, writePayload, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += verifyCalls
	if err != nil {
		return PanelConfigurationResult{}, err
	}
	if !ok {
		return PanelConfigurationResult{}, fmt.Errorf("%s write verification mismatch", request.Kind)
	}
	return PanelConfigurationResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		DeviceID:   deviceID,
		Capability: string(request.Kind),
		Verified:   true,
		VerifiedBy: string(request.Kind) + "_read_after_write",
		APICalls:   apiCalls,
	}, nil
}

func (client PanelConfigurationClient) readCurrent(ctx context.Context, kind PanelConfigurationKind, deviceID string, credentials requestCredentials) (any, int, error) {
	switch kind {
	case PanelButtonConfigure, PanelButtonEventUpdate, PanelButtonEventBatchUpdate, PanelButtonEventReset:
		data, err := client.readPanel(ctx, deviceID, credentials)
		return data, 2, err
	case KnobConfigure, KnobReset:
		data, err := client.readKnob(ctx, deviceID, credentials)
		return data, 1, err
	default:
		return nil, 0, fmt.Errorf("unsupported panel configuration kind %q", kind)
	}
}

func (client PanelConfigurationClient) write(ctx context.Context, kind PanelConfigurationKind, deviceID string, payload map[string]any, credentials requestCredentials) error {
	var response map[string]any
	var err error
	switch kind {
	case PanelButtonConfigure:
		buttons, ok := payload[semantic.FieldButtons].([]any)
		if !ok || len(buttons) == 0 {
			return fmt.Errorf("buttons are required")
		}
		response, err = callJSONBody(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/panel/w/button/update/"+deviceID, buttons, credentials)
	case PanelButtonEventUpdate:
		event, ok := payload[semantic.FieldButtonEvent].(map[string]any)
		if !ok || strings.TrimSpace(stringFromAny(event[semantic.FieldButtonEventID])) == "" {
			return fmt.Errorf("button event is required")
		}
		event = mapWithoutKeys(event)
		event[semantic.FieldDeviceID] = deviceID
		response, err = callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/panel/w/button/event/update", event, credentials)
	case PanelButtonEventBatchUpdate:
		events, ok := payload[semantic.FieldButtonEvents].([]any)
		if !ok || len(events) == 0 {
			return fmt.Errorf("button events are required")
		}
		normalizedEvents := make([]any, 0, len(events))
		for _, rawEvent := range events {
			event, ok := rawEvent.(map[string]any)
			if !ok || strings.TrimSpace(stringFromAny(event[semantic.FieldButtonEventID])) == "" {
				return fmt.Errorf("button events are required")
			}
			event = mapWithoutKeys(event)
			event[semantic.FieldDeviceID] = deviceID
			normalizedEvents = append(normalizedEvents, event)
		}
		body := map[string]any{
			semantic.FieldButtonEvents: normalizedEvents,
		}
		response, err = callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/panel/w/button/event/update/batch", body, credentials)
	case PanelButtonEventReset:
		buttonEventID := strings.TrimSpace(stringFromAny(payload[semantic.FieldButtonEventID]))
		if buttonEventID == "" {
			return fmt.Errorf("button event id is required")
		}
		response, err = callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/panel/w/button/event/"+pathSegment(buttonEventID)+"/reset", nil, credentials)
	case KnobConfigure:
		body := map[string]any{
			semantic.FieldID:      deviceID,
			semantic.FieldDetails: payload[semantic.FieldDetails],
		}
		response, err = callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/multi-knob/update", body, credentials)
	case KnobReset:
		index := strings.TrimSpace(stringFromAny(payload[semantic.FieldIndex]))
		if index == "" {
			return fmt.Errorf("knob index is required")
		}
		response, err = callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/multi-knob/"+pathSegment(deviceID)+"/"+pathSegment(index)+"/reset", nil, credentials)
	default:
		return fmt.Errorf("unsupported panel configuration kind %q", kind)
	}
	if err != nil {
		return err
	}
	if !isBusinessOK(response) {
		return fmt.Errorf("%s returned non-success business response: code=%s message=%s dataType=%s", kind, responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return nil
}

func buildPanelButtonConfigureWritePayload(current any, deviceID string, payload map[string]any) (map[string]any, error) {
	expectedRows, ok := payload[semantic.FieldButtons].([]any)
	if !ok || len(expectedRows) == 0 {
		return nil, fmt.Errorf("buttons are required")
	}
	currentRows := configRowsFromData(current)
	if len(currentRows) == 0 {
		return nil, fmt.Errorf("current panel buttons are required")
	}
	merged := make([]any, 0, len(expectedRows))
	for _, rawExpected := range expectedRows {
		expected, ok := rawExpected.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("buttons are required")
		}
		base, ok := findPanelButtonBase(currentRows, expected)
		if !ok {
			return nil, fmt.Errorf("panel button reference not found")
		}
		item := mapWithoutKeys(base)
		for _, key := range semantic.PanelButtonWriteFields() {
			if value, exists := expected[key]; exists {
				item[key] = value
			}
		}
		item[semantic.FieldDeviceID] = deviceID
		if strings.TrimSpace(stringFromAny(item[semantic.FieldID])) == "" {
			if id := strings.TrimSpace(stringFromAny(expected[semantic.FieldID])); id != "" {
				item[semantic.FieldID] = id
			}
		}
		if strings.TrimSpace(stringFromAny(item[semantic.FieldType])) == "" {
			return nil, fmt.Errorf("panel button type is required")
		}
		merged = append(merged, item)
	}
	return map[string]any{
		semantic.FieldButtons: merged,
	}, nil
}

func findPanelButtonBase(rows []any, expected map[string]any) (map[string]any, bool) {
	for _, raw := range rows {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if panelButtonBaseMatches(row, expected) {
			return row, true
		}
	}
	return nil, false
}

func panelButtonBaseMatches(row map[string]any, expected map[string]any) bool {
	if expectedID := strings.TrimSpace(stringFromAny(expected[semantic.FieldID])); expectedID != "" {
		if strings.TrimSpace(stringFromAny(row[semantic.FieldID])) == expectedID {
			return true
		}
		if index := strings.TrimSpace(stringFromAny(row[semantic.FieldIndex])); index != "" && index == expectedID {
			return true
		}
		if keyValue := strings.TrimSpace(stringFromAny(row[semantic.FieldKeyValue])); keyValue != "" && keyValue == expectedID {
			return true
		}
	}
	for _, key := range []string{semantic.FieldID, semantic.FieldIndex, semantic.FieldKeyValue, semantic.FieldName, semantic.FieldAlias} {
		expectedValue := strings.TrimSpace(stringFromAny(expected[key]))
		if expectedValue == "" {
			continue
		}
		if strings.TrimSpace(stringFromAny(row[key])) == expectedValue {
			return true
		}
	}
	return false
}

func (client PanelConfigurationClient) verifyAfterWrite(ctx context.Context, kind PanelConfigurationKind, deviceID string, payload map[string]any, credentials requestCredentials, attempts int, interval time.Duration) (bool, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		var data any
		var readCalls int
		var err error
		switch kind {
		case PanelButtonConfigure, PanelButtonEventUpdate, PanelButtonEventBatchUpdate, PanelButtonEventReset:
			data, err = client.readPanel(ctx, deviceID, credentials)
			readCalls = 2
		case KnobConfigure, KnobReset:
			data, err = client.readKnob(ctx, deviceID, credentials)
			readCalls = 1
		default:
			return false, calls, fmt.Errorf("unsupported panel configuration kind %q", kind)
		}
		calls += readCalls
		if err != nil {
			return false, calls, err
		}
		if panelConfigurationMatches(kind, data, payload) || attempt == attempts-1 {
			return panelConfigurationMatches(kind, data, payload), calls, nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return false, calls, nil
}

func (client PanelConfigurationClient) readPanel(ctx context.Context, deviceID string, credentials requestCredentials) (map[string]any, error) {
	detail, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/panel/r/detail/"+deviceID, nil, credentials)
	if err != nil {
		return nil, err
	}
	if !isBusinessOK(detail) {
		return nil, metadataReadonlyBusinessError("panel detail", detail)
	}
	buttons, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/panel/r/button/info/"+deviceID, nil, credentials)
	if err != nil {
		return nil, err
	}
	if !isBusinessOK(buttons) {
		return nil, metadataReadonlyBusinessError("panel button info", buttons)
	}
	return map[string]any{
		semantic.FieldDetail:  detail["data"],
		semantic.FieldButtons: buttons["data"],
	}, nil
}

func (client PanelConfigurationClient) readKnob(ctx context.Context, deviceID string, credentials requestCredentials) (any, error) {
	response, err := callJSON(ctx, client.client, http.MethodGet, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/multi-knob/"+deviceID+"/detail", nil, credentials)
	if err != nil {
		return nil, err
	}
	if !isBusinessOK(response) {
		return nil, metadataReadonlyBusinessError("multi knob detail", response)
	}
	return response["data"], nil
}

func panelConfigurationMatches(kind PanelConfigurationKind, data any, payload map[string]any) bool {
	switch kind {
	case PanelButtonConfigure:
		source, ok := data.(map[string]any)
		if !ok {
			return false
		}
		return configRowsContainExpected(source[semantic.FieldButtons], payload[semantic.FieldButtons], panelButtonVerifyFields())
	case PanelButtonEventUpdate:
		source, ok := data.(map[string]any)
		if !ok {
			return false
		}
		keys := []string{semantic.FieldID, semantic.FieldButtonEventID, semantic.FieldAlias, semantic.FieldDetails}
		return configRowsContainExpected(source[semantic.FieldButtons], []any{payload[semantic.FieldButtonEvent]}, keys) ||
			configRowsContainExpected(source[semantic.FieldDetail], []any{payload[semantic.FieldButtonEvent]}, keys) ||
			configRowsContainExpected(source, []any{payload[semantic.FieldButtonEvent]}, keys)
	case PanelButtonEventBatchUpdate:
		source, ok := data.(map[string]any)
		if !ok {
			return false
		}
		keys := []string{semantic.FieldID, semantic.FieldButtonEventID, semantic.FieldAlias, semantic.FieldDetails}
		return configRowsContainExpected(source[semantic.FieldButtons], payload[semantic.FieldButtonEvents], keys) ||
			configRowsContainExpected(source[semantic.FieldDetail], payload[semantic.FieldButtonEvents], keys) ||
			configRowsContainExpected(source, payload[semantic.FieldButtonEvents], keys)
	case PanelButtonEventReset:
		_, ok := data.(map[string]any)
		return ok
	case KnobConfigure:
		if item, ok := data.(map[string]any); ok {
			return configRowsContainExpected(item[semantic.FieldDetails], payload[semantic.FieldDetails], knobDetailVerifyFields())
		}
		return configRowsContainExpected(data, payload[semantic.FieldDetails], knobDetailVerifyFields())
	case KnobReset:
		return data != nil
	default:
		return false
	}
}

func panelButtonVerifyFields() []string {
	return []string{
		semantic.FieldID,
		semantic.FieldName,
		semantic.FieldAlias,
		semantic.FieldKeyValue,
		semantic.FieldIndex,
		semantic.InternalField(semantic.DomainPanel, semantic.FieldTargetID),
		semantic.InternalField(semantic.DomainPanel, semantic.FieldTargetType),
		semantic.FieldVisible,
		semantic.FieldIcon,
		semantic.FieldSort,
		semantic.FieldType,
		semantic.FieldExtend,
	}
}

func knobDetailVerifyFields() []string {
	return []string{
		semantic.FieldID,
		semantic.FieldIndex,
		semantic.FieldConfigType,
		semantic.FieldMode,
		semantic.FieldModel,
		semantic.InternalField(semantic.DomainKnob, semantic.FieldTargetID),
		semantic.FieldResourceIndex,
		semantic.InternalField(semantic.DomainKnob, semantic.FieldTargetName),
		semantic.InternalKnobActionParamsField(),
		semantic.InternalField(semantic.DomainKnob, semantic.FieldSensitivity),
		semantic.FieldAction,
		semantic.FieldProperty,
		semantic.FieldValue,
	}
}

func configRowsContainExpected(actual any, expected any, keys []string) bool {
	expectedRows, ok := expected.([]any)
	if !ok || len(expectedRows) == 0 {
		return false
	}
	actualRows := configRowsFromData(actual)
	if len(actualRows) == 0 {
		return false
	}
	for _, rawExpected := range expectedRows {
		expectedItem, ok := rawExpected.(map[string]any)
		if !ok {
			return false
		}
		matched := false
		for _, rawActual := range actualRows {
			actualItem, ok := rawActual.(map[string]any)
			if !ok {
				continue
			}
			if configRowMatches(actualItem, expectedItem, keys) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func configRowMatches(actual map[string]any, expected map[string]any, keys []string) bool {
	for _, key := range keys {
		expectedValue, ok := expected[key]
		if !ok {
			continue
		}
		actualValue := actual[key]
		if key == semantic.FieldButtonEventID && strings.TrimSpace(stringFromAny(actualValue)) == "" {
			actualValue = actual[semantic.FieldID]
		}
		if !configValueContainsExpected(actualValue, expectedValue) {
			return false
		}
	}
	return true
}

func configValueContainsExpected(actual any, expected any) bool {
	switch expectedTyped := expected.(type) {
	case string, float64, int, int64:
		expectedText := strings.TrimSpace(stringFromAny(expectedTyped))
		return expectedText == "" || strings.TrimSpace(stringFromAny(actual)) == expectedText
	case bool:
		actualBool, ok := actual.(bool)
		return ok && actualBool == expectedTyped
	case []any:
		actualRows := configRowsFromData(actual)
		if len(actualRows) == 0 {
			return len(expectedTyped) == 0
		}
		for _, expectedItem := range expectedTyped {
			expectedMap, ok := expectedItem.(map[string]any)
			if !ok {
				return false
			}
			matched := false
			for _, actualItem := range actualRows {
				actualMap, ok := actualItem.(map[string]any)
				if ok && panelEventDetailContainsExpected(actualMap, expectedMap) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
		return true
	case map[string]any:
		if len(expectedTyped) == 0 {
			return true
		}
		actualMap, ok := actual.(map[string]any)
		return ok && configMapContainsExpected(actualMap, expectedTyped)
	default:
		return true
	}
}

func configMapContainsExpected(actual map[string]any, expected map[string]any) bool {
	for key, expectedValue := range expected {
		if key == semantic.FieldDetails {
			continue
		}
		if !configValueContainsExpected(actual[key], expectedValue) {
			return false
		}
	}
	if expectedDetails, ok := expected[semantic.FieldDetails]; ok {
		actualDetails := actual[semantic.FieldDetails]
		if !configValueContainsExpected(actualDetails, expectedDetails) {
			return false
		}
	}
	return true
}

func panelEventDetailContainsExpected(actual map[string]any, expected map[string]any) bool {
	if !configAnyFieldContainsExpected(actual, expected,
		semantic.InternalField(semantic.DomainPanel, semantic.FieldTargetType),
		semantic.InternalField(semantic.DomainKnob, semantic.FieldTargetType),
	) {
		return false
	}
	if !configAnyFieldContainsExpected(actual, expected,
		semantic.InternalPanelActionParamsField(),
		semantic.InternalKnobActionParamsField(),
	) {
		return false
	}
	for _, key := range []string{
		semantic.InternalField(semantic.DomainPanel, semantic.FieldTargetID),
		semantic.FieldRank,
		semantic.InternalField(semantic.DomainPanel, semantic.FieldSubIndex),
		semantic.InternalRepeatTypeField(),
		semantic.InternalRepeatValueField(),
		semantic.FieldStartTime,
		semantic.FieldEndTime,
	} {
		expectedValue, ok := expected[key]
		if !ok {
			continue
		}
		if !configValueContainsExpected(actual[key], expectedValue) {
			return false
		}
	}
	return true
}

func configAnyFieldContainsExpected(actual map[string]any, expected map[string]any, keys ...string) bool {
	var expectedValue any
	hasExpected := false
	for _, key := range keys {
		if value, ok := expected[key]; ok {
			expectedValue = value
			hasExpected = true
			break
		}
	}
	if !hasExpected {
		return true
	}
	for _, key := range keys {
		if configValueContainsExpected(actual[key], expectedValue) {
			return true
		}
	}
	return false
}

func configRowsFromData(data any) []any {
	switch typed := data.(type) {
	case []any:
		result := make([]any, 0, len(typed))
		for _, value := range typed {
			if _, ok := value.(map[string]any); ok {
				result = append(result, value)
			}
			result = append(result, configRowsFromData(value)...)
		}
		return result
	case map[string]any:
		result := []any{}
		if firstAnyString(typed, semantic.FieldID, semantic.FieldButtonEventID) != "" {
			result = append(result, typed)
		}
		for _, value := range typed {
			result = append(result, configRowsFromData(value)...)
		}
		return result
	default:
		return nil
	}
}

func stringFromJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}
