package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/storage"
)

func TestInvokeMemoryRememberUsesPendingPlanAndCommit(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	planInput := `{"contractVersion":"1.0","requestId":"req-memory-plan","locale":"zh-CN","utterance":"记住我喜欢客厅亮度 45","intent":"memory.remember","parameters":{"houseId":"house-1","scopeType":"room","scopeRef":"客厅","preferenceType":"brightness","preferenceValue":"45","evidence":"用户明确说明"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(planInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("plan exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-memory-secret") || strings.Contains(stderr.String(), "token-memory-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	var planResponse map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &planResponse); err != nil {
		t.Fatalf("invalid plan response: %v", err)
	}
	if planResponse["status"] != "confirmation_required" || planResponse["traceId"] != "memory-pending-plan-created" {
		t.Fatalf("planResponse = %#v", planResponse)
	}
	planID := planResponse["confirmation"].(map[string]any)["planId"].(string)

	stdout.Reset()
	stderr.Reset()
	commitInput := `{"contractVersion":"1.0","requestId":"req-memory-commit","locale":"zh-CN","utterance":"确认记住","intent":"plan.commit","parameters":{"planId":"` + planID + `","preferenceValue":"ignored"}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("commit exit code = %d, stderr = %s", code, stderr.String())
	}
	var commitResponse map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &commitResponse); err != nil {
		t.Fatalf("invalid commit response: %v", err)
	}
	if commitResponse["status"] != "success" || commitResponse["traceId"] != "memory-remember-commit" {
		t.Fatalf("commitResponse = %#v", commitResponse)
	}
	memory := commitResponse["memory"].(map[string]any)
	if memory["preferenceValue"] != "45" || memory["kind"] != "explicit" {
		t.Fatalf("memory = %#v", memory)
	}

	list, err := app.memoryStore.ListPreferences("default", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(list) != 1 || list[0].PreferenceValue != "45" {
		t.Fatalf("list = %#v", list)
	}
	recommendations, err := app.memoryStore.ListRecommendations("default", "house-1", time.Now().Unix(), 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(recommendations) != 1 || recommendations[0].Type != "preference_based" {
		t.Fatalf("recommendations = %#v", recommendations)
	}
}

func TestInvokeMemoryListPauseAndForget(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	if err := app.memoryStore.SavePreference(storage.PreferenceRecord{
		ID:              "pref-1",
		Profile:         "default",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "brightness",
		PreferenceValue: "45",
		Kind:            "explicit",
		UpdatedAt:       123,
	}); err != nil {
		t.Fatalf("SavePreference error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-memory-list","locale":"zh-CN","utterance":"查看记忆","intent":"memory.list","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("list exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid list response: %v", err)
	}
	items := response["memory"].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("response = %#v", response)
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-memory-pause","locale":"zh-CN","utterance":"暂停记忆","intent":"memory.pause","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("pause exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"paused":true`) {
		t.Fatalf("pause stdout = %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-memory-forget","locale":"zh-CN","utterance":"删除记忆","intent":"memory.forget","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("forget exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"deletedCount":1`) {
		t.Fatalf("forget stdout = %s", stdout.String())
	}
	list, err := app.memoryStore.ListPreferences("default", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list after forget = %#v", list)
	}
}

func TestInvokeRecommendationListReturnsAtMostOneItem(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	now := time.Now().Unix()
	for _, record := range []storage.RecommendationRecord{
		{ID: "rec-1", Profile: "default", HouseID: "house-1", Type: "scene", Explanation: "晚上常调暗客厅灯", Evidence: "脱敏 evidence 3 次", Status: "pending", CreatedAt: now, UpdatedAt: now},
		{ID: "rec-2", Profile: "default", HouseID: "house-1", Type: "automation", Explanation: "睡前常关灯", Evidence: "脱敏 evidence 2 次", Status: "pending", CreatedAt: now, UpdatedAt: now},
	} {
		if err := app.memoryStore.SaveRecommendation(record); err != nil {
			t.Fatalf("SaveRecommendation error: %v", err)
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-rec-list","locale":"zh-CN","utterance":"有什么建议","intent":"recommendation.list","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("recommendation exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid recommendation response: %v", err)
	}
	recommendation := response["recommendation"].(map[string]any)
	items := recommendation["items"].([]any)
	if len(items) != 1 || recommendation["sessionLimit"] != float64(1) {
		t.Fatalf("recommendation = %#v", recommendation)
	}
}

func TestInvokeRecommendationListGeneratesFromSavedPreference(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	if err := app.memoryStore.SavePreference(storage.PreferenceRecord{
		ID:              "pref-brightness-living",
		Profile:         "default",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "brightness",
		PreferenceValue: "45",
		Kind:            "explicit",
		Evidence:        "用户明确说明",
		CreatedAt:       123,
		UpdatedAt:       123,
	}); err != nil {
		t.Fatalf("SavePreference error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-rec-generated","locale":"zh-CN","utterance":"有什么建议","intent":"recommendation.list","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("recommendation exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid recommendation response: %v", err)
	}
	items := response["recommendation"].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("recommendation response = %#v", response)
	}
	item := items[0].(map[string]any)
	if item["id"] != "pref-pref-brightness-living" || item["type"] != "preference_based" || !strings.Contains(item["explanation"].(string), "brightness=45") {
		t.Fatalf("recommendation item = %#v", item)
	}
}

func TestInvokeRecommendationListDoesNotReviveRejectedPreferenceRecommendation(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	now := time.Now().Unix()
	if err := app.memoryStore.SavePreference(storage.PreferenceRecord{
		ID:              "pref-brightness-living",
		Profile:         "default",
		HouseID:         "house-1",
		ScopeType:       "room",
		ScopeRef:        "客厅",
		PreferenceType:  "brightness",
		PreferenceValue: "45",
		Kind:            "explicit",
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("SavePreference error: %v", err)
	}
	if err := app.memoryStore.SaveRecommendation(storage.RecommendationRecord{
		ID:          "pref-pref-brightness-living",
		Profile:     "default",
		HouseID:     "house-1",
		Type:        "preference_based",
		Explanation: "旧推荐",
		Evidence:    "用户已拒绝",
		Status:      "rejected",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("SaveRecommendation error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-rec-not-revived","locale":"zh-CN","utterance":"有什么建议","intent":"recommendation.list","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("recommendation exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid recommendation response: %v", err)
	}
	items := response["recommendation"].(map[string]any)["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("recommendation response = %#v", response)
	}
}

func TestInvokeRecommendationFeedbackRejectsLocalRecommendation(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	now := time.Now().Unix()
	if err := app.memoryStore.SaveRecommendation(storage.RecommendationRecord{
		ID:          "rec-1",
		Profile:     "default",
		HouseID:     "house-1",
		Type:        "scene",
		Explanation: "晚上常调暗客厅灯",
		Evidence:    "脱敏 evidence 3 次",
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("SaveRecommendation error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	input := `{"contractVersion":"1.0","requestId":"req-rec-reject","locale":"zh-CN","utterance":"不采纳这个建议","intent":"recommendation.feedback","parameters":{"houseId":"house-1","recommendationId":"rec-1","feedback":"reject"}}`
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("feedback exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid feedback response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "recommendation-feedback-local" {
		t.Fatalf("response = %#v", response)
	}
	recommendation := response["recommendation"].(map[string]any)
	if recommendation["status"] != "rejected" || recommendation["feedbackRecorded"] != true {
		t.Fatalf("recommendation = %#v", recommendation)
	}
	list, err := app.memoryStore.ListRecommendations("default", "house-1", time.Now().Unix(), 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}

func TestInvokeRecommendationFeedbackCooldownKeepsRecommendationPending(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	now := time.Now().Unix()
	if err := app.memoryStore.SaveRecommendation(storage.RecommendationRecord{
		ID:          "rec-1",
		Profile:     "default",
		HouseID:     "house-1",
		Type:        "scene",
		Explanation: "晚上常调暗客厅灯",
		Evidence:    "脱敏 evidence 3 次",
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("SaveRecommendation error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	input := `{"contractVersion":"1.0","requestId":"req-rec-cooldown","locale":"zh-CN","utterance":"稍后再提醒","intent":"recommendation.feedback","parameters":{"houseId":"house-1","recommendationId":"rec-1","feedback":"cooldown","cooldownHours":2}}`
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("feedback exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid feedback response: %v", err)
	}
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
	recommendation := response["recommendation"].(map[string]any)
	if recommendation["status"] != "pending" || recommendation["cooldownUntil"] == float64(0) {
		t.Fatalf("recommendation = %#v", recommendation)
	}
	list, err := app.memoryStore.ListRecommendations("default", "house-1", time.Now().Unix(), 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}
