package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/auth"
	"github.com/yeelight/yeelight-home/internal/credential"
	"github.com/yeelight/yeelight-home/internal/semantic"
	"github.com/yeelight/yeelight-home/internal/storage"
)

func TestRootHelpAndVersionFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantOutput string
	}{
		{name: "empty args show root help", args: []string{}, wantOutput: "Usage:\n  yeelight-home <command> [flags]"},
		{name: "long help", args: []string{"--help"}, wantOutput: "Commands:\n  auth"},
		{name: "short help", args: []string{"-h"}, wantOutput: "Global flags:"},
		{name: "root help explains command model", args: []string{"--help"}, wantOutput: "Human-friendly operations use: yeelight-home <resource> <action> [flags]"},
		{name: "help command", args: []string{"help", "home"}, wantOutput: "yeelight-home home list"},
		{name: "module help command", args: []string{"help", "device"}, wantOutput: "yeelight-home device detail --device-id <id> --json"},
		{name: "intent help command", args: []string{"help", "intent"}, wantOutput: "yeelight-home intent explain --intent <intent> [--json]"},
		{name: "intent explain help command", args: []string{"help", "intent", "explain"}, wantOutput: "Returns accepted parameter fields, nested payloadShape, examples, and nextStep"},
		{name: "intent schema help command", args: []string{"help", "intent", "schema"}, wantOutput: "machine-readable SkillRequest schema"},
		{name: "explain help command", args: []string{"help", "explain"}, wantOutput: "yeelight-home explain <intent> [--json]"},
		{name: "module action help command", args: []string{"help", "scene", "execute"}, wantOutput: "Intent:\n  scene.execute"},
		{name: "scene update help shows payload shape", args: []string{"help", "scene", "update"}, wantOutput: "complete updated actions[] list"},
		{name: "scene update help shows millisecond timing", args: []string{"help", "scene", "update"}, wantOutput: "non-negative milliseconds"},
		{name: "scene update help shows source backed action keys", args: []string{"help", "scene", "update"}, wantOutput: "blink, motorAdjust, delayCancel"},
		{name: "scene update help shows audio action evidence keys", args: []string{"help", "scene", "update"}, wantOutput: "musicPlayerCtrl, or localAudioCtrl"},
		{name: "scene update help shows evidence-only custom actions", args: []string{"help", "scene", "update"}, wantOutput: "only when Runtime detail/capability"},
		{name: "scene create help shows actions shape", args: []string{"help", "scene", "create"}, wantOutput: "actions[] item fields:"},
		{name: "automation update help shows payload shape", args: []string{"help", "automation", "update"}, wantOutput: "complete condition/action payload"},
		{name: "automation update help shows source backed condition fields", args: []string{"help", "automation", "update"}, wantOutput: "targetType, targetId, property, operation, value"},
		{name: "automation create help shows actions shape", args: []string{"help", "automation", "create"}, wantOutput: "actions[] item fields:"},
		{name: "scene action help shows standard target fields", args: []string{"help", "scene", "update"}, wantOutput: "targetType=device|group|meshGroup|scene"},
		{name: "automation action help shows standard target fields", args: []string{"help", "automation", "update"}, wantOutput: "targetType=device|group|meshGroup|scene"},
		{name: "lighting apply help shows properties", args: []string{"help", "lighting", "apply"}, wantOutput: "property      one of power, brightness, colorTemperature, color"},
		{name: "lighting import help shows design model", args: []string{"help", "lighting", "import"}, wantOutput: "standard lighting design model"},
		{name: "lighting import help shows product fields", args: []string{"help", "lighting", "import"}, wantOutput: "skuCode, capabilityPid, productComponentId"},
		{name: "lighting import help shows group fields", args: []string{"help", "lighting", "import"}, wantOutput: "groupCapability, slotKeys[]"},
		{name: "device module help uses design slot example", args: []string{"help", "device"}, wantOutput: "\"deviceSlots\""},
		{name: "lighting module help uses design import example", args: []string{"help", "lighting"}, wantOutput: "\"slotKeys\""},
		{name: "panel button event help shows actions", args: []string{"help", "panel", "button-event-update"}, wantOutput: "Button event updates replace the target event's complete action list"},
		{name: "knob configure help shows config type", args: []string{"help", "knob", "configure"}, wantOutput: "index, configType, targetType, targetId"},
		{name: "knob configure help shows event evidence words", args: []string{"help", "knob", "configure"}, wantOutput: "rotate, press_rotate, click, double_click, and hold"},
		{name: "operation batch help shows operations", args: []string{"help", "operation", "batch-configure"}, wantOutput: "operations[] is an ordered list"},
		{name: "favorite batch help shows items", args: []string{"help", "favorite", "batch-add"}, wantOutput: "Batch favorite intents use"},
		{name: "room batch create help shows rooms", args: []string{"help", "room", "batch-create"}, wantOutput: "rooms[] or items[] contains"},
		{name: "area update help shows complete room association", args: []string{"help", "area", "update"}, wantOutput: "roomIds is a complete association list"},
		{name: "batch delete help shows items ids names", args: []string{"help", "scene", "batch-delete"}, wantOutput: "items[], ids[], or names[]"},
		{name: "module action trailing help", args: []string{"light", "brightness", "--help"}, wantOutput: "--brightness <1-100>"},
		{name: "module trailing help", args: []string{"scene", "--help"}, wantOutput: "batch-delete"},
		{name: "subcommand help", args: []string{"home", "--help"}, wantOutput: "home list is account-scoped"},
		{name: "nested help command", args: []string{"help", "auth", "token", "set"}, wantOutput: "Omit houseId for token-only account-scoped use"},
		{name: "nested trailing help", args: []string{"auth", "token", "set", "--help"}, wantOutput: "yeelight-home auth token set (--token"},
		{name: "nested dev help", args: []string{"dev", "seed-room", "--help"}, wantOutput: "yeelight-home dev seed-room --json --region dev"},
		{name: "completion help", args: []string{"completion", "--help"}, wantOutput: "yeelight-home completion <bash|zsh|fish|powershell>"},
		{name: "long version", args: []string{"--version"}, wantOutput: "yeelight-home dev"},
		{name: "short version", args: []string{"-v"}, wantOutput: "yeelight-home dev"},
		{name: "version command json help", args: []string{"help", "version"}, wantOutput: "yeelight-home version [--json]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run(test.args, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			if !strings.Contains(stdout.String(), test.wantOutput) {
				t.Fatalf("stdout = %s, want substring %q", stdout.String(), test.wantOutput)
			}
		})
	}
}

func TestVersionJSONReportsBuildMetadata(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"version", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["cli"] != "yeelight-home" || response["version"] != version || response["commit"] == "" || response["date"] == "" || response["os"] == "" || response["arch"] == "" {
		t.Fatalf("response = %#v", response)
	}
}

func TestCompletionCommandPrintsShellScripts(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantOutput   string
		forbidOutput []string
		wantCode     int
	}{
		{name: "bash", args: []string{"completion", "bash"}, wantOutput: "device) COMPREPLY=( $(compgen -W \"attrs capabilities detail", forbidOutput: []string{" dev ", " release "}},
		{name: "zsh", args: []string{"completion", "zsh"}, wantOutput: "device) local -a actions; actions=('attrs' 'capabilities' 'detail'", forbidOutput: []string{"'dev'", "'release'"}},
		{name: "fish", args: []string{"completion", "fish"}, wantOutput: "complete -c yeelight-home", forbidOutput: []string{" -a dev\n", " -a release\n"}},
		{name: "powershell", args: []string{"completion", "powershell"}, wantOutput: "Register-ArgumentCompleter", forbidOutput: []string{"'dev'", "'release'"}},
		{name: "unsupported shell", args: []string{"completion", "tcsh"}, wantCode: exitInvalidInput},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run(test.args, strings.NewReader(""), &stdout, &stderr)
			wantCode := test.wantCode
			if wantCode == 0 {
				wantCode = exitOK
			}
			if code != wantCode {
				t.Fatalf("exit code = %d, want %d, stdout=%s stderr=%s", code, wantCode, stdout.String(), stderr.String())
			}
			if test.wantOutput != "" && !strings.Contains(stdout.String(), test.wantOutput) {
				t.Fatalf("stdout = %s, want substring %q", stdout.String(), test.wantOutput)
			}
			for _, forbidden := range test.forbidOutput {
				if strings.Contains(stdout.String(), forbidden) {
					t.Fatalf("stdout contains forbidden substring %q: %s", forbidden, stdout.String())
				}
			}
		})
	}
}

