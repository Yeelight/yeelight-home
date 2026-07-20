package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/credential"
)

func TestProfileUsePersistsCommercialBizType(t *testing.T) {
	app := newTestApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"profile", "use", "--profile", "commercial", "--region", "cn", "--biz-type", "商照家庭", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	metadata, ok, err := app.metadataStore.Load("commercial")
	if err != nil || !ok || metadata.BizType != api.BizTypeCommercial {
		t.Fatalf("metadata=%#v ok=%v err=%v", metadata, ok, err)
	}
}

func TestProfileUseClearsHouseWhenBizTypeChanges(t *testing.T) {
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "cn", BizType: api.BizTypeConsumer, HouseID: "consumer-house"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"profile", "use", "--profile", "default", "--biz-type", "commercial", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil || !ok || metadata.BizType != api.BizTypeCommercial || metadata.HouseID != "" {
		t.Fatalf("metadata=%#v ok=%v err=%v", metadata, ok, err)
	}
}

func TestProfileUseAllowsExplicitHouseWhenBizTypeChanges(t *testing.T) {
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "cn", BizType: api.BizTypeConsumer, HouseID: "consumer-house"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"profile", "use", "--profile", "default", "--biz-type", "commercial", "--house-id", "project-1", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	metadata, _, _ := app.metadataStore.Load("default")
	if metadata.BizType != api.BizTypeCommercial || metadata.HouseID != "project-1" {
		t.Fatalf("metadata=%#v", metadata)
	}
}

func TestAuthTokenSetClearsHouseWhenBizTypeChanges(t *testing.T) {
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "cn", BizType: api.BizTypeConsumer, HouseID: "consumer-house"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"auth", "token", "set", "--token", "Bearer test-token", "--profile", "default", "--biz-type", "commercial", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	metadata, _, _ := app.metadataStore.Load("default")
	if metadata.BizType != api.BizTypeCommercial || metadata.HouseID != "" {
		t.Fatalf("metadata=%#v", metadata)
	}
}

func TestRuntimeContextDoesNotReuseUnpairedEnvironmentHouseAcrossBizTypes(t *testing.T) {
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "cn", BizType: api.BizTypeConsumer, HouseID: "consumer-house"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	t.Setenv("YEELIGHT_HOME_HOUSE_ID", "environment-house")
	contextInfo, err := app.resolveRuntimeContext(cliFlags{values: map[string]string{"biz-type": api.BizTypeCommercial}})
	if err != nil {
		t.Fatalf("resolveRuntimeContext error: %v", err)
	}
	if contextInfo.HouseID != "" {
		t.Fatalf("cross-type houseId=%q", contextInfo.HouseID)
	}
	t.Setenv("YEELIGHT_HOME_BIZ_TYPE", api.BizTypeCommercial)
	contextInfo, err = app.resolveRuntimeContext(cliFlags{values: map[string]string{}})
	if err != nil || contextInfo.HouseID != "environment-house" {
		t.Fatalf("paired context=%#v err=%v", contextInfo, err)
	}
}

func TestSetupPlanExposesSelectedCommercialBizType(t *testing.T) {
	app := newTestApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"setup", "--mode", "skill", "--agent", "codex", "--lang", "en-US", "--biz-type", "commercial", "--home-dir", t.TempDir(), "--plan", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var plan map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if plan["bizType"] != api.BizTypeCommercial {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestSetupHomeSelectionReplacesHouseFromDifferentBizType(t *testing.T) {
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "cn", BizType: api.BizTypeConsumer, HouseID: "consumer-house"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	data := []byte(`{"houses":[{"id":"project-1","name":"Commercial Project"}]}`)
	if err := app.selectDefaultSetupHome(data, setupExecutionOptions{Profile: "default", Region: "cn", BizType: api.BizTypeCommercial, Locale: "en-US"}); err != nil {
		t.Fatalf("selectDefaultSetupHome error: %v", err)
	}
	metadata, _, _ := app.metadataStore.Load("default")
	if metadata.BizType != api.BizTypeCommercial || metadata.HouseID != "project-1" {
		t.Fatalf("metadata=%#v", metadata)
	}
}

func TestDoctorReportsCommercialBizType(t *testing.T) {
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "cn", BizType: api.BizTypeCommercial, HouseID: "project-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"doctor", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if response["bizType"] != api.BizTypeCommercial || response["houseId"] != "project-1" {
		t.Fatalf("response=%#v", response)
	}
}

func TestDoctorDoesNotTreatLegacyAuthenticatedFlagAsProfileCredential(t *testing.T) {
	t.Setenv("YEELIGHT_HOME_AUTHENTICATED", "1")
	app := newTestApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"doctor", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if response["authenticated"] != false || response["status"] != "warning" {
		t.Fatalf("response=%#v", response)
	}
}

func TestHomeListUsesCommercialProfileDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("bizType") != api.BizTypeCommercial {
			t.Fatalf("bizType = %q", request.Header.Get("bizType"))
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/commercial/saas/v1/user/r/saas-role":
			_, _ = writer.Write([]byte(`{"success":true,"data":"commercial_saas_admin"}`))
		case "/apis/commercial/saas/v1/user/r/project-role":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"project-1":1}}`))
		case "/apis/commercial/saas/v1/project/r/page":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"project-1","name":"商照展厅"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "Bearer test-token")
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", BizType: api.BizTypeCommercial}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"home", "list", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if response["bizType"] != api.BizTypeCommercial || response["houseCount"] != float64(1) || response["source"] != "/apis/commercial/saas/v1/project/r/page" {
		t.Fatalf("response = %#v", response)
	}
}
