package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/plan"
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

func TestInvokeMemoryRememberExtractsExplicitUtterancePreference(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	planInput := `{"contractVersion":"1.0","requestId":"req-memory-nl-plan","locale":"zh-CN","utterance":"记住以后卧室默认柔和暖光，不要太亮","intent":"memory.remember","parameters":{"houseId":"house-1"}}`

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(planInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("plan exit code = %d, stderr = %s", code, stderr.String())
	}
	var planResponse map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &planResponse); err != nil {
		t.Fatalf("invalid plan response: %v", err)
	}
	if planResponse["status"] != "confirmation_required" {
		t.Fatalf("planResponse = %#v", planResponse)
	}
	planID := planResponse["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok {
		t.Fatalf("Load plan ok=%v err=%v", ok, err)
	}
	if record.Payload["preferenceType"] != "brightness" || record.Payload["scopeRef"] != "卧室" {
		t.Fatalf("payload = %#v", record.Payload)
	}

	stdout.Reset()
	stderr.Reset()
	commitInput := `{"contractVersion":"1.0","requestId":"req-memory-nl-commit","locale":"zh-CN","utterance":"确认记住","intent":"plan.commit","parameters":{"planId":"` + planID + `"}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("commit exit code = %d, stderr = %s", code, stderr.String())
	}

	list, err := app.memoryStore.ListPreferences("default", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(list) != 1 || list[0].ScopeRef != "卧室" || !strings.Contains(list[0].PreferenceValue, "柔和暖光") {
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

func TestInvokeMemoryDefaultsLearningAndRecommendationEnabled(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-memory-default","locale":"zh-CN","utterance":"查看记忆","intent":"memory.list","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("memory list exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid memory response: %v", err)
	}
	memory := response["memory"].(map[string]any)
	if memory["learningEnabled"] != true || memory["paused"] != false {
		t.Fatalf("memory = %#v", memory)
	}

	consent, ok, err := app.memoryStore.Consent("default", "house-1")
	if err != nil || !ok {
		t.Fatalf("Consent ok=%v err=%v", ok, err)
	}
	if !consent.LearningEnabled || consent.Paused {
		t.Fatalf("consent = %#v", consent)
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-recommendation-default","locale":"zh-CN","utterance":"有什么建议","intent":"recommendation.list","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("recommendation list exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"success"`) {
		t.Fatalf("recommendation stdout = %s", stdout.String())
	}
}

