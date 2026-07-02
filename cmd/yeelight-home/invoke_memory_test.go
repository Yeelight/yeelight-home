package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
	"github.com/yeelight/yeelight-home/internal/storage"
)

func TestInvokeMemoryRememberWritesDirectlyAndDeduplicates(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-memory-direct","locale":"zh-CN","utterance":"记住我喜欢客厅亮度 45","intent":"memory.remember","parameters":{"houseId":"house-1","scopeType":"room","scopeRef":"客厅","preferenceType":"brightness","preferenceValue":"45","evidence":"用户明确说明"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("remember exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-memory-secret") || strings.Contains(stderr.String(), "token-memory-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid remember response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "memory-remember-local" {
		t.Fatalf("response = %#v", response)
	}
	memory := response["memory"].(map[string]any)
	if memory["preferenceValue"] != "45" || memory["kind"] != "explicit" || memory["created"] != true {
		t.Fatalf("memory = %#v", memory)
	}

	stdout.Reset()
	stderr.Reset()
	duplicate := `{"contractVersion":"1.0","requestId":"req-memory-duplicate","locale":"zh-CN","utterance":"再记一下，我喜欢客厅亮度 45","intent":"memory.remember","parameters":{"houseId":"house-1","scopeType":"room","scopeRef":"客厅","preferenceType":"brightness","preferenceValue":"45","evidence":"用户重复说明"}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(duplicate), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("duplicate exit code = %d, stderr = %s", code, stderr.String())
	}
	var duplicateResponse map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &duplicateResponse); err != nil {
		t.Fatalf("invalid duplicate response: %v", err)
	}
	duplicateMemory := duplicateResponse["memory"].(map[string]any)
	if duplicateMemory["created"] != false || duplicateMemory["merged"] != true {
		t.Fatalf("duplicate memory = %#v", duplicateMemory)
	}
	list, err := app.memoryStore.ListPreferences("default", "dev", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(list) != 1 || list[0].PreferenceValue != "45" {
		t.Fatalf("list = %#v", list)
	}
	recommendations, err := app.memoryStore.ListRecommendations("default", "dev", "house-1", time.Now().Unix(), 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(recommendations) != 0 {
		t.Fatalf("recommendations = %#v", recommendations)
	}
}

func TestInvokeMemoryRememberRequiresStructuredPreference(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-memory-nl","locale":"zh-CN","utterance":"记住以后卧室默认柔和暖光，不要太亮","intent":"memory.remember","parameters":{"houseId":"house-1"}}`

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("remember exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid remember response: %v", err)
	}
	if response["status"] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}

	list, err := app.memoryStore.ListPreferences("default", "dev", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}

func TestInvokeMemoryRememberStoresSkillStructuredBatchPreferences(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-memory-batch","locale":"zh-CN","utterance":"记住我喜欢喜欢浪漫的色调还有高端奢华","intent":"memory.remember","parameters":{"houseId":"house-1","preferences":[{"scopeType":"home","preferenceType":"ambience","preferenceValue":"prefer_romantic_warm","evidence":"用户明确要求记住喜欢浪漫色调"},{"scopeType":"home","preferenceType":"product_preference","preferenceValue":"prefer_premium_luxury","evidence":"用户明确要求记住高端奢华产品定位"}]}}`

	for attempt := 0; attempt < 2; attempt++ {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("remember attempt %d exit code = %d, stderr = %s", attempt, code, stderr.String())
		}
		var response map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("invalid remember response: %v", err)
		}
		if response["status"] != "success" {
			t.Fatalf("response = %#v", response)
		}
		memory := response["memory"].(map[string]any)
		if memory["count"] != float64(2) {
			t.Fatalf("memory = %#v", memory)
		}
		if attempt == 0 && (memory["createdCount"] != float64(2) || memory["mergedCount"] != float64(0)) {
			t.Fatalf("first memory = %#v", memory)
		}
		if attempt == 1 && (memory["createdCount"] != float64(0) || memory["mergedCount"] != float64(2)) {
			t.Fatalf("second memory = %#v", memory)
		}
	}

	list, err := app.memoryStore.ListPreferences("default", "dev", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %#v", list)
	}
	valuesByType := map[string]string{}
	for _, item := range list {
		if item.ScopeType != "home" || item.ScopeRef != "" {
			t.Fatalf("list = %#v", list)
		}
		valuesByType[item.PreferenceType] = item.PreferenceValue
	}
	if valuesByType["ambience"] != "prefer_romantic_warm" {
		t.Fatalf("ambience memory missing: %#v", list)
	}
	if valuesByType["product_preference"] != "prefer_premium_luxury" {
		t.Fatalf("product preference memory missing: %#v", list)
	}
}

