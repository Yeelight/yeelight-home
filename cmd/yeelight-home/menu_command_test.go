package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/credential"
)

func TestExplicitMenuCanExitWithoutChangingState(t *testing.T) {
	app := newTestApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"menu"}, strings.NewReader("0\n"), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 || !strings.Contains(stdout.String(), "Yeelight Home") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestTTYWithoutArgumentsStartsMenuWhileNonTTYPrintsHelp(t *testing.T) {
	app := newTestApp(t)
	app.terminal = func(_ io.Reader) bool { return true }
	var interactive bytes.Buffer
	if code := app.run(nil, strings.NewReader("0\n"), &interactive, &bytes.Buffer{}); code != exitOK || !strings.Contains(interactive.String(), "Yeelight Home") {
		t.Fatalf("interactive code=%d output=%s", code, interactive.String())
	}
	app.terminal = func(_ io.Reader) bool { return false }
	var nonInteractive bytes.Buffer
	if code := app.run(nil, strings.NewReader(""), &nonInteractive, &bytes.Buffer{}); code != exitOK || !strings.Contains(nonInteractive.String(), "Usage:") {
		t.Fatalf("non-interactive code=%d output=%s", code, nonInteractive.String())
	}
}

func TestMenuSelectsDeviceByNameAndReadsLocalState(t *testing.T) {
	gateway := newRuntimeGateway(t, false)
	defer gateway.Close()
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalOnly, gateway.URL+"/mcp")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"menu"}, strings.NewReader("3\n1\n1\n0\n"), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Living Light") || !strings.Contains(stdout.String(), "家庭网关读取本地设备状态") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestMenuConfirmsDeviceLightControl(t *testing.T) {
	gateway := newRuntimeGateway(t, false)
	defer gateway.Close()
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalOnly, gateway.URL+"/mcp")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"menu"}, strings.NewReader("3\n1\n2\ny\n0\n"), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 || !strings.Contains(stdout.String(), "局域网内完成操作") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestMenuUsesSavedEnglishLanguage(t *testing.T) {
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", Language: "en-US"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	var stdout bytes.Buffer
	code := app.run([]string{"menu"}, strings.NewReader("0\n"), &stdout, &bytes.Buffer{})
	if code != exitOK || !strings.Contains(stdout.String(), "Choose home") {
		t.Fatalf("code=%d stdout=%s", code, stdout.String())
	}
}