func TestInvokeMemoryListRecoversCommittedMemoryPlanWithExistingConsent(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	consentAt := time.Now().Add(-time.Hour).Unix()
	if err := app.memoryStore.SetConsent(storage.ConsentRecord{
		Profile:         "default",
		HouseID:         "house-1",
		ConsentVersion:  memoryConsentVersion,
		LearningEnabled: true,
		UpdatedAt:       consentAt,
	}); err != nil {
		t.Fatalf("SetConsent error: %v", err)
	}
	record, err := plan.NewRecord("default", "dev", "house-1", "memory.remember", "req-recover", "记住偏好 brightness=35", map[string]any{
		"scopeType":       "room",
		"scopeRef":        "卧室",
		"preferenceType":  "brightness",
		"preferenceValue": "35",
		"kind":            "explicit",
		"evidence":        "用户明确说明",
	}, nil, time.Now().Add(-30*time.Minute), time.Minute)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	record.Status = plan.StatusCommitted
	record.CommittedAt = time.Now().Add(-20 * time.Minute).Unix()
	if err := app.planStore.Save(record); err != nil {
		t.Fatalf("Save plan error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-memory-recover","locale":"zh-CN","utterance":"查看记忆","intent":"memory.list","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("memory list exit code = %d, stderr = %s", code, stderr.String())
	}
	list, err := app.memoryStore.ListPreferences("default", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(list) != 1 || list[0].ID != record.ID || list[0].PreferenceValue != "35" {
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

func TestPendingPlanCompactionRecoversCommittedMemoryPlanBeforePrune(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	now := time.Unix(20_000, 0)
	app.planStore = app.planStore.
		WithClock(func() time.Time { return now }).
		WithRetention(time.Hour, time.Hour, 2)
	app.configureMemoryPlanRecovery()
	if err := app.memoryStore.SetConsent(storage.ConsentRecord{
		Profile:         "default",
		HouseID:         "house-1",
		ConsentVersion:  memoryConsentVersion,
		LearningEnabled: true,
		UpdatedAt:       now.Add(-3 * time.Hour).Unix(),
	}); err != nil {
		t.Fatalf("SetConsent error: %v", err)
	}
	oldMemory, err := plan.NewRecord("default", "dev", "house-1", "memory.remember", "req-old-memory", "记住偏好 brightness=35", map[string]any{
		"scopeType":       "room",
		"scopeRef":        "卧室",
		"preferenceType":  "brightness",
		"preferenceValue": "35",
	}, nil, now.Add(-3*time.Hour), time.Minute)
	if err != nil {
		t.Fatalf("old memory plan error: %v", err)
	}
	oldMemory.Status = plan.StatusCommitted
	oldMemory.CommittedAt = now.Add(-2 * time.Hour).Unix()
	recentOne, err := plan.NewRecord("default", "dev", "house-1", "room.create", "req-recent-1", "最近计划1", map[string]any{"name": "one"}, nil, now, time.Minute)
	if err != nil {
		t.Fatalf("recent one error: %v", err)
	}
	recentOne.Status = plan.StatusCommitted
	recentOne.CommittedAt = now.Add(-time.Minute).Unix()
	recentTwo, err := plan.NewRecord("default", "dev", "house-1", "room.create", "req-recent-2", "最近计划2", map[string]any{"name": "two"}, nil, now, time.Minute)
	if err != nil {
		t.Fatalf("recent two error: %v", err)
	}
	recentTwo.Status = plan.StatusCommitted
	recentTwo.CommittedAt = now.Unix()

	for _, record := range []plan.Record{oldMemory, recentOne, recentTwo} {
		if err := app.planStore.Save(record); err != nil {
			t.Fatalf("Save %s error: %v", record.SourceRequestID, err)
		}
	}
	if _, ok, err := app.planStore.Load(oldMemory.ID); err != nil || ok {
		t.Fatalf("old memory plan should be compacted, ok=%v err=%v", ok, err)
	}
	list, err := app.memoryStore.ListPreferences("default", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(list) != 1 || list[0].ID != oldMemory.ID || list[0].PreferenceValue != "35" {
		t.Fatalf("recovered preferences = %#v", list)
	}
}

func TestInvokeMemoryListDoesNotRecoverForgottenOldPlan(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	record, err := plan.NewRecord("default", "dev", "house-1", "memory.remember", "req-old", "记住偏好 brightness=35", map[string]any{
		"scopeType":       "room",
		"scopeRef":        "卧室",
		"preferenceType":  "brightness",
		"preferenceValue": "35",
	}, nil, time.Now().Add(-2*time.Hour), time.Minute)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	record.Status = plan.StatusCommitted
	record.CommittedAt = time.Now().Add(-90 * time.Minute).Unix()
	if err := app.planStore.Save(record); err != nil {
		t.Fatalf("Save plan error: %v", err)
	}
	if err := app.memoryStore.SetConsent(storage.ConsentRecord{
		Profile:         "default",
		HouseID:         "house-1",
		ConsentVersion:  memoryConsentVersion,
		LearningEnabled: true,
		UpdatedAt:       time.Now().Unix(),
	}); err != nil {
		t.Fatalf("SetConsent error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-memory-no-recover","locale":"zh-CN","utterance":"查看记忆","intent":"memory.list","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("memory list exit code = %d, stderr = %s", code, stderr.String())
	}
	list, err := app.memoryStore.ListPreferences("default", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}

func TestInvokeMemorySignalCreatesImplicitRecommendationAfterRepeatedCorrection(t *testing.T) {
	stateReadCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"dev-1","name":"客厅主灯","type":"device","roomId":"客厅"}]}}`))
		case "/apis/iot/v1/controll/device/dev-1/r/properties/l":
			stateReadCount++
			if stateReadCount%2 == 1 {
				_, _ = writer.Write([]byte(`{"success":true,"data":42}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":32}`))
		case "/apis/iot/v1/controll/device/2/dev-1/w/properties/l/adjust":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-signal-1","locale":"zh-CN","utterance":"客厅太亮了，调暗点","intent":"light.brightness.adjust","targets":[{"entityType":"device","id":"dev-1"}],"parameters":{"houseId":"house-1","delta":-10}}`
	for index := 0; index < 2; index++ {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("signal run %d exit code = %d, stderr = %s stdout=%s", index, code, stderr.String(), stdout.String())
		}
	}
	signals, err := app.memoryStore.ListInteractionSignals("default", "house-1")
	if err != nil {
		t.Fatalf("ListInteractionSignals error: %v", err)
	}
	if len(signals) != 1 || signals[0].Count != 2 || signals[0].SignalType != "preference_hint" {
		t.Fatalf("signals = %#v", signals)
	}
	recommendations, err := app.memoryStore.ListRecommendations("default", "house-1", time.Now().Unix(), 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(recommendations) != 1 || recommendations[0].Type != "implicit_candidate" {
		t.Fatalf("recommendations = %#v", recommendations)
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
