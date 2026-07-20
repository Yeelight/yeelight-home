package main

import (
	"testing"

	"github.com/yeelight/yeelight-home/internal/contract"
)

func TestEnglishLightClarificationMessage(t *testing.T) {
	response := lightControlClarificationResponse(contract.Request{
		ContractVersion: contract.Version,
		RequestID:       "req-en-light",
		Locale:          "en-US",
		Utterance:       "Turn on a light",
		Intent:          "light.power.set",
	}, "missing_target", entityGetTarget{}, nil, 0)
	if response.UserMessage != "Choose the light, room, area, or group to control and provide the desired value." {
		t.Fatalf("UserMessage = %q", response.UserMessage)
	}
}

func TestEnglishSceneClarificationMessage(t *testing.T) {
	response := sceneExecuteClarificationResponse(contract.Request{
		ContractVersion: contract.Version,
		RequestID:       "req-en-scene",
		Locale:          "en-US",
		Utterance:       "Run a scene",
		Intent:          "scene.execute",
	}, "missing_target", entityGetTarget{}, nil, 0)
	if response.UserMessage != "Choose the scene to run." {
		t.Fatalf("UserMessage = %q", response.UserMessage)
	}
}
