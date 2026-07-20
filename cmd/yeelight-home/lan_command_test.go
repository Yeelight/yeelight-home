package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLANInspectListsLiveGatewayTools(t *testing.T) {
	gateway := newRuntimeGateway(t, false)
	defer gateway.Close()
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalPreferred, gateway.URL+"/mcp")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"lan", "inspect", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if result["ok"] != true || int(result["toolCount"].(float64)) != 4 {
		t.Fatalf("result = %#v", result)
	}
}

func TestLANCallPreviewsBeforeExplicitExecution(t *testing.T) {
	gateway := newRuntimeGateway(t, false)
	defer gateway.Close()
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalPreferred, gateway.URL+"/mcp")
	arguments := `{"nodeId":"device-1","propertyName":"l","value":80}`

	var preview bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"lan", "call", "control_node", "--args-json", arguments}, strings.NewReader(""), &preview, &stderr)
	if code != exitOK || !strings.Contains(preview.String(), `"executed":false`) {
		t.Fatalf("code=%d preview=%s stderr=%s", code, preview.String(), stderr.String())
	}

	var executed bytes.Buffer
	stderr.Reset()
	code = app.run([]string{"lan", "call", "control_node", "--args-json", arguments, "--yes"}, strings.NewReader(""), &executed, &stderr)
	if code != exitOK || stderr.Len() != 0 || !strings.Contains(executed.String(), `"name":"control_node"`) {
		t.Fatalf("code=%d executed=%s stderr=%s", code, executed.String(), stderr.String())
	}
}

func TestLANCallRejectsUnknownTool(t *testing.T) {
	gateway := newRuntimeGateway(t, false)
	defer gateway.Close()
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalPreferred, gateway.URL+"/mcp")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"lan", "call", "missing", "--yes"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput || !strings.Contains(stderr.String(), "does not expose tool") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}
