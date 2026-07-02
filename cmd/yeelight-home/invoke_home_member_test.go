package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeHomeMemberConfigureExecutesDirectly(t *testing.T) {
	var gotCalls []string
	var writeBody map[string]any
	roleUpdated := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"uid":9000,"nickname":"业主"}}`))
		case "/apis/iot/v1/house/r/memberlistV2":
			if roleUpdated {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[{"uid":1001,"nickname":"成员","userRole":2}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[{"uid":1001,"nickname":"成员","userRole":0}]}}`))
		case "/apis/iot/v1/house/w/updateUserRole":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode role body: %v", err)
			}
			roleUpdated = true
			_, _ = writer.Write([]byte(`{"success":true}`))
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
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-member-execute" {
		t.Fatalf("response = %#v", response)
	}
	if writeBody["memberId"] != float64(1001) || writeBody["userRole"] != float64(2) {
		t.Fatalf("writeBody = %#v calls=%#v", writeBody, gotCalls)
	}
}

func TestInvokeHomeMemberConfigureResolvesUniqueMemberName(t *testing.T) {
	var writeBody map[string]any
	roleUpdated := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"uid":9000,"nickname":"业主"}}`))
		case "/apis/iot/v1/house/r/memberlistV2":
			if roleUpdated {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[{"uid":1001,"nickname":"张三","userRole":2},{"uid":1002,"nickname":"李四","userRole":0}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"memberList":[{"uid":1001,"nickname":"张三","userRole":0},{"uid":1002,"nickname":"李四","userRole":0}]}}`))
		case "/apis/iot/v1/house/w/updateUserRole":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode role body: %v", err)
			}
			roleUpdated = true
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-member-name-secret", "client-member-name-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-member-configure-name","locale":"zh-CN","utterance":"把张三设为管理员","intent":"home.member.configure","parameters":{"houseId":"200171","memberName":"张三","userRole":"admin"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-member-execute" {
		t.Fatalf("response = %#v", response)
	}
	if writeBody["memberId"] != float64(1001) || writeBody["userRole"] != float64(2) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if strings.Contains(stdout.String(), "1001") {
		t.Fatalf("public response leaked raw member id: %s", stdout.String())
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
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if acceptBody["toUid"] != float64(9000) {
		t.Fatalf("acceptBody = %#v", acceptBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-member-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "home.member.accept_share" || result["verifiedBy"] != "home.summary" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeHomeMemberRemoveExecutesDirectlyAfterCallerConfirmation(t *testing.T) {
	memberVisible := true
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
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

	input := `{"contractVersion":"1.0","requestId":"req-member-remove","locale":"zh-CN","utterance":"移除成员","intent":"home.member.remove","parameters":{"houseId":"200171","memberId":"1001","confirmed":true}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-member-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "home.member.remove" || result["risk"] != "R3" || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
	removeCalls := 0
	for _, call := range gotCalls {
		if call == "POST /apis/iot/v1/house/w/remove" {
			removeCalls++
		}
	}
	if removeCalls != 1 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}