func TestIntentExplainReturnsMachineReadablePayloadGuide(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"intent", "explain", "--intent", "lighting.design.import", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response[semantic.FieldIntent] != "lighting.design.import" || response[semantic.FieldImplemented] != true {
		t.Fatalf("response = %#v", response)
	}
	guide := response[semantic.FieldPayloadGuide].(map[string]any)
	shape := guide[semantic.FieldPayloadShape].(map[string]any)
	if shape[semantic.FieldRooms] == nil || shape[semantic.FieldScenes] == nil || shape[semantic.FieldAutomations] == nil {
		t.Fatalf("payload shape = %#v", shape)
	}
	if !strings.Contains(response[semantic.FieldNextStep].(string), "standard lighting design model") {
		t.Fatalf("nextStep = %#v", response[semantic.FieldNextStep])
	}
	fields := response[semantic.FieldAcceptedFields].([]any)
	if len(fields) == 0 {
		t.Fatalf("acceptedFields empty")
	}
	for _, field := range []string{
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldDeviceSlots)),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldGroups), semantic.FieldSlotKeys),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldScenes), semantic.ArrayField(semantic.FieldActions)),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldAutomations), semantic.FieldTrigger),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
	text := string(stdout.Bytes())
	for _, forbidden := range []string{"\"tid\"", "\"n\"", "\"rl\"", "\"atl\"", "HouseMeta", "/v1/meta/import", "deviceTempIdList"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("intent explain should not recommend short-key HouseMeta fields %s: %s", forbidden, text)
		}
	}
}

func TestIntentExplainThingProductInfoV3ShowsProductVersionContext(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"intent", "explain", "--intent", "thing.product.info.v3.batch_get", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response[semantic.FieldIntent] != "thing.product.info.v3.batch_get" || response[semantic.FieldImplemented] != true {
		t.Fatalf("response = %#v", response)
	}
	fields := response[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldCapabilityProductID),
		semantic.ParameterPath(semantic.FieldCapabilityProductIDs),
		semantic.ParameterPath(semantic.FieldVersion),
		semantic.ParameterPath(semantic.FieldSchemaVersion),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
	guide := response[semantic.FieldPayloadGuide].(map[string]any)
	shape := guide[semantic.FieldPayloadShape].(map[string]any)
	if shape[semantic.FieldCapabilityProductID] == nil || shape[semantic.FieldVersion] == nil || shape[semantic.FieldSchemaVersion] == nil {
		t.Fatalf("payload shape = %#v", shape)
	}
	if !strings.Contains(response[semantic.FieldNextStep].(string), "version") {
		t.Fatalf("nextStep = %#v", response[semantic.FieldNextStep])
	}
}

func TestIntentExplainProductFAQShowsSearchAndDetailContext(t *testing.T) {
	tests := []struct {
		intent string
		fields []string
	}{
		{
			intent: "thing.product_faq.detail.get",
			fields: []string{
				semantic.ParameterPath(semantic.FieldFAQID),
				semantic.ParameterPath(semantic.FieldID),
			},
		},
		{
			intent: "thing.product_faq.page_detail.list",
			fields: []string{
				semantic.ParameterPath(semantic.FieldCapabilityProductID),
				semantic.ParameterPath(semantic.FieldModuleID),
				semantic.ParameterPath(semantic.FieldKeyword),
				semantic.ParameterPath(semantic.FieldPageNo),
				semantic.ParameterPath(semantic.FieldPageSize),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.intent, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := run([]string{"intent", "explain", "--intent", test.intent, "--json"}, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			var response map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatalf("invalid json response: %v", err)
			}
			if response[semantic.FieldIntent] != test.intent || response[semantic.FieldImplemented] != true {
				t.Fatalf("response = %#v", response)
			}
			fields := response[semantic.FieldAcceptedFields].([]any)
			for _, field := range test.fields {
				if !containsAnyString(fields, field) {
					t.Fatalf("acceptedFields should include %s: %#v", field, fields)
				}
			}
			guide := response[semantic.FieldPayloadGuide].(map[string]any)
			shape := guide[semantic.FieldPayloadShape].(map[string]any)
			for _, field := range test.fields {
				key := strings.TrimPrefix(field, semantic.FieldParameters+".")
				if shape[key] == nil {
					t.Fatalf("payload shape should include %s: %#v", key, shape)
				}
			}
			if !strings.Contains(response[semantic.FieldNextStep].(string), "FAQ") {
				t.Fatalf("nextStep = %#v", response[semantic.FieldNextStep])
			}
		})
	}
}

func TestIntentSchemaCommandReturnsSkillRequestSchema(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"intent", "schema", "--intent", "lighting.design.import", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("invalid json schema: %v", err)
	}
	properties := schema["properties"].(map[string]any)
	intentSchema := properties["intent"].(map[string]any)
	if intentSchema["const"] != "lighting.design.import" {
		t.Fatalf("intent schema = %#v", intentSchema)
	}
	parameters := properties["parameters"].(map[string]any)
	parameterProperties := parameters["properties"].(map[string]any)
	if parameterProperties["rooms"] == nil || parameterProperties["scenes"] == nil || parameterProperties["automations"] == nil {
		t.Fatalf("parameter properties = %#v", parameterProperties)
	}
}

func TestIntentSchemaCommandKeepsOperationBatchStepIntentString(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"intent", "schema", "--intent", "operation.batch.configure", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("invalid json schema: %v", err)
	}
	properties := schema["properties"].(map[string]any)
	parameters := properties["parameters"].(map[string]any)
	parameterProperties := parameters["properties"].(map[string]any)
	operations := parameterProperties[semantic.FieldOperations].(map[string]any)
	items := operations["items"].(map[string]any)
	itemProperties := items["properties"].(map[string]any)
	intentSchema := itemProperties[semantic.FieldIntent].(map[string]any)
	if intentSchema["type"] != "string" {
		t.Fatalf("operation step intent schema = %#v", intentSchema)
	}
	targetsSchema := itemProperties[semantic.FieldTargets].(map[string]any)
	if targetsSchema["type"] != "array" {
		t.Fatalf("operation step targets schema = %#v", targetsSchema)
	}
}

func TestIntentSchemaKeepsPluralNameFieldsAsArrays(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"intent", "schema", "--intent", "gateway.configure", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("invalid json schema: %v", err)
	}
	properties := schema["properties"].(map[string]any)
	parameters := properties["parameters"].(map[string]any)
	parameterProperties := parameters["properties"].(map[string]any)
	roomNames := parameterProperties[semantic.FieldRoomNames].(map[string]any)
	if roomNames["type"] != "array" {
		t.Fatalf("roomNames schema = %#v", roomNames)
	}
}

func TestIntentSchemaShowsSemanticActionAliases(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"intent", "schema", "--intent", "scene.update", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	text := stdout.String()
	for _, want := range []string{"targetType", "targetId", "targetName", "set"} {
		if !strings.Contains(text, want) {
			t.Fatalf("schema missing semantic alias %q: %s", want, text)
		}
	}
}

func TestIntentSchemaSceneUpdateShowsNaturalSceneTarget(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"intent", "schema", "--intent", "scene.update", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	text := stdout.String()
	for _, want := range []string{"sceneName", "currentName", "newName"} {
		if !strings.Contains(text, want) {
			t.Fatalf("schema missing scene update target field %q: %s", want, text)
		}
	}
}

func TestIntentSchemaPayloadGuidesPreserveHouseID(t *testing.T) {
	for _, intent := range []string{"scene.create", "automation.create", "group.create"} {
		t.Run(intent, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run([]string{"intent", "schema", "--intent", intent, "--json"}, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			var schema map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
				t.Fatalf("invalid json schema: %v", err)
			}
			parameters := schema["properties"].(map[string]any)[semantic.FieldParameters].(map[string]any)
			properties := parameters["properties"].(map[string]any)
			if properties[semantic.FieldHouseID] == nil {
				t.Fatalf("schema for %s missing houseId: %s", intent, stdout.String())
			}
		})
	}
}

func TestIntentExplainProductPediaShowsQueryFields(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"intent", "explain", "--intent", "product.pedia.search", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	fields := response[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldQuery),
		semantic.ParameterPath(semantic.FieldKeyword),
		semantic.ParameterPath(semantic.FieldSKUCode),
		semantic.ParameterPath(semantic.FieldProductModel),
		semantic.ParameterPath(semantic.FieldLimit),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
}

func TestIntentExplainTargetIntentsShowNaturalTargetFields(t *testing.T) {
	tests := map[string][]string{
		"diagnose.device": {
			semantic.ParameterPath(semantic.FieldDeviceName),
			semantic.ParameterPath(semantic.FieldRoomName),
		},
		"panel.get": {
			semantic.ParameterPath(semantic.FieldDeviceName),
		},
		"scene.execute": {
			semantic.ParameterPath(semantic.FieldSceneName),
		},
		"automation.disable": {
			semantic.ParameterPath(semantic.FieldAutomationName),
		},
		"gateway.detail.get": {
			semantic.ParameterPath(semantic.FieldGatewayName),
		},
	}
	for intent, expectedFields := range tests {
		t.Run(intent, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run([]string{"intent", "explain", "--intent", intent, "--json"}, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			var response map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatalf("invalid json response: %v", err)
			}
			fields := response[semantic.FieldAcceptedFields].([]any)
			for _, field := range expectedFields {
				if !containsAnyString(fields, field) {
					t.Fatalf("acceptedFields should include %s: %#v", field, fields)
				}
			}
		})
	}
}