func TestInvokeMemoryListPauseAndForget(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	if err := app.memoryStore.SavePreference(storage.PreferenceRecord{
		ID:              "pref-1",
		Profile:         "default",
		Region:          "dev",
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
	namespace := response["memory"].(map[string]any)["namespace"].(map[string]any)
	if namespace["profile"] != "default" || namespace["region"] != "dev" || namespace["houseId"] != "house-1" || namespace["dataType"] != "memory" {
		t.Fatalf("memory namespace = %#v", namespace)
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
	list, err := app.memoryStore.ListPreferences("default", "dev", "house-1")
	if err != nil {
		t.Fatalf("ListPreferences error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list after forget = %#v", list)
	}
}

func TestInvokeMemoryForgetDeletesOnlyRequestedIDs(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	for _, item := range []storage.PreferenceRecord{
		{ID: "pref-delete", Profile: "default", Region: "dev", HouseID: "house-1", ScopeType: "test", ScopeRef: "session-a", PreferenceType: "ambience", PreferenceValue: "romantic", Kind: "explicit", UpdatedAt: 123},
		{ID: "pref-keep", Profile: "default", Region: "dev", HouseID: "house-1", ScopeType: "home", PreferenceType: "ambience", PreferenceValue: "warm", Kind: "explicit", UpdatedAt: 124},
	} {
		if err := app.memoryStore.SavePreference(item); err != nil {
			t.Fatalf("SavePreference error: %v", err)
		}
	}
	for _, item := range []storage.RecommendationRecord{
		{ID: "rec-delete", Profile: "default", Region: "dev", HouseID: "house-1", Type: "lighting_design", Explanation: "delete me", Evidence: "test", Status: "pending", CreatedAt: 123, UpdatedAt: 123},
		{ID: "rec-keep", Profile: "default", Region: "dev", HouseID: "house-1", Type: "lighting_design", Explanation: "keep me", Evidence: "real", Status: "pending", CreatedAt: 124, UpdatedAt: 124},
	} {
		if err := app.memoryStore.SaveRecommendation(item); err != nil {
			t.Fatalf("SaveRecommendation error: %v", err)
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-memory-forget-selected","locale":"zh-CN","utterance":"只删除这次测试记忆","intent":"memory.forget","parameters":{"houseId":"house-1","preferenceIds":["pref-delete"],"recommendationIds":["rec-delete"]}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("forget selected exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"deletedCount":2`) {
		t.Fatalf("forget selected stdout = %s", stdout.String())
	}

	exported, err := app.memoryStore.Export("default", "dev", "house-1")
	if err != nil {
		t.Fatalf("Export error: %v", err)
	}
	preferences := exported[semantic.FieldPreferences].([]storage.PreferenceRecord)
	recommendations := exported[semantic.FieldRecommendations].([]storage.RecommendationRecord)
	if len(preferences) != 1 || preferences[0].ID != "pref-keep" {
		t.Fatalf("preferences after selected forget = %#v", preferences)
	}
	if len(recommendations) != 1 || recommendations[0].ID != "rec-keep" {
		t.Fatalf("recommendations after selected forget = %#v", recommendations)
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

	consent, ok, err := app.memoryStore.Consent("default", "dev", "house-1")
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

func TestInvokeMemorySignalDoesNotInferPreferenceOrRecommendation(t *testing.T) {
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
	signals, err := app.memoryStore.ListInteractionSignals("default", "dev", "house-1")
	if err != nil {
		t.Fatalf("ListInteractionSignals error: %v", err)
	}
	if len(signals) != 1 || signals[0].Count != 2 || signals[0].SignalType != "interaction" {
		t.Fatalf("signals = %#v", signals)
	}
	if signals[0].SignalKey != "light.brightness.adjust|interaction" {
		t.Fatalf("runtime should store only coarse interaction signal: %#v", signals)
	}
	if strings.Contains(signals[0].Evidence, "客厅") || strings.Contains(signals[0].Evidence, "太亮") || signals[0].Evidence != "intent=light.brightness.adjust; status=success" {
		t.Fatalf("runtime should not store user utterance as signal evidence: %#v", signals)
	}
	recommendations, err := app.memoryStore.ListRecommendations("default", "dev", "house-1", time.Now().Unix(), 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(recommendations) != 0 {
		t.Fatalf("recommendations = %#v", recommendations)
	}
}

func TestInvokeRecommendationListReturnsAtMostOneItem(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	now := time.Now().Unix()
	for _, record := range []storage.RecommendationRecord{
		{ID: "rec-1", Profile: "default", Region: "dev", HouseID: "house-1", Type: "scene", Explanation: "晚上常调暗客厅灯", Evidence: "脱敏 evidence 3 次", Status: "pending", Priority: 10, Confidence: "medium", CreatedAt: now, UpdatedAt: now},
		{ID: "rec-2", Profile: "default", Region: "dev", HouseID: "house-1", Type: "automation", Explanation: "睡前常关灯", Evidence: "脱敏 evidence 2 次", Status: "pending", Priority: 90, Confidence: "high", CreatedAt: now, UpdatedAt: now},
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
	if items[0].(map[string]any)["id"] != "rec-2" {
		t.Fatalf("recommendation should return highest ranked item first: %#v", recommendation)
	}
	namespace := recommendation["namespace"].(map[string]any)
	if namespace["profile"] != "default" || namespace["region"] != "dev" || namespace["houseId"] != "house-1" {
		t.Fatalf("recommendation namespace = %#v", namespace)
	}
}

func TestInvokeRecommendationRecordStoresSkillAuthoredCandidate(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	input := `{"contractVersion":"1.0","requestId":"req-rec-record","locale":"zh-CN","utterance":"记录一个建议","intent":"recommendation.record","parameters":{"houseId":"house-1","type":"automation","source":"ai_skill","targetIntent":"automation.create","scopeType":"room","scopeRef":"主卧","priority":80,"confidence":"high","explanation":"可以把浪漫暖光做成主卧晚间自动化。","evidence":"本地记忆 ambience=prefer_romantic_warm","actionHint":{"label":"创建主卧晚间自动化"},"parametersHint":{"roomName":"主卧","tone":"warm_romantic"}}}`
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("recommendation record exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid recommendation record response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "recommendation-record-local" {
		t.Fatalf("response = %#v", response)
	}
	recommendation := response["recommendation"].(map[string]any)
	item := recommendation["item"].(map[string]any)
	if item["type"] != "automation" || item["targetIntent"] != "automation.create" || item["scopeRef"] != "主卧" || item["created"] != true {
		t.Fatalf("item = %#v", item)
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-rec-list-recorded","locale":"zh-CN","utterance":"有什么建议","intent":"recommendation.list","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("recommendation list exit code = %d, stderr = %s", code, stderr.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid recommendation list response: %v", err)
	}
	items := response["recommendation"].(map[string]any)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["source"] != "ai_skill" {
		t.Fatalf("recommendation response = %#v", response)
	}
}

func TestInvokeRecommendationRecordRequiresStructuredFields(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	input := `{"contractVersion":"1.0","requestId":"req-rec-invalid","locale":"zh-CN","utterance":"给个建议","intent":"recommendation.record","parameters":{"houseId":"house-1","type":"automation","explanation":"可以创建自动化"}}`
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("recommendation record exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid recommendation response: %v", err)
	}
	if response["status"] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}
	list, err := app.memoryStore.ListRecommendations("default", "dev", "house-1", time.Now().Unix(), 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}

func TestInvokeRecommendationRecordMergesDuplicateCandidate(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-rec-merge","locale":"zh-CN","utterance":"记录建议","intent":"recommendation.record","parameters":{"houseId":"house-1","type":"automation","source":"ai_skill","targetIntent":"automation.create","scopeType":"room","scopeRef":"主卧","explanation":"可以把浪漫暖光做成主卧晚间自动化。","evidence":"第一次证据"}}`
	duplicate := `{"contractVersion":"1.0","requestId":"req-rec-merge-2","locale":"zh-CN","utterance":"重复记录建议","intent":"recommendation.record","parameters":{"houseId":"house-1","type":"automation","source":"ai_skill","targetIntent":"automation.create","scopeType":"room","scopeRef":"主卧","explanation":"可以把浪漫暖光做成主卧晚间自动化。","evidence":"第二次证据"}}`
	for index, raw := range []string{input, duplicate} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(raw), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("record %d exit code = %d, stderr = %s", index, code, stderr.String())
		}
		var response map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("invalid record %d response: %v", index, err)
		}
		item := response["recommendation"].(map[string]any)["item"].(map[string]any)
		if index == 0 && item["created"] != true {
			t.Fatalf("first item = %#v", item)
		}
		if index == 1 && (item["created"] != false || item["merged"] != true) {
			t.Fatalf("second item = %#v", item)
		}
	}
	list, err := app.memoryStore.ListRecommendations("default", "dev", "house-1", time.Now().Unix(), 10)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 1 || !strings.Contains(list[0].Evidence, "第一次证据") || !strings.Contains(list[0].Evidence, "第二次证据") {
		t.Fatalf("list = %#v", list)
	}
}

func TestInvokeRecommendationListDoesNotMaterializePreferenceAutomatically(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	now := time.Now().Unix()
	if err := app.memoryStore.SavePreference(storage.PreferenceRecord{
		ID:              "pref-brightness-living",
		Profile:         "default",
		Region:          "dev",
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
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-rec-not-generated","locale":"zh-CN","utterance":"有什么建议","intent":"recommendation.list","parameters":{"houseId":"house-1"}}`), &stdout, &stderr)
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
		Region:      "dev",
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
	list, err := app.memoryStore.ListRecommendations("default", "dev", "house-1", time.Now().Unix(), 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}

func TestInvokeRecommendationFeedbackHideDismissesLocalRecommendation(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	now := time.Now().Unix()
	if err := app.memoryStore.SaveRecommendation(storage.RecommendationRecord{
		ID:          "rec-1",
		Profile:     "default",
		Region:      "dev",
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
	input := `{"contractVersion":"1.0","requestId":"req-rec-hide","locale":"zh-CN","utterance":"这条建议不要再提醒我","intent":"recommendation.feedback","parameters":{"houseId":"house-1","recommendationId":"rec-1","feedback":"hide"}}`
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
	if recommendation["status"] != "dismissed" || recommendation["feedbackRecorded"] != true {
		t.Fatalf("recommendation = %#v", recommendation)
	}
	list, err := app.memoryStore.ListRecommendations("default", "dev", "house-1", time.Now().Unix(), 1)
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
		Region:      "dev",
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
	list, err := app.memoryStore.ListRecommendations("default", "dev", "house-1", time.Now().Unix(), 1)
	if err != nil {
		t.Fatalf("ListRecommendations error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}
