package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeDeviceRemoveRequiresLocalApprovalBeforeCommit(t *testing.T) {
	deviceVisible := true
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			if deviceVisible {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401398"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/50018330/w/info":
			deviceVisible = false
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-r3-secret", "client-r3-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-remove-plan","locale":"zh-CN","utterance":"删除主灯","intent":"device.remove","parameters":{"houseId":"200171","deviceId":"50018330"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("plan exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/info") {
			t.Fatalf("device.remove should not write before approval and commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	if confirmation["risk"] != "R3" || confirmation["approvalRequired"] != true || confirmation["approvalChallenge"] == "" {
		t.Fatalf("confirmation = %#v", confirmation)
	}
	planID := confirmation["planId"].(string)

	stdout.Reset()
	stderr.Reset()
	commitInput := `{"contractVersion":"1.0","requestId":"req-device-remove-commit-blocked","locale":"zh-CN","utterance":"确认执行","intent":"plan.commit","parameters":{"planId":"` + planID + `"}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("blocked commit exit code = %d, stderr = %s", code, stderr.String())
	}
	response = decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "blocked" || response["error"].(map[string]any)["code"] != "local_approval_required" {
		t.Fatalf("blocked response = %#v", response)
	}

	stdout.Reset()
	stderr.Reset()
	challenge := confirmation["approvalChallenge"].(string)
	code = app.run([]string{"approve", "--json", "--plan-id", planID, "--challenge", challenge}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("approve exit code = %d, stderr = %s", code, stderr.String())
	}
	var approveResponse map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &approveResponse); err != nil {
		t.Fatalf("invalid approve json: %v", err)
	}
	if approveResponse["status"] != "approved" || approveResponse["risk"] != "R3" {
		t.Fatalf("approveResponse = %#v", approveResponse)
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("approved commit exit code = %d, stderr = %s", code, stderr.String())
	}
	response = decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "destructive-delete-commit" {
		t.Fatalf("commit response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "device.remove" || result["risk"] != "R3" || result["localApproval"] != true || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}