func TestIntentSchemaAutomationUpdateShowsNaturalAutomationTarget(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"intent", "schema", "--intent", "automation.update", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	text := stdout.String()
	for _, want := range []string{"automationName", "currentName", "newName"} {
		if !strings.Contains(text, want) {
			t.Fatalf("schema missing automation update target field %q: %s", want, text)
		}
	}
}

func TestIntentExplainGroupCreateShowsSemanticMemberShape(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"intent", "explain", "--intent", "group.create", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	fields := response[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldName),
		semantic.ParameterPath(semantic.FieldRoomID),
		semantic.ParameterPath(semantic.FieldRoomName),
		semantic.ParameterPath(semantic.FieldGroupCapability),
		semantic.ParameterPath(semantic.FieldDeviceIDs),
		semantic.ParameterPath(semantic.FieldDeviceNames),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
	text := stdout.String()
	if strings.Contains(text, "componentId") || strings.Contains(text, "groupComponent") {
		t.Fatalf("group.create explain should not expose lower-level group component fields: %s", text)
	}
}

func TestIntentExplainGroupUpdateShowsSemanticShape(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"intent", "explain", "--intent", "group.update", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	text := stdout.String()
	for _, want := range []string{"groupId", "groupName", "currentName", "name", "description", "icon", "roomId", "targetRoomName"} {
		if !strings.Contains(text, want) {
			t.Fatalf("group.update explain missing %s: %s", want, text)
		}
	}
	if strings.Contains(text, "componentId") || strings.Contains(text, "groupComponent") {
		t.Fatalf("group.update explain should not expose lower-level group component fields: %s", text)
	}
}

func TestIntentExplainSpaceOrganizationShowsNaturalNameFields(t *testing.T) {
	tests := map[string][]string{
		"room.rename": {
			semantic.ParameterPath(semantic.FieldRoomName),
			semantic.ParameterPath(semantic.FieldCurrentName),
			semantic.ParameterPath(semantic.FieldNewName),
		},
		"device.rename": {
			semantic.ParameterPath(semantic.FieldDeviceName),
			semantic.ParameterPath(semantic.FieldCurrentName),
			semantic.ParameterPath(semantic.FieldNewName),
		},
		"device.move": {
			semantic.ParameterPath(semantic.FieldDeviceName),
			semantic.ParameterPath(semantic.FieldTargetRoomName),
		},
		"favorite.add": {
			semantic.ParameterPath(semantic.FieldEntityType),
			semantic.ParameterPath(semantic.FieldEntityID),
			semantic.ParameterPath(semantic.FieldTargetType),
			semantic.ParameterPath(semantic.FieldTargetID),
		},
	}
	for intent, expectedFields := range tests {
		t.Run(intent, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run([]string{"intent", "explain", "--intent", intent, "--json"}, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			var response map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatalf("invalid json response: %v", err)
			}
			fields := response[semantic.FieldAcceptedFields].([]any)
			for _, field := range expectedFields {
				if !containsAnyString(fields, field) {
					t.Fatalf("acceptedFields should include %s: %#v", field, fields)
				}
			}
		})
	}
}

func TestIntentSchemaFavoriteAddDoesNotRequireUpdateOrBatchFields(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"intent", "schema", "--intent", "favorite.add", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("invalid json schema: %v", err)
	}
	parameters := schema["properties"].(map[string]any)[semantic.FieldParameters].(map[string]any)
	if required, ok := parameters["required"].([]any); ok {
		for _, forbidden := range []string{semantic.FieldFavoriteID, semantic.FieldItems} {
			if containsAnyString(required, forbidden) {
				t.Fatalf("favorite.add schema must not require %s: %#v", forbidden, required)
			}
		}
	}
	properties := parameters["properties"].(map[string]any)
	for _, field := range []string{semantic.FieldTargetType, semantic.FieldTargetID, semantic.FieldEntityType, semantic.FieldEntityID} {
		if properties[field] == nil {
			t.Fatalf("favorite.add schema missing %s: %#v", field, properties)
		}
	}
}

func TestIntentExplainLightControlShowsValueFields(t *testing.T) {
	tests := []struct {
		intent string
		fields []string
	}{
		{intent: "light.power.set", fields: []string{semantic.ParameterPath(semantic.FieldPower), semantic.ParameterPath(semantic.FieldDeviceName), semantic.ParameterPath(semantic.FieldRoomName)}},
		{intent: "light.brightness.set", fields: []string{semantic.ParameterPath(semantic.FieldBrightness), semantic.ParameterPath(semantic.FieldDeviceName), semantic.ParameterPath(semantic.FieldRoomName)}},
		{intent: "light.color_temperature.set", fields: []string{semantic.ParameterPath(semantic.FieldColorTemperature), semantic.ParameterPath(semantic.FieldDeviceName), semantic.ParameterPath(semantic.FieldRoomName)}},
		{intent: "light.color.set", fields: []string{semantic.ParameterPath(semantic.FieldColor), semantic.ParameterPath(semantic.FieldHex), semantic.ParameterPath(semantic.FieldDeviceName), semantic.ParameterPath(semantic.FieldRoomName)}},
	}
	for _, test := range tests {
		t.Run(test.intent, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run([]string{"intent", "explain", "--intent", test.intent, "--json"}, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			var response map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatalf("invalid json response: %v", err)
			}
			fields := response[semantic.FieldAcceptedFields].([]any)
			for _, field := range test.fields {
				if !containsAnyString(fields, field) {
					t.Fatalf("acceptedFields should include %s: %#v", field, fields)
				}
			}
		})
	}
}

func TestIntentExplainLightingExperienceApplyShowsExplicitActionFields(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"intent", "explain", "--intent", "lighting.experience.apply", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	fields := response[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldBrightness),
		semantic.ParameterPath(semantic.FieldColorTemperature),
		semantic.ParameterPath(semantic.FieldColor),
		semantic.ParameterPath(semantic.FieldHex),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
	guide := response[semantic.FieldPayloadGuide].(map[string]any)
	shape := guide[semantic.FieldPayloadShape].(map[string]any)
	if shape[semantic.FieldBrightness] == nil || shape[semantic.FieldColorTemperature] == nil || shape[semantic.FieldColor] == nil {
		t.Fatalf("payload shape = %#v", shape)
	}
	if !strings.Contains(response[semantic.FieldNextStep].(string), "does not invent") {
		t.Fatalf("nextStep = %#v", response[semantic.FieldNextStep])
	}
}

func TestIntentExplainPanelButtonTypeGetShowsButtonTypeField(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"intent", "explain", "--intent", "panel.button.type.get", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	fields := response[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldDeviceID),
		semantic.ParameterPath(semantic.FieldDeviceName),
		semantic.ParameterPath(semantic.FieldButtonType),
		semantic.ParameterPath(semantic.FieldType),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
	guide := response[semantic.FieldPayloadGuide].(map[string]any)
	shape := guide[semantic.FieldPayloadShape].(map[string]any)
	if shape[semantic.FieldButtonType] == nil || shape[semantic.FieldType] == nil {
		t.Fatalf("payload shape = %#v", shape)
	}
	if !strings.Contains(response[semantic.FieldNextStep].(string), "panel.get first") {
		t.Fatalf("nextStep = %#v", response[semantic.FieldNextStep])
	}
}

func TestIntentExplainNodePropertyConfigGetShowsNodeTypeField(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"intent", "explain", "--intent", "node.property_config.get", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	fields := response[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldNodeID),
		semantic.ParameterPath(semantic.FieldDeviceID),
		semantic.ParameterPath(semantic.FieldNodeType),
		semantic.ParameterPath(semantic.FieldType),
		semantic.ParameterPath(semantic.FieldEntityType),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
	guide := response[semantic.FieldPayloadGuide].(map[string]any)
	shape := guide[semantic.FieldPayloadShape].(map[string]any)
	if shape[semantic.FieldNodeType] == nil || shape[semantic.FieldNodeID] == nil {
		t.Fatalf("payload shape = %#v", shape)
	}
	if !strings.Contains(response[semantic.FieldNextStep].(string), "Pass nodeType explicitly") {
		t.Fatalf("nextStep = %#v", response[semantic.FieldNextStep])
	}
}

func TestIntentExplainAppUpgradeLatestGetShowsSemanticFields(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"intent", "explain", "--intent", "app_upgrade.latest.get", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	fields := response[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldAppType),
		semantic.ParameterPath(semantic.FieldOSType),
		semantic.ParameterPath(semantic.FieldLanguageCode),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
	if !strings.Contains(response[semantic.FieldNextStep].(string), "semantic appType") {
		t.Fatalf("nextStep = %#v", response[semantic.FieldNextStep])
	}
}

func TestIntentExplainProgressGetShowsProgressKeyField(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"intent", "explain", "--intent", "progress.get", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	fields := response[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldProgressKey),
		semantic.ParameterPath(semantic.FieldKey),
		semantic.ParameterPath(semantic.FieldID),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
	if !strings.Contains(response[semantic.FieldNextStep].(string), "concrete progressKey") {
		t.Fatalf("nextStep = %#v", response[semantic.FieldNextStep])
	}
}

