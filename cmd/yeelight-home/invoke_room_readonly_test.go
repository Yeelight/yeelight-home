package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeRoomListAndSearchUseCloudReadonlyAdapters(t *testing.T) {
	var gotCalls []string
	var gotSearchBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/room/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"roomId":10,"houseId":1001,"name":"客厅","deviceIds":[1,2],"accessToken":"not-allowed"}]}}`))
		case "/apis/iot/v1/room/1001/r/fuzzy":
			if err := json.NewDecoder(request.Body).Decode(&gotSearchBody); err != nil {
				t.Fatalf("decode search body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"roomId":11,"houseId":1001,"name":"卧室","deviceIds":[3],"secret":"not-allowed"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-room-secret", "client-room-1", "1001")

	for _, input := range []string{
		`{"contractVersion":"1.0","requestId":"req-room-list","locale":"zh-CN","utterance":"列出这个家的房间","intent":"room.list","parameters":{"houseId":"1001"}}`,
		`{"contractVersion":"1.0","requestId":"req-room-search","locale":"zh-CN","utterance":"搜索卧室","intent":"room.search","parameters":{"houseId":"1001","name":"卧","pageNo":2,"pageSize":5}}`,
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		for _, forbidden := range []string{"token-room-secret", "not-allowed"} {
			if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
				t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
			}
		}
		var response map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("invalid json response: %v", err)
		}
		if response["status"] != "success" {
			t.Fatalf("response = %#v", response)
		}
		result := response["result"].(map[string]any)
		data := result["data"].(map[string]any)
		rooms := data["rooms"].([]any)
		if len(rooms) != 1 {
			t.Fatalf("rooms = %#v", data["rooms"])
		}
	}
	if strings.Join(gotCalls, "\n") != "POST /apis/iot/v1/room/r/all\nPOST /apis/iot/v1/room/1001/r/fuzzy" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if gotSearchBody["fuzzyName"] != "卧" || gotSearchBody["pageNo"] != float64(2) || gotSearchBody["pageSize"] != float64(5) {
		t.Fatalf("gotSearchBody = %#v", gotSearchBody)
	}
}
