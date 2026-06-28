package operation

import (
	"strings"
	"testing"
	"time"
)

func TestPreparedOperationKeepsPayloadInMemoryOnly(t *testing.T) {
	now := time.Unix(1000, 0)
	prepared, err := NewPrepared("default", "dev", "200171", "room.create", "req-1", "创建房间 测试", map[string]any{
		"houseId": "200171",
		"name":    "测试",
	}, []string{"重新读取家庭实体"}, now)
	if err != nil {
		t.Fatalf("NewPrepared error: %v", err)
	}
	if prepared.CreatedAt != now.Unix() || prepared.Intent != "room.create" {
		t.Fatalf("prepared = %#v", prepared)
	}
	if err := prepared.Verify(now.Add(24 * time.Hour)); err != nil {
		t.Fatalf("Verify error: %v", err)
	}
}

func TestPreparedOperationRejectsSensitivePayload(t *testing.T) {
	_, err := NewPrepared("default", "dev", "200171", "room.create", "req-1", "bad", map[string]any{
		"accessToken": "secret",
	}, nil, time.Unix(1000, 0))
	if err == nil || !strings.Contains(err.Error(), "token-like") {
		t.Fatalf("err = %v", err)
	}
}

func TestAccountPreparedUsesExplicitAccountScope(t *testing.T) {
	now := time.Unix(1000, 0)
	prepared, err := NewAccountPrepared("default", "dev", "home.create", "req-1", "创建家庭", map[string]any{
		"name": "新家",
	}, nil, now)
	if err != nil {
		t.Fatalf("NewAccountPrepared error: %v", err)
	}
	if !IsAccountScope(prepared.HouseID) {
		t.Fatalf("prepared house scope = %q", prepared.HouseID)
	}
	if err := prepared.Verify(now); err != nil {
		t.Fatalf("Verify error: %v", err)
	}
}

func TestR3PreparedOperationIsCallerPolicyLabelOnly(t *testing.T) {
	now := time.Unix(1000, 0)
	prepared, err := NewPreparedWithRisk("default", "dev", "200171", "device.remove", "req-1", "删除设备", RiskR3, map[string]any{
		"deviceId": "device-1",
	}, nil, now)
	if err != nil {
		t.Fatalf("NewPreparedWithRisk error: %v", err)
	}
	if prepared.Risk != RiskR3 {
		t.Fatalf("prepared = %#v", prepared)
	}
	if err := prepared.Verify(now.Add(2 * time.Hour)); err != nil {
		t.Fatalf("Verify error: %v", err)
	}
}