func TestIntentSchemaLightColorAllowsRGBObject(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"intent", "schema", "--intent", "light.color.set", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	parameters := schema["properties"].(map[string]any)[semantic.FieldParameters].(map[string]any)
	color := parameters["properties"].(map[string]any)[semantic.FieldColor].(map[string]any)
	colorTypes, ok := color["type"].([]any)
	if !ok || !containsAnyString(colorTypes, "object") {
		t.Fatalf("color schema should allow object: %#v", color)
	}
}

func TestIntentSchemaAutomationActionSetIsOptionalObject(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"intent", "schema", "--intent", "automation.create", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	parameters := schema["properties"].(map[string]any)[semantic.FieldParameters].(map[string]any)
	actions := parameters["properties"].(map[string]any)[semantic.FieldActions].(map[string]any)
	item := actions["items"].(map[string]any)
	required, _ := item["required"].([]any)
	if containsAnyString(required, semantic.FieldSet) {
		t.Fatalf("automation action set must not be globally required: %#v", item)
	}
	setSchema := item["properties"].(map[string]any)[semantic.FieldSet].(map[string]any)
	if setSchema["type"] != "object" {
		t.Fatalf("automation action set should be object: %#v", setSchema)
	}
}

func TestIntentSchemaAutomationTriggerShowsAdvancedConditionFields(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"intent", "schema", "--intent", "automation.create", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	parameters := schema["properties"].(map[string]any)[semantic.FieldParameters].(map[string]any)
	trigger := parameters["properties"].(map[string]any)[semantic.FieldTrigger].(map[string]any)
	triggerProperties := trigger["properties"].(map[string]any)
	for _, field := range []string{
		semantic.FieldConditionKind,
		semantic.FieldTime,
		semantic.FieldTargetType,
		semantic.FieldTargetID,
		semantic.FieldCapabilityProductID,
		semantic.FieldEventID,
		semantic.FieldEventArgs,
		semantic.FieldProperty,
		semantic.FieldOperation,
		semantic.FieldValue,
	} {
		if _, ok := triggerProperties[field]; !ok {
			t.Fatalf("automation trigger schema missing %s: %#v", field, triggerProperties)
		}
	}
}

func TestIntentSchemaEntityRenameBatchAllowsCurrentNameWithoutID(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"intent", "schema", "--intent", "entity.rename.batch", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	parameters := schema["properties"].(map[string]any)[semantic.FieldParameters].(map[string]any)
	items := parameters["properties"].(map[string]any)[semantic.FieldItems].(map[string]any)
	item := items["items"].(map[string]any)
	required, _ := item["required"].([]any)
	if containsAnyString(required, semantic.FieldID) {
		t.Fatalf("entity.rename.batch item id must not be globally required: %#v", item)
	}
}

func TestIntentExplainBasicCreateIntentsShowSemanticShape(t *testing.T) {
	tests := map[string][]string{
		"home.create": {
			semantic.ParameterPath(semantic.FieldName),
			semantic.ParameterPath(semantic.FieldDescription),
			semantic.ParameterPath(semantic.FieldAreaName),
		},
		"room.create": {
			semantic.ParameterPath(semantic.FieldName),
			semantic.ParameterPath(semantic.FieldRoomName),
			semantic.ParameterPath(semantic.FieldIcon),
		},
		"area.create": {
			semantic.ParameterPath(semantic.FieldName),
			semantic.ParameterPath(semantic.FieldRoomIDs),
			semantic.ParameterPath(semantic.FieldParentID),
		},
	}
	for intent, wantFields := range tests {
		t.Run(intent, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run([]string{"intent", "explain", "--intent", intent, "--json"}, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			var response map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatalf("invalid json response: %v", err)
			}
			fields := response[semantic.FieldAcceptedFields].([]any)
			for _, field := range wantFields {
				if !containsAnyString(fields, field) {
					t.Fatalf("acceptedFields should include %s: %#v", field, fields)
				}
			}
		})
	}
}

func TestIntentSchemaRequiresExistingIDOnlyForUpdates(t *testing.T) {
	tests := []struct {
		intent          string
		idField         string
		wantIDRule      bool
		wantRequired    []string
		wantNotRequired []string
	}{
		{intent: "scene.create", idField: "sceneId", wantIDRule: false, wantRequired: []string{"name", "actions"}, wantNotRequired: []string{"sceneId"}},
		{intent: "scene.update", idField: "sceneId", wantIDRule: false, wantRequired: []string{"actions"}, wantNotRequired: []string{"sceneId", "name"}},
		{intent: "automation.create", idField: "automationId", wantIDRule: false, wantRequired: []string{"name", "trigger", "actions"}, wantNotRequired: []string{"automationId"}},
		{intent: "automation.update", idField: "automationId", wantIDRule: false, wantRequired: []string{"trigger", "actions"}, wantNotRequired: []string{"automationId", "name"}},
	}
	for _, test := range tests {
		t.Run(test.intent, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run([]string{"intent", "schema", "--intent", test.intent, "--json"}, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			var schema map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
				t.Fatalf("invalid json schema: %v", err)
			}
			required := parameterRequiredFields(schema)
			hasIDRule := containsAnyString(required, test.idField)
			if hasIDRule != test.wantIDRule {
				t.Fatalf("%s required fields = %#v", test.intent, required)
			}
			for _, field := range test.wantRequired {
				if !containsAnyString(required, field) {
					t.Fatalf("%s should require %s; required = %#v", test.intent, field, required)
				}
			}
			for _, field := range test.wantNotRequired {
				if containsAnyString(required, field) {
					t.Fatalf("%s should not require %s; required = %#v", test.intent, field, required)
				}
			}
		})
	}
}

func TestIntentSchemaActionRowsAllowTargetNameWithoutTargetID(t *testing.T) {
	for _, intent := range []string{"scene.create", "automation.create"} {
		t.Run(intent, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run([]string{"intent", "schema", "--intent", intent, "--json"}, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			var schema map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
				t.Fatalf("invalid json schema: %v", err)
			}
			actionRequired := parameterArrayItemRequiredFields(t, schema, semantic.FieldActions)
			if !containsAnyString(actionRequired, semantic.FieldTargetType) {
				t.Fatalf("%s action rows should require targetType; required = %#v", intent, actionRequired)
			}
			if containsAnyString(actionRequired, semantic.FieldTargetID) {
				t.Fatalf("%s action rows must allow targetName without targetId; required = %#v", intent, actionRequired)
			}
		})
	}
}

func TestExplainAliasReturnsIntentSchema(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"explain", "scene.update", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatalf("invalid json schema: %v", err)
	}
	properties := schema["properties"].(map[string]any)
	intentSchema := properties["intent"].(map[string]any)
	if intentSchema["const"] != "scene.update" {
		t.Fatalf("intent schema = %#v", intentSchema)
	}
	if schema[semantic.FieldExamples] == nil || schema[semantic.FieldNextStep] == nil {
		t.Fatalf("schema missing examples/nextStep: %#v", schema)
	}
}

func parameterRequiredFields(schema map[string]any) []any {
	properties := schema["properties"].(map[string]any)
	parameters := properties["parameters"].(map[string]any)
	required, _ := parameters["required"].([]any)
	return required
}

func parameterArrayItemRequiredFields(t *testing.T, schema map[string]any, field string) []any {
	t.Helper()
	properties := schema["properties"].(map[string]any)
	parameters := properties["parameters"].(map[string]any)
	parameterProperties := parameters["properties"].(map[string]any)
	arraySchema := parameterProperties[field].(map[string]any)
	itemSchema := arraySchema["items"].(map[string]any)
	required, _ := itemSchema["required"].([]any)
	return required
}

