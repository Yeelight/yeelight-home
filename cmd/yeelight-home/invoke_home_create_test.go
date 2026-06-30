package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeHomeCreateDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-home-create-secret", "client-home-create-1", "")

	input := `{"contractVersion":"1.0","requestId":"req-home-create-plan","locale":"zh-CN","utterance":"创建一个叫新家的家庭","intent":"home.create","parameters":{"name":"新家","description":"常住房","icon":"home","areaCode":"CN-310000","areaName":"上海"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/create") {
			t.Fatalf("home.create dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)
	payloadPreview := preview["payloadPreview"].(map[string]any)
	if payloadPreview["scope"] != "account" || payloadPreview["name"] != "新家" {
		t.Fatalf("payloadPreview = %#v", payloadPreview)
	}
}

func TestInvokeHomeCreateExecutesDirectlyAndVerifies(t *testing.T) {
	var gotCalls []string
	listCalls := 0
	var createBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/list":
			listCalls++
			if listCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"home-created","name":"新家"}]}`))
		case "/apis/iot/v2/thing/manage/house/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/w/create":
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"home-created"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-home-create-secret", "client-home-create-1", "")

	input := `{"contractVersion":"1.0","requestId":"req-home-create-execute","locale":"zh-CN","utterance":"创建一个叫新家的家庭","intent":"home.create","parameters":{"name":"新家","description":"常住房","icon":"home","areaCode":"CN-310000","areaName":"上海"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("execute exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["houseId"] != "home-created" || result["name"] != "新家" || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
	if createBody["name"] != "新家" || createBody["desc"] != "常住房" || createBody["icon"] != "home" || createBody["areaCode"] != "CN-310000" || createBody["areaName"] != "上海" {
		t.Fatalf("create body = %#v", createBody)
	}
	if strings.Contains(stdout.String(), "token-home-create-secret") || strings.Contains(stderr.String(), "token-home-create-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) < 5 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}
