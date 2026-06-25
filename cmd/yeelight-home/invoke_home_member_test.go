package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeHomeMemberConfigureCreatesR2PendingPlan(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"uid":9000,"nickname":"业主"}}`))
		case "/apis/iot/v1/house/r/memberlistV2":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[{"uid":1001,"nickname":"成员","userRole":0}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-member-secret", "client-member-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-member-configure","locale":"zh-CN","utterance":"把成员设为管理员","intent":"home.member.configure","parameters":{"houseId":"200171","memberId":"1001","userRole":2}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/updateUserRole") {
			t.Fatalf("configure should not write before commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	if confirmation["risk"] != "R2" || confirmation["approvalRequired"] == true {
		t.Fatalf("confirmation = %#v", confirmation)
	}
	record, ok, err := app.planStore.Load(confirmation["planId"].(string))
	if err != nil || !ok || record.Intent != "home.member.configure" || record.Risk != "R2" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokeHomeMemberAcceptShareUsesCurrentUserAndCommits(t *testing.T) {
	var gotCalls []string
	var acceptBody map[string]any
	homeVisible := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"uid":9000,"nickname":"接收人"}}`))
		case "/apis/iot/v1/share/w/acceptbarcodeshare":
			if err := json.NewDecoder(request.Body).Decode(&acceptBody); err != nil {
				t.Fatalf("decode accept body: %v", err)
			}
			homeVisible = true
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":7001,"resId":200171,"toUid":9000,"status":1}}`))
		case "/apis/iot/v1/house/r/list":
			if !homeVisible {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"200171","houseName":"分享家庭"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-member-accept-secret", "client-member-accept-1", "")

	input := `{"contractVersion":"1.0","requestId":"req-member-accept","locale":"zh-CN","utterance":"接受家庭分享","intent":"home.member.accept_share","parameters":{"houseId":"200171","shareId":"7001","createTime":1710000000,"toUid":"1111"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("plan exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/acceptbarcodeshare") {
			t.Fatalf("accept share should not write before commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	record, ok, err := app.planStore.Load(confirmation["planId"].(string))
	if err != nil || !ok {
		t.Fatalf("load plan ok=%v err=%v", ok, err)
	}
	if record.Intent != "home.member.accept_share" || record.Risk != "R2" || record.HouseID != "200171" {
		t.Fatalf("record = %#v", record)
	}
	if record.Payload["toUid"] != float64(9000) && record.Payload["toUid"] != 9000 {
		t.Fatalf("Runtime must use current account uid, payload = %#v", record.Payload)
	}

	stdout.Reset()
	stderr.Reset()
	commitInput := `{"contractVersion":"1.0","requestId":"req-member-accept-commit","locale":"zh-CN","utterance":"确认","intent":"plan.commit","parameters":{"planId":"` + record.ID + `","toUid":"2222"}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("commit exit code = %d, stderr = %s", code, stderr.String())
	}
	if acceptBody["toUid"] != float64(9000) {
		t.Fatalf("acceptBody = %#v", acceptBody)
	}
	response = decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-member-commit" {
		t.Fatalf("commit response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "home.member.accept_share" || result["verifiedBy"] != "home.summary" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeHomeMemberRemoveRequiresApprovalBeforeCommit(t *testing.T) {
	memberVisible := true
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"uid":9000,"nickname":"业主"}}`))
		case "/apis/iot/v1/house/r/memberlistV2":
			if memberVisible {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[{"uid":1001,"nickname":"成员","userRole":0}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[]}}`))
		case "/apis/iot/v1/house/w/remove":
			memberVisible = false
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-member-remove-secret", "client-member-remove-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-member-remove","locale":"zh-CN","utterance":"移除成员","intent":"home.member.remove","parameters":{"houseId":"200171","memberId":"1001"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("plan exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	confirmation := response["confirmation"].(map[string]any)
	if response["status"] != "confirmation_required" || confirmation["risk"] != "R3" || confirmation["approvalRequired"] != true {
		t.Fatalf("response = %#v", response)
	}
	planID := confirmation["planId"].(string)
	commitInput := `{"contractVersion":"1.0","requestId":"req-member-remove-commit","locale":"zh-CN","utterance":"确认","intent":"plan.commit","parameters":{"planId":"` + planID + `"}}`

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("blocked exit code = %d, stderr = %s", code, stderr.String())
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
		t.Fatalf("approve json: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("commit exit code = %d, stderr = %s", code, stderr.String())
	}
	response = decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-member-commit" {
		t.Fatalf("commit response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "home.member.remove" || result["risk"] != "R3" || result["localApproval"] != true || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}