func TestInvokeIntentExplainReturnsMachineReadablePayloadGuide(t *testing.T) {
	app := newTestApp(t)
	input := `{"contractVersion":"1.0","requestId":"req-intent-explain","locale":"zh-CN","utterance":"解释照明设计导入参数","intent":"intent.explain","parameters":{"intent":"lighting.design.import"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response[semantic.FieldStatus] != "success" {
		t.Fatalf("response = %#v", response)
	}
	result := response[semantic.FieldResult].(map[string]any)
	explanation := result[semantic.FieldIntentExplanation].(map[string]any)
	if explanation[semantic.FieldIntent] != "lighting.design.import" || explanation[semantic.FieldImplemented] != true {
		t.Fatalf("intent explanation = %#v", explanation)
	}
	guide := explanation[semantic.FieldPayloadGuide].(map[string]any)
	shape := guide[semantic.FieldPayloadShape].(map[string]any)
	if shape[semantic.FieldRooms] == nil || shape[semantic.FieldScenes] == nil || shape[semantic.FieldAutomations] == nil {
		t.Fatalf("payload shape = %#v", shape)
	}
	if !strings.Contains(explanation[semantic.FieldNextStep].(string), "standard lighting design model") {
		t.Fatalf("nextStep = %#v", explanation[semantic.FieldNextStep])
	}
	fields := explanation[semantic.FieldAcceptedFields].([]any)
	if !containsAnyString(fields, semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldDeviceSlots), semantic.FieldProduct, semantic.FieldProductCode)) {
		t.Fatalf("acceptedFields missing nested design product field: %#v", fields)
	}
}

func TestInvokeIntentExplainHomeMemberIncludesPublicPayloadGuide(t *testing.T) {
	app := newTestApp(t)
	input := `{"contractVersion":"1.0","requestId":"req-intent-explain-member","locale":"zh-CN","utterance":"解释家庭分享参数","intent":"intent.explain","parameters":{"intent":"home.member.invite"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	explanation := response[semantic.FieldResult].(map[string]any)[semantic.FieldIntentExplanation].(map[string]any)
	guide := explanation[semantic.FieldPayloadGuide].(map[string]any)
	shape := guide[semantic.FieldPayloadShape].(map[string]any)
	for _, field := range []string{semantic.FieldHouseID, semantic.FieldExpiresAt, semantic.FieldUserRole, semantic.FieldReuseBarcode} {
		if shape[field] == nil {
			t.Fatalf("payload shape should include %s: %#v", field, shape)
		}
	}
	fields := explanation[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldExpiresAt),
		semantic.ParameterPath(semantic.FieldUserRole),
		semantic.ParameterPath(semantic.FieldReuseBarcode),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
}

func TestInvokeIntentExplainOperationLessonRecordIncludesPublicPayloadGuide(t *testing.T) {
	app := newTestApp(t)
	input := `{"contractVersion":"1.0","requestId":"req-intent-explain-lesson","locale":"zh-CN","utterance":"解释经验记录参数","intent":"intent.explain","parameters":{"intent":"operation.lesson.record"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	explanation := response[semantic.FieldResult].(map[string]any)[semantic.FieldIntentExplanation].(map[string]any)
	guide := explanation[semantic.FieldPayloadGuide].(map[string]any)
	shape := guide[semantic.FieldPayloadShape].(map[string]any)
	lessonShape := shape[semantic.FieldLesson].(map[string]any)
	for _, field := range []string{semantic.FieldIntent, semantic.FieldLessonType, semantic.FieldSymptom, semantic.FieldRecommendedPath} {
		if lessonShape[field] == nil {
			t.Fatalf("lesson shape should include %s: %#v", field, lessonShape)
		}
	}
	fields := explanation[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldLesson, semantic.FieldIntent),
		semantic.ParameterPath(semantic.FieldLesson, semantic.FieldLessonType),
		semantic.ParameterPath(semantic.FieldLesson, semantic.FieldSymptom),
		semantic.ParameterPath(semantic.FieldLesson, semantic.FieldRecommendedPath),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
}

func TestInvokeIntentExplainRecommendationFeedbackIncludesPublicPayloadGuide(t *testing.T) {
	app := newTestApp(t)
	input := `{"contractVersion":"1.0","requestId":"req-intent-explain-recommendation-feedback","locale":"zh-CN","utterance":"解释推荐反馈参数","intent":"intent.explain","parameters":{"intent":"recommendation.feedback"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	explanation := response[semantic.FieldResult].(map[string]any)[semantic.FieldIntentExplanation].(map[string]any)
	guide := explanation[semantic.FieldPayloadGuide].(map[string]any)
	shape := guide[semantic.FieldPayloadShape].(map[string]any)
	for _, field := range []string{semantic.FieldHouseID, semantic.FieldRecommendationID, semantic.FieldFeedback, semantic.FieldCooldownHours} {
		if shape[field] == nil {
			t.Fatalf("payload shape should include %s: %#v", field, shape)
		}
	}
	fields := explanation[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldRecommendationID),
		semantic.ParameterPath(semantic.FieldFeedback),
		semantic.ParameterPath(semantic.FieldCooldownHours),
	} {
		if !containsAnyString(fields, field) {
			t.Fatalf("acceptedFields should include %s: %#v", field, fields)
		}
	}
}

func TestIntentExplainRejectsUnsupportedIntent(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"intent", "explain", "--intent", "behavior.execute", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), `unsupported intent "behavior.execute"`) {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestPublicHelpDoesNotAdvertiseLegacyLightingImportPayloads(t *testing.T) {
	tests := [][]string{
		{"help", "device"},
		{"help", "lighting"},
		{"help", "lighting", "import"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run(args, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			output := stdout.String()
			for _, forbidden := range []string{"\"rooms\":[{\"name\":\"客厅\",\"items\"", "HouseMeta", "/v1/meta/import", "deviceTempIdList", "\"typeId\"", "\"resId\"", "\"params\":{\"set\":{\"p\""} {
				if strings.Contains(output, forbidden) {
					t.Fatalf("help contains legacy payload marker %q: %s", forbidden, output)
				}
			}
		})
	}
}

func TestUnknownHelpTopicReturnsInvalidInput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"help", "missing-command"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), `unknown help topic "missing-command"`) {
		t.Fatalf("stderr = %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage:\n  yeelight-home <command> [flags]") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestInvokeRejectsUnknownIntent(t *testing.T) {
	input := `{"contractVersion":"1.0","requestId":"req-1","locale":"zh-CN","utterance":"测试","intent":"raw.api.call"}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid response json: %v", err)
	}
	if response["status"] != "not_supported" || response["traceId"] != "invoke-unsupported-intent" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["intent"] != "raw.api.call" || result["safeToRetry"] != false {
		t.Fatalf("result = %#v", result)
	}
	responseError := response["error"].(map[string]any)
	if responseError["code"] != "unsupported_intent" || !strings.Contains(responseError["message"].(string), "unsupported intent") {
		t.Fatalf("error = %#v", responseError)
	}
}

func TestInvokeRequiresStdinFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"invoke"}, strings.NewReader("{}"), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: yeelight-home invoke --stdin") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestInvokeAcceptsRuntimeContextFlags(t *testing.T) {
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "Bearer invoke-flag-secret")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer invoke-flag-secret" {
			t.Fatalf("authorization header not sourced from env token")
		}
		switch request.URL.Path {
		case "/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"house-flag","houseName":"Flag Home"}]}}`))
		case "/v1/house/house-flag/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"houseId":"house-flag","name":"Flag Home"}}`))
		default:
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL)

	input := `{"contractVersion":"1.0","requestId":"req-invoke-flags","locale":"zh-CN","utterance":"列出家庭","intent":"home.list","parameters":{}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"invoke", "--stdin", "--profile", "flag-profile", "--region", "dev", "--house-id", "house-flag"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid response json: %v", err)
	}
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	houses := result["houses"].([]any)
	first := houses[0].(map[string]any)
	if result["region"] != "dev" || result["source"] != "/v1/house/r/list" || first["id"] != "house-flag" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeRejectsUnknownFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"invoke", "--stdin", "--json"}, strings.NewReader("{}"), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: yeelight-home invoke --stdin") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestAuthStatusJSONDoesNotExposeToken(t *testing.T) {
	t.Setenv("YEELIGHT_HOME_AUTHENTICATED", "1")
	t.Setenv("YEELIGHT_HOME_PROFILE", "family-main")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "secret-token-value")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"auth", "status", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "secret-token-value") {
		t.Fatalf("stdout leaked token: %s", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["authenticated"] != true {
		t.Fatalf("authenticated = %v", response["authenticated"])
	}
	if response["profile"] != "family-main" {
		t.Fatalf("profile = %v", response["profile"])
	}
}

func TestAuthStatusDefaultPrintsHumanReadableStatus(t *testing.T) {
	t.Setenv("YEELIGHT_HOME_PROFILE", "family-main")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "secret-token-value")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"auth", "status"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	text := stdout.String()
	for _, expected := range []string{"Yeelight Home Auth", "Authenticated: true", "Profile: family-main", "Region: cn", "Token present: true", "Token source: env"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("auth status text missing %q: %s", expected, text)
		}
	}
	if strings.Contains(text, "secret-token-value") {
		t.Fatalf("auth status text leaked token: %s", text)
	}
}

func TestDoctorJSONReportsConfigAndAuthWarning(t *testing.T) {
	t.Setenv("YEELIGHT_HOME_DIR", "/tmp/yeelight-home-test")
	t.Setenv("YEELIGHT_HOME_PROFILE", "doctor-empty-profile")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := newTestApp(t).run([]string{"doctor", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "warning" {
		t.Fatalf("status = %v", response["status"])
	}
	if response["configDir"] != "/tmp/yeelight-home-test/config" {
		t.Fatalf("configDir = %v", response["configDir"])
	}
	migrations, ok := response["memoryMigrations"].(map[string]any)
	if !ok {
		t.Fatalf("memoryMigrations = %#v", response["memoryMigrations"])
	}
	if migrations["status"] != "available" {
		t.Fatalf("memory migration status = %v", migrations["status"])
	}
	install, ok := response["install"].(map[string]any)
	if !ok {
		t.Fatalf("install = %#v", response["install"])
	}
	if install["cli"] != "yeelight-home" || install["version"] != version {
		t.Fatalf("install = %#v", install)
	}
}

func TestDoctorDefaultPrintsHumanReadableDiagnostics(t *testing.T) {
	t.Setenv("YEELIGHT_HOME_DIR", "/tmp/yeelight-home-test")
	t.Setenv("YEELIGHT_HOME_PROFILE", "doctor-empty-profile")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := newTestApp(t).run([]string{"doctor"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	text := stdout.String()
	for _, expected := range []string{
		"Yeelight Home Doctor",
		"Status: warning",
		"Authenticated: false",
		"Profile: doctor-empty-profile",
		"House ID: (not selected)",
		"Runtime version: " + version,
		"Warnings:",
		"auth_required",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("doctor text missing %q: %s", expected, text)
		}
	}
	if strings.Contains(text, "accessToken") || strings.Contains(text, "token-secret") {
		t.Fatalf("doctor text leaked token-like data: %s", text)
	}
}

func TestDoctorTextPrintsInstallRemediations(t *testing.T) {
	var stdout bytes.Buffer
	code := writeDoctorText(&stdout, map[string]any{
		"status":        "warning",
		"authenticated": false,
		"profile":       "default",
		"region":        "cn",
		"homeDir":       "/tmp/yeelight-home",
		"install": map[string]any{
			"version":            "0.1.6",
			"executable":         "/tmp/yeelight-home",
			"pathLookup":         "/opt/homebrew/bin/yeelight-home",
			"pathLookupResolved": "/opt/homebrew/lib/node_modules/yeelight-home/bin/yeelight-home.js",
			"npmWrapperResolved": "/opt/homebrew/lib/node_modules/yeelight-home/bin/yeelight-home.js",
			"packageManagers": map[string]any{
				"npm": map[string]any{"available": true, "installed": true, "version": "0.1.4"},
				"homebrew": map[string]any{
					"available": true,
					"installed": false,
					"formula":   map[string]any{"installed": false},
					"cask":      map[string]any{"installed": true, "version": "0.1.5"},
				},
			},
			"warnings":     []string{"path_lookup_uses_npm_wrapper"},
			"remediations": []string{"Upgrade the npm wrapper with `npm install -g yeelight-home@latest`, then restart the shell or Skill host."},
			"latest": map[string]any{
				"checked": true,
				"channels": map[string]any{
					"githubRelease": map[string]any{"ok": true, "version": "0.1.6"},
					"npm":           map[string]any{"ok": true, "version": "0.1.6"},
					"homebrew":      map[string]any{"ok": true, "version": "0.1.6"},
					"homebrewCask":  map[string]any{"ok": true, "version": "0.1.6"},
				},
			},
		},
	})
	if code != exitOK {
		t.Fatalf("exit code = %d", code)
	}
	text := stdout.String()
	for _, expected := range []string{
		"Suggested fixes:",
		"Public latest:",
		"githubRelease: ok=true version=0.1.6",
		"npm: ok=true version=0.1.6",
		"homebrewCask: ok=true version=0.1.6",
		"Install source summary:",
		"PATH channel: npm wrapper",
		"Running through npm wrapper: true",
		"npm global version: 0.1.4",
		"Homebrew cask version: 0.1.5",
		"cask: installed=true version=0.1.5",
		"npm install -g yeelight-home@latest",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("doctor text missing %q: %s", expected, text)
		}
	}
}

func TestAuthLoginQRNoWaitPrintsPayloadWithoutToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-nowait-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
	}

	code := app.run([]string{"auth", "login", "--qr", "--json", "--no-wait", "--region", "dev", "--device", "f82441000001"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["payload"] != "cli&F8:24:41:00:00:01&qr-nowait-1" {
		t.Fatalf("payload = %v", response["payload"])
	}
	if strings.Contains(stdout.String(), "token") {
		t.Fatalf("stdout leaked token-like data: %s", stdout.String())
	}
}

func TestAuthLoginQRUsesStableProfileDevice(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	client := &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-nowait-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
	}
	app.qrClient = client

	code := app.run([]string{"auth", "login", "--qr", "--json", "--no-wait", "--region", "dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("first login exit code = %d, stderr = %s", code, stderr.String())
	}
	firstDevice := client.createDevices[0]
	if firstDevice == "" || firstDevice == "F8:24:41:00:00:01" {
		t.Fatalf("first device = %q", firstDevice)
	}

	stdout.Reset()
	stderr.Reset()
	client.created = auth.QRInfo{QRCodeID: "qr-nowait-2", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()}
	code = app.run([]string{"auth", "login", "--qr", "--json", "--no-wait", "--region", "dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("second login exit code = %d, stderr = %s", code, stderr.String())
	}
	if client.createDevices[1] != firstDevice {
		t.Fatalf("second device = %q, want stable %q", client.createDevices[1], firstDevice)
	}
}

func TestAuthLoginQRExplicitDeviceOverridesStableProfileDevice(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	client := &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-nowait-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
	}
	app.qrClient = client

	code := app.run([]string{"auth", "login", "--qr", "--json", "--no-wait", "--region", "dev", "--device", "f82441010203"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("login exit code = %d, stderr = %s", code, stderr.String())
	}
	if client.createDevices[0] != "F8:24:41:01:02:03" {
		t.Fatalf("create device = %q", client.createDevices[0])
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil {
		t.Fatalf("Load metadata error: %v", err)
	}
	if !ok || metadata.QRDevice != "F8:24:41:01:02:03" {
		t.Fatalf("metadata = %#v ok=%v", metadata, ok)
	}
}

func TestAuthLoginQRNoWaitPlainTextPrintsTerminalQRCode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-nowait-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
	}

	code := app.run([]string{"auth", "login", "--qr", "--no-wait", "--region", "dev", "--device", "f82441000001"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "██") {
		t.Fatalf("expected terminal QR output, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Payload: cli&F8:24:41:00:00:01&qr-nowait-1") {
		t.Fatalf("expected payload fallback, got %s", stdout.String())
	}
}

func TestAuthLoginQRPlainTextPrintsQRCodeBeforePollingCompletes(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-login-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{
			QRCodeID: "qr-login-1",
			Status:   "LOGIN",
			Token:    auth.QRToken{AccessToken: "token-qr-secret-123456"},
		}},
	}
	app.sleep = func(context.Context, time.Duration) error {
		if !strings.Contains(stdout.String(), "Payload: cli&") || !strings.Contains(stdout.String(), "&qr-login-1") {
			t.Fatalf("expected QR prompt before polling, got %s", stdout.String())
		}
		return nil
	}

	code := app.run([]string{"auth", "login", "--qr", "--region", "dev", "--poll-interval-ms", "1", "--timeout-ms", "1000"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
}

func TestAuthLoginQRSavesTokenAndMetadataWithoutLeakingToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-login-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{
			QRCodeID: "qr-login-1",
			Status:   "LOGIN",
			Token:    auth.QRToken{AccessToken: "token-qr-secret-123456", ClientID: "client-qr-123456"},
			Source:   `dali:{"houseId":"house-qr-123456"}`,
		}},
	}

	code := app.run([]string{"auth", "login", "--qr", "--json", "--region", "dev", "--poll-interval-ms", "1", "--timeout-ms", "1000"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-qr-secret-123456") || strings.Contains(stderr.String(), "token-qr-secret-123456") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	record, ok, err := app.tokenStore.Load("default")
	if err != nil {
		t.Fatalf("Load token error: %v", err)
	}
	if !ok || record.AccessToken != "Bearer token-qr-secret-123456" {
		t.Fatalf("record = %#v ok=%v", record, ok)
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil {
		t.Fatalf("Load metadata error: %v", err)
	}
	if !ok || metadata.Region != "dev" || metadata.ClientID != "client-qr-123456" || metadata.HouseID != "house-qr-123456" {
		t.Fatalf("metadata = %#v ok=%v", metadata, ok)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	credentials, ok := response["credentials"].(map[string]any)
	if !ok {
		t.Fatalf("credentials = %#v", response["credentials"])
	}
	if credentials["accessTokenPresent"] != true || credentials["houseId"] != "house-qr-123456" {
		t.Fatalf("credentials = %#v", credentials)
	}
	if _, ok := credentials["clientId"]; ok {
		t.Fatalf("credentials exposed clientId: %#v", credentials)
	}
}

func TestAuthLoginQRThenHomeListUsesStoredTokenAndFallbackWithoutHouseID(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var calls []string
	var requestBodies []string
	var gotAuthorization []string
	var gotClientID []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		gotAuthorization = append(gotAuthorization, request.Header.Get("Authorization"))
		gotClientID = append(gotClientID, request.Header.Get("Client-Id"))
		if request.Body != nil {
			body, _ := io.ReadAll(request.Body)
			requestBodies = append(requestBodies, string(body))
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[]}}`))
		case "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"house-after-qr","houseName":"扫码后家庭"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-login-home-list-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{
			QRCodeID: "qr-login-home-list-1",
			Status:   "LOGIN",
			Token:    auth.QRToken{AccessToken: "token-qr-home-list-secret", ClientID: "client-qr-home-list"},
		}},
	}

	code := app.run([]string{"auth", "login", "--qr", "--json", "--region", "dev", "--poll-interval-ms", "1", "--timeout-ms", "1000"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("auth login exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-qr-home-list-secret") || strings.Contains(stderr.String(), "token-qr-home-list-secret") {
		t.Fatalf("auth login leaked token: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = app.run([]string{"home", "list", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("home list exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-qr-home-list-secret") || strings.Contains(stderr.String(), "token-qr-home-list-secret") {
		t.Fatalf("home list leaked token: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if strings.Join(calls, "\n") != "POST /apis/iot/v1/house/r/all\nPOST /apis/iot/v1/house/r/list" {
		t.Fatalf("calls = %#v", calls)
	}
	for index, authorization := range gotAuthorization {
		if authorization != "Bearer token-qr-home-list-secret" {
			t.Fatalf("authorization[%d] = %q", index, authorization)
		}
	}
	for index, clientID := range gotClientID {
		if clientID != "client-qr-home-list" {
			t.Fatalf("clientID[%d] = %q", index, clientID)
		}
	}
	if strings.Contains(strings.Join(requestBodies, "\n"), "houseId") {
		t.Fatalf("home list should not send houseId when profile has no selected home: %#v", requestBodies)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid home list json: %v", err)
	}
	houses := response["houses"].([]any)
	first := houses[0].(map[string]any)
	if response["region"] != "dev" || response["houseId"] != "" || response["houseCount"] != float64(1) || response["source"] != "/v1/house/r/list" {
		t.Fatalf("response = %#v", response)
	}
	if first["id"] != "house-after-qr" || first["name"] != "扫码后家庭" {
		t.Fatalf("houses = %#v", houses)
	}
}

func TestAPISmokeUsesEnvCredentialsAndDoesNotExposeToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var requestBodies []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Body != nil {
			body, _ := io.ReadAll(request.Body)
			requestBodies = append(requestBodies, string(body))
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_, _ = writer.Write([]byte(`{"code":"200","data":{"nickname":"测试用户"}}`))
		case "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[]}}`))
		case "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"house-1","name":"默认家庭"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "token-smoke-secret-123456")
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-smoke-123456"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"api", "smoke", "--json", "--region", "dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-smoke-secret-123456") || strings.Contains(stderr.String(), "token-smoke-secret-123456") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if strings.Contains(strings.Join(requestBodies, "\n"), "houseId") {
		t.Fatalf("api smoke should not require or send houseId when unset: %#v", requestBodies)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["accountOk"] != true || response["houseListOk"] != true || response["houseCount"] != float64(1) {
		t.Fatalf("response = %#v", response)
	}
	if response["houseListSource"] != "/v1/house/r/list" || response["houseListApiCalls"] != float64(2) {
		t.Fatalf("response = %#v", response)
	}
}

func TestAPISmokeDefaultPrintsHumanReadableSummary(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_, _ = writer.Write([]byte(`{"code":"200","data":{"nickname":"测试用户"}}`))
		case "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"houseList":[{"id":"house-1","name":"默认家庭"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "token-smoke-secret-123456")
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"api", "smoke", "--region", "dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	text := stdout.String()
	for _, expected := range []string{"Yeelight Home API Smoke", "Region: dev", "Account: ok", "Home list: ok", "House count: 1", "Home list source: /v1/house/r/all"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("api smoke text missing %q: %s", expected, text)
		}
	}
	if strings.Contains(text, "token-smoke-secret-123456") {
		t.Fatalf("api smoke text leaked token: %s", text)
	}
}

func TestHomeListJSONAccountScopedEmptyListIncludesDiagnostics(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/all", "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[]}}`))
		default:
			http.NotFound(writer, request)
			return
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "token-home-list-secret")
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"home", "list", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-home-list-secret") || strings.Contains(stderr.String(), "token-home-list-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["houseCount"] != float64(0) || response[semantic.FieldAPICalls] != float64(2) || response["source"] != "/v1/house/r/all+/v1/house/r/list" {
		t.Fatalf("response = %#v", response)
	}
	warnings, ok := response["warnings"].([]any)
	if !ok || len(warnings) != 1 || warnings[0] != "empty_account_home_list" {
		t.Fatalf("warnings = %#v", response["warnings"])
	}
}

func TestHomeListJSONFallsBackWhenStatsHomeListIsEmpty(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[]}}`))
		case "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"house-fallback","houseName":"回退家庭"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "token-home-list-secret")
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"home", "list", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Join(calls, "\n") != "POST /apis/iot/v1/house/r/all\nPOST /apis/iot/v1/house/r/list" {
		t.Fatalf("calls = %#v", calls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	houses := response["houses"].([]any)
	first := houses[0].(map[string]any)
	if response["houseCount"] != float64(1) || response["source"] != "/v1/house/r/list" || first["name"] != "回退家庭" {
		t.Fatalf("response = %#v", response)
	}
}

func TestHomeListJSONReturnsAllAccountHomesWithSelectedHouse(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var calls []string
	var houseHeaderCalls []string
	var gotBizTypes []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		if request.Header.Get("houseId") != "" || request.Header.Get("house-id") != "" {
			houseHeaderCalls = append(houseHeaderCalls, request.Method+" "+request.URL.Path)
		}
		gotBizTypes = append(gotBizTypes, request.Header.Get("bizType"))
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"house-selected","houseName":"当前家庭"},{"houseId":"house-other","houseName":"另一个家"}]}}`))
		case "/apis/iot/v1/house/r/list", "/apis/iot/v1/house/house-selected/r/info":
			t.Fatalf("home list should use account all-list without selected house fallback: %s", request.URL.Path)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "token-home-list-secret")
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", HouseID: "house-selected"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"home", "list", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Join(calls, "\n") != "POST /apis/iot/v1/house/r/all" {
		t.Fatalf("calls = %#v", calls)
	}
	if len(houseHeaderCalls) != 0 {
		t.Fatalf("home list must not send selected house headers: %#v", houseHeaderCalls)
	}
	for index, bizType := range gotBizTypes {
		if bizType != "" {
			t.Fatalf("bizType[%d] = %q, want backend default PRO", index, bizType)
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	houses := response["houses"].([]any)
	if response["houseId"] != "" || response["selectedHouseId"] != "house-selected" || response["houseCount"] != float64(2) || len(houses) != 2 {
		t.Fatalf("response = %#v", response)
	}
}

func TestHomeListJSONIgnoresSelectedHouseWhenAccountListsAreEmpty(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/all", "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[]}}`))
		case "/apis/iot/v1/house/house-selected/r/info":
			t.Fatalf("home list must not fall back to selected house detail")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "token-home-list-secret")
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", HouseID: "house-selected"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"home", "list", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-home-list-secret") || strings.Contains(stderr.String(), "token-home-list-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if strings.Join(calls, "\n") != "POST /apis/iot/v1/house/r/all\nPOST /apis/iot/v1/house/r/list" {
		t.Fatalf("calls = %#v", calls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	houses := response["houses"].([]any)
	if len(houses) != 0 || response["houseCount"] != float64(0) || response[semantic.FieldAPICalls] != float64(2) || response["source"] != "/v1/house/r/all+/v1/house/r/list" || response["houseId"] != "" || response["selectedHouseId"] != "house-selected" {
		t.Fatalf("response = %#v", response)
	}
	warnings, ok := response["warnings"].([]any)
	if !ok || len(warnings) != 1 || warnings[0] != "empty_account_home_list" {
		t.Fatalf("warnings = %#v", response["warnings"])
	}
	if !strings.Contains(response["rawShape"].(string), "/v1/house/r/all:") || strings.Contains(response["rawShape"].(string), "home.detail.get:") {
		t.Fatalf("rawShape = %s", response["rawShape"])
	}
}

func TestHomeListUnauthorizedReturnsActionableAuthError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "token-home-list-secret")
	app := newTestApp(t)

	code := app.run([]string{"home", "list", "--json", "--region", "dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "authorization failed") || !strings.Contains(stderr.String(), "auth login --qr --region dev") {
		t.Fatalf("stderr = %s", stderr.String())
	}
	if strings.Contains(stderr.String(), "token-home-list-secret") {
		t.Fatalf("stderr leaked token: %s", stderr.String())
	}
}

