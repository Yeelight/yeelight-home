package contract

import "testing"

func TestDecodeRequestRejectsUnknownFields(t *testing.T) {
	_, err := DecodeRequest([]byte(`{"contractVersion":"1.0","requestId":"req-1","locale":"zh-CN","utterance":"测试","intent":"home.summary","url":"https://example.com"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeRequestAcceptsKnownIntent(t *testing.T) {
	request, err := DecodeRequest([]byte(`{"contractVersion":"1.0","requestId":"req-1","locale":"zh-CN","utterance":"测试","intent":"home.summary"}`))
	if err != nil {
		t.Fatalf("DecodeRequest error: %v", err)
	}
	if request.Intent != "home.summary" {
		t.Fatalf("intent = %s", request.Intent)
	}
}

func TestDecodeRequestAcceptsIntentExplain(t *testing.T) {
	request, err := DecodeRequest([]byte(`{"contractVersion":"1.0","requestId":"req-intent-explain","locale":"zh-CN","utterance":"解释参数","intent":"intent.explain","parameters":{"intent":"lighting.design.import"}}`))
	if err != nil {
		t.Fatalf("DecodeRequest error: %v", err)
	}
	if request.Intent != "intent.explain" {
		t.Fatalf("intent = %s", request.Intent)
	}
}

func TestDecodeRequestAcceptsTargetID(t *testing.T) {
	request, err := DecodeRequest([]byte(`{"contractVersion":"1.0","requestId":"req-1","locale":"zh-CN","utterance":"看看主灯","intent":"entity.get","targets":[{"entityType":"device","id":"device-1"}]}`))
	if err != nil {
		t.Fatalf("DecodeRequest error: %v", err)
	}
	if request.Targets[0]["id"] != "device-1" {
		t.Fatalf("targets = %#v", request.Targets)
	}
}
