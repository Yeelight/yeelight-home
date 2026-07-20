package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/credential"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func TestConfigSetPersistsLanguageAndLANMode(t *testing.T) {
	app := newTestApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{
		"config", "set", "--profile", "lan-home", "--language", "zh_CN.UTF-8",
		"--control-mode", "local-preferred", "--gateway-ip", "192.168.1.2", "--json",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("config set code = %d, stderr = %s", code, stderr.String())
	}
	metadata, ok, err := app.metadataStore.Load("lan-home")
	if err != nil || !ok {
		t.Fatalf("Load metadata ok=%v err=%v", ok, err)
	}
	if metadata.Language != "zh-CN" || metadata.ControlMode != controlModeLocalPreferred || metadata.GatewayIP != "192.168.1.2" || metadata.LANEndpoint != "http://192.168.1.2:18080/mcp" {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestConfigGetResolvesLANPrecedence(t *testing.T) {
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{
		Profile: "default", Language: "zh-CN", ControlMode: controlModeLocalPreferred,
		GatewayIP: "192.168.1.2", LANEndpoint: "http://192.168.1.2:18080/mcp",
	}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	t.Setenv("YEELIGHT_HOME_LANGUAGE", "en_GB.UTF-8")
	t.Setenv("YEELIGHT_HOME_CONTROL_MODE", controlModeLocalOnly)
	t.Setenv("YEELIGHT_HOME_GATEWAY_IP", "10.0.0.5")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"config", "get", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("config get code = %d, stderr = %s", code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal result error: %v", err)
	}
	if result[semantic.FieldLanguage] != "en-US" || result[semantic.FieldControlMode] != controlModeLocalOnly || result[semantic.FieldLANEndpoint] != "http://10.0.0.5:18080/mcp" {
		t.Fatalf("result = %#v", result)
	}
}

func TestConfigRejectsInvalidLANSettings(t *testing.T) {
	tests := [][]string{
		{"config", "set", "--language", "fr-FR"},
		{"config", "set", "--control-mode", "nearby"},
		{"config", "set", "--control-mode", "local-only"},
		{"config", "set", "--control-mode", "local-only", "--lan-endpoint", "https://8.8.8.8/mcp"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			app := newTestApp(t)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if code := app.run(args, strings.NewReader(""), &stdout, &stderr); code != exitInvalidInput {
				t.Fatalf("code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestConfigUnsetLANFields(t *testing.T) {
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{
		Profile: "default", Language: "en-US", ControlMode: controlModeLocalOnly,
		GatewayIP: "192.168.1.2", LANEndpoint: "http://192.168.1.2:18080/mcp",
	}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"config", "unset", "--language", "--control-mode", "--gateway-ip", "--lan-endpoint", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("config unset code = %d, stderr = %s", code, stderr.String())
	}
	metadata, _, err := app.metadataStore.Load("default")
	if err != nil {
		t.Fatalf("Load metadata error: %v", err)
	}
	if metadata.Language != "" || metadata.ControlMode != "" || metadata.GatewayIP != "" || metadata.LANEndpoint != "" {
		t.Fatalf("metadata = %#v", metadata)
	}
}