func TestAuthStatusReadsStoredCredentialMetadata(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-1", HouseID: "house-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"auth", "status", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-secret") {
		t.Fatalf("status leaked token: %s", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["authenticated"] != true || response["profile"] != "default" || response["houseId"] != "house-1" {
		t.Fatalf("response = %#v", response)
	}
	if _, ok := response["clientId"]; ok {
		t.Fatalf("status exposed clientId: %#v", response)
	}
}

func TestTokenOnlyProfileSupportsAuthStatusDoctorAndRuntimeContext(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)

	code := app.run([]string{"auth", "token", "set", "--token", "Bearer token-only-secret", "--profile", "token-only", "--region", "cn", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("auth token set exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-only-secret") || strings.Contains(stderr.String(), "token-only-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	var tokenSet map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &tokenSet); err != nil {
		t.Fatalf("invalid token set json: %v", err)
	}
	if tokenSet["tokenPresent"] != true || tokenSet["region"] != "cn" || tokenSet["houseId"] != "" {
		t.Fatalf("token set response = %#v", tokenSet)
	}
	metadata, ok, err := app.metadataStore.Load("token-only")
	if err != nil {
		t.Fatalf("Load metadata error: %v", err)
	}
	if !ok || metadata.Region != "cn" || metadata.HouseID != "" {
		t.Fatalf("metadata = %#v ok=%v", metadata, ok)
	}
	context, err := app.resolveRuntimeContext(cliFlags{values: map[string]string{"profile": "token-only"}})
	if err != nil {
		t.Fatalf("resolveRuntimeContext error: %v", err)
	}
	if !context.TokenPresent || context.AccessToken != "Bearer token-only-secret" || context.HouseID != "" {
		t.Fatalf("context = %#v", context)
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"auth", "status", "--profile", "token-only", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("auth status exit code = %d, stderr = %s", code, stderr.String())
	}
	var status map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatalf("invalid auth status json: %v", err)
	}
	if status["authenticated"] != true || status["houseId"] != "" {
		t.Fatalf("auth status = %#v", status)
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"doctor", "--profile", "token-only", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("doctor exit code = %d, stderr = %s", code, stderr.String())
	}
	var doctor map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &doctor); err != nil {
		t.Fatalf("invalid doctor json: %v", err)
	}
	if doctor["status"] != "ok" || doctor["authenticated"] != true || doctor["houseId"] != "" {
		t.Fatalf("doctor = %#v", doctor)
	}
}

