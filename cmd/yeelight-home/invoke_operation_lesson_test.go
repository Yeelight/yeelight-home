package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/storage"
)

func TestInvokeOperationLessonRecordAndListWithoutToken(t *testing.T) {
	app := newTestApp(t)
	input := `{"contractVersion":"1.0","requestId":"req-lesson-record","locale":"zh-CN","utterance":"记录 scene.update 参数经验","intent":"operation.lesson.record","parameters":{"houseId":"house-1","lesson":{"intent":"scene.update","lessonType":"parameter_shape","symptom":"invalid_scene_update_payload","cause":"details 内部 action rows 不能靠 acceptedFields 猜","recommendedPath":"先调用 scene.detail.get 获取 editablePayload/updateShape，再 read-modify-send 完整 details","avoid":"不要凭空拼 details/params","fallbackIntent":"scene.create","evidence":"Runtime 返回 invalid_scene_update_payload"}}}`

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("record exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid record response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "operation-lesson-record-local" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	lesson := result["operationLesson"].(map[string]any)
	if lesson["intent"] != "scene.update" || result["created"] != true {
		t.Fatalf("lesson result = %#v", result)
	}

	stdout.Reset()
	stderr.Reset()
	listInput := `{"contractVersion":"1.0","requestId":"req-lesson-list","locale":"zh-CN","utterance":"查询 scene.update 实操经验","intent":"operation.lesson.list","parameters":{"houseId":"house-1","intent":"scene.update","limit":5}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(listInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("list exit code = %d, stderr = %s", code, stderr.String())
	}
	var listResponse map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &listResponse); err != nil {
		t.Fatalf("invalid list response: %v", err)
	}
	items := listResponse["result"].(map[string]any)["operationLessons"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %#v", items)
	}
}

func TestInvokeOperationLessonRecordMergesDuplicate(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-lesson-secret", "client-lesson-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-lesson-record","locale":"zh-CN","utterance":"记录最快路径","intent":"operation.lesson.record","parameters":{"lesson":{"intent":"light.power.set","lessonType":"fast_path","symptom":"用户开灯时多轮查实体太慢","recommendedPath":"直接调用 light.power.set，带自然语言目标和 roomName","evidence":"控制灯光实测慢在多轮查找"}}}`
	for attempt := 0; attempt < 2; attempt++ {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("record attempt %d exit code = %d, stderr = %s", attempt, code, stderr.String())
		}
	}
	lessons, err := app.memoryStore.ListOperationLessons(newLessonFilter("default", "dev", "", "light.power.set"))
	if err != nil {
		t.Fatalf("ListOperationLessons error: %v", err)
	}
	if len(lessons) != 1 || lessons[0].HitCount != 2 {
		t.Fatalf("lessons = %#v", lessons)
	}
}

func TestInvokeOperationLessonRecordDryRunDoesNotPersist(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-lesson-dry-run-secret", "client-lesson-dry-run-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-lesson-dry-run","locale":"zh-CN","utterance":"先预览一下要记录的经验","intent":"operation.lesson.record","options":{"dryRun":true},"parameters":{"houseId":"house-1","lesson":{"intent":"panel.button.type.get","lessonType":"parameter_shape","symptom":"developer dry-run payload validation only","recommendedPath":"Use panel.get returned button row type for panel.button.type.get.","evidence":"dry-run validation"}}}`

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("dry-run record exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid dry-run response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "operation-lesson-record-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["dryRun"] != true {
		t.Fatalf("result = %#v", result)
	}

	lessons, err := app.memoryStore.ListOperationLessons(newLessonFilter("default", "dev", "house-1", "panel.button.type.get"))
	if err != nil {
		t.Fatalf("ListOperationLessons error: %v", err)
	}
	if len(lessons) != 0 {
		t.Fatalf("dry-run should not persist lessons: %#v", lessons)
	}
}

func TestInvokeOperationLessonRecordRequiresStructuredFields(t *testing.T) {
	app := newTestApp(t)
	input := `{"contractVersion":"1.0","requestId":"req-lesson-invalid","locale":"zh-CN","utterance":"记录一个经验","intent":"operation.lesson.record","parameters":{"lesson":{"intent":"scene.update"}}}`

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("record exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	if response["status"] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}
}

func newLessonFilter(profile string, region string, houseID string, intent string) storage.OperationLessonFilter {
	return storage.OperationLessonFilter{Profile: profile, Region: region, HouseID: houseID, Intent: intent}
}
