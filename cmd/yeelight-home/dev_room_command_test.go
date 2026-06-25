package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/credential"
)

func TestDevSeedRoomRequiresAllowWriteDev(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := newTestApp(t).run([]string{"dev", "seed-room", "--json", "--region", "dev", "--name", "Codex Dev Test Room"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--allow-write-dev") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestDevSeedRoomRequiresHouseID(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-dev-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-dev-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"dev", "seed-room", "--json", "--region", "dev", "--name", "Codex Dev Test Room", "--allow-write-dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "house id is required") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestDevSeedRoomCreatesRoomUsingStoredHouseID(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var createBody map[string]any
	roomListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200176/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200176/room/r/info/1/100":
			roomListCalls++
			if roomListCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-created","name":"Codex Dev Test Room"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200176/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200176/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200176/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200176/room/w/create":
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"room-created"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")

	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-dev-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-dev-1", HouseID: "200176"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"dev", "seed-room", "--json", "--region", "dev", "--name", "Codex Dev Test Room", "--allow-write-dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-dev-secret") || strings.Contains(stderr.String(), "token-dev-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if createBody["houseId"] != float64(200176) {
		t.Fatalf("createBody = %#v", createBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["houseId"] != "200176" || response["roomId"] != "room-created" || response["created"] != true || response["verified"] != true {
		t.Fatalf("response = %#v", response)
	}
}