func TestAuthTokenSetCanReadSecretFromStdin(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)

	code := app.run([]string{"auth", "token", "set", "--stdin", "--profile", "stdin-token", "--region", "dev", "--json"}, strings.NewReader("Bearer stdin-secret\n"), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("auth token set --stdin exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "stdin-secret") || strings.Contains(stderr.String(), "stdin-secret") {
		t.Fatalf("stdin token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	context, err := app.resolveRuntimeContext(cliFlags{values: map[string]string{"profile": "stdin-token"}})
	if err != nil {
		t.Fatalf("resolveRuntimeContext error: %v", err)
	}
	if context.AccessToken != "Bearer stdin-secret" || context.Region != "dev" || context.HouseID != "" {
		t.Fatalf("context = %#v", context)
	}
}

func TestAuthTokenSetRejectsTokenAndStdinTogether(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)

	code := app.run([]string{"auth", "token", "set", "--token", "Bearer flag-secret", "--stdin", "--profile", "ambiguous"}, strings.NewReader("Bearer stdin-secret\n"), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("auth token set ambiguous exit code = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestRuntimeContextPrecedenceFlagsEnvProfileDefaults(t *testing.T) {
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-store"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "cn", ClientID: "client-profile", HouseID: "house-profile"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	t.Setenv("YEELIGHT_CLOUD_REGION", "sg")
	t.Setenv("YEELIGHT_HOME_HOUSE_ID", "house-env")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "Bearer token-env")

	context, err := app.resolveRuntimeContext(cliFlags{values: map[string]string{
		"region":   "eu",
		"house-id": "house-flag",
	}})
	if err != nil {
		t.Fatalf("resolveRuntimeContext error: %v", err)
	}
	if context.Region != "eu" || context.ClientID != "client-profile" || context.HouseID != "house-flag" || context.AccessToken != "Bearer token-env" {
		t.Fatalf("context = %#v", context)
	}
}

func TestRuntimeContextDefaultsToCNRegion(t *testing.T) {
	app := newTestApp(t)
	context, err := app.resolveRuntimeContext(cliFlags{values: map[string]string{}})
	if err != nil {
		t.Fatalf("resolveRuntimeContext error: %v", err)
	}
	if context.Region != "cn" {
		t.Fatalf("Region = %q", context.Region)
	}
}

func TestAuthTokenSetDoesNotWriteTokenToProfileMetadata(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)

	code := app.run([]string{"auth", "token", "set", "--token", "Bearer token-manual-secret", "--profile", "manual", "--region", "cn", "--house-id", "house-1", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-manual-secret") {
		t.Fatalf("stdout leaked token: %s", stdout.String())
	}
	data, err := os.ReadFile(app.metadataStore.Path())
	if err != nil {
		t.Fatalf("ReadFile metadata error: %v", err)
	}
	if strings.Contains(string(data), "token-manual-secret") || strings.Contains(string(data), "accessToken") {
		t.Fatalf("metadata leaked token: %s", string(data))
	}
	record, ok, err := app.tokenStore.Load("manual")
	if err != nil {
		t.Fatalf("Load token error: %v", err)
	}
	if !ok || record.AccessToken != "Bearer token-manual-secret" {
		t.Fatalf("record = %#v ok=%v", record, ok)
	}
}

func TestConfigSetAndHomeSelectUpdateProfileMetadata(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)

	code := app.run([]string{"config", "set", "--profile", "family", "--region", "sg", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("config set exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"home", "select", "--profile", "family", "--house-id", "house-selected", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("home select exit code = %d, stderr = %s", code, stderr.String())
	}
	metadata, ok, err := app.metadataStore.Load("family")
	if err != nil {
		t.Fatalf("Load metadata error: %v", err)
	}
	if !ok || metadata.Region != "sg" || metadata.ClientID != "" || metadata.HouseID != "house-selected" {
		t.Fatalf("metadata = %#v ok=%v", metadata, ok)
	}
}

func TestProfileUseSetsActiveProfile(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)

	code := app.run([]string{"profile", "use", "--profile", "family", "--region", "cn", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	context, err := app.resolveRuntimeContext(cliFlags{values: map[string]string{}})
	if err != nil {
		t.Fatalf("resolveRuntimeContext error: %v", err)
	}
	if context.Profile != "family" || context.Region != "cn" {
		t.Fatalf("context = %#v", context)
	}
}

func TestMutatingConfigCommandsUseActiveProfileUnlessOverridden(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "cn-main", Region: "cn", HouseID: "cn-house"}); err != nil {
		t.Fatalf("Save cn metadata error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "dev-main", Region: "dev", HouseID: "dev-house"}); err != nil {
		t.Fatalf("Save dev metadata error: %v", err)
	}
	if err := app.metadataStore.SetActiveProfile("dev-main"); err != nil {
		t.Fatalf("SetActiveProfile error: %v", err)
	}

	code := app.run([]string{"config", "set", "--region", "sg", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("config set exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"home", "select", "--house-id", "sg-house", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("home select exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"auth", "token", "set", "--token", "Bearer active-profile-secret", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("auth token set exit code = %d, stderr = %s", code, stderr.String())
	}

	devMetadata, _, err := app.metadataStore.Load("dev-main")
	if err != nil {
		t.Fatalf("Load dev metadata error: %v", err)
	}
	cnMetadata, _, err := app.metadataStore.Load("cn-main")
	if err != nil {
		t.Fatalf("Load cn metadata error: %v", err)
	}
	if devMetadata.Region != "sg" || devMetadata.HouseID != "sg-house" {
		t.Fatalf("dev metadata = %#v", devMetadata)
	}
	if cnMetadata.Region != "cn" || cnMetadata.HouseID != "cn-house" {
		t.Fatalf("cn metadata = %#v", cnMetadata)
	}
	if _, ok, err := app.tokenStore.Load("dev-main"); err != nil || !ok {
		t.Fatalf("active profile token ok=%v err=%v", ok, err)
	}
	if _, ok, err := app.tokenStore.Load("cn-main"); err != nil || ok {
		t.Fatalf("cn profile token ok=%v err=%v", ok, err)
	}

	t.Setenv("YEELIGHT_HOME_PROFILE", "cn-main")
	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"config", "set", "--region", "eu", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("env override config set exit code = %d, stderr = %s", code, stderr.String())
	}
	cnMetadata, _, err = app.metadataStore.Load("cn-main")
	if err != nil {
		t.Fatalf("Reload cn metadata error: %v", err)
	}
	if cnMetadata.Region != "eu" || cnMetadata.HouseID != "cn-house" {
		t.Fatalf("cn metadata after env override = %#v", cnMetadata)
	}
}

func newTestApp(t *testing.T) *app {
	t.Helper()
	tokenStore := credential.NewMemoryStore()
	app := &app{
		tokenStore:    tokenStore,
		metadataStore: credential.NewFileMetadataStore(t.TempDir() + "/profiles.json"),
		memoryStore:   storage.NewJSONStore(t.TempDir() + "/memory.json"),
		topologyCache: newTopologyCache(t.TempDir() + "/topology.json"),
		sleep:         func(context.Context, time.Duration) error { return nil },
	}
	return app
}

type testQRClient struct {
	created       auth.QRInfo
	checked       []auth.QRInfo
	checkCalls    int
	createDevices []string
}

func (client *testQRClient) Create(_ context.Context, device string) (auth.QRInfo, error) {
	client.createDevices = append(client.createDevices, device)
	return client.created, nil
}

func (client *testQRClient) Check(context.Context, string) (auth.QRInfo, error) {
	index := client.checkCalls
	client.checkCalls++
	if index >= len(client.checked) {
		return client.checked[len(client.checked)-1], nil
	}
	return client.checked[index], nil
}
