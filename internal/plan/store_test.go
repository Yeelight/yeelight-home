package plan

import (
	"strings"
	"testing"
	"time"
)

func TestStoreSavesLoadsAndVerifiesPendingPlan(t *testing.T) {
	now := time.Unix(1000, 0)
	store := NewStore(t.TempDir() + "/pending_plans.json").WithClock(func() time.Time { return now })
	record, err := NewRecord("default", "dev", "200171", "room.create", "req-1", "创建房间 测试", map[string]any{
		"houseId": float64(200171),
		"name":    "测试",
	}, []string{"重新读取家庭实体"}, now, time.Minute)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	loaded, ok, err := store.Load(record.ID)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if !ok || loaded.Hash != record.Hash || loaded.Status != StatusPending {
		t.Fatalf("loaded = %#v ok=%v", loaded, ok)
	}
	if err := loaded.Verify(now.Add(30 * time.Second)); err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	committed, err := store.MarkCommitted(record.ID)
	if err != nil {
		t.Fatalf("MarkCommitted error: %v", err)
	}
	if committed.Status != StatusCommitted || committed.CommittedAt != now.Unix() {
		t.Fatalf("committed = %#v", committed)
	}
}

func TestStoreListReturnsRecordsCopy(t *testing.T) {
	now := time.Unix(1000, 0)
	store := NewStore(t.TempDir() + "/pending_plans.json").WithClock(func() time.Time { return now })
	record, err := NewRecord("default", "dev", "200171", "memory.remember", "req-1", "记住偏好", map[string]any{
		"preferenceType":  "brightness",
		"preferenceValue": "45",
	}, nil, now, time.Minute)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	records, err := store.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(records) != 1 || records[0].ID != record.ID {
		t.Fatalf("records = %#v", records)
	}
	records[0].ID = "mutated"
	loaded, ok, err := store.Load(record.ID)
	if err != nil || !ok || loaded.ID != record.ID {
		t.Fatalf("loaded after mutation = %#v ok=%v err=%v", loaded, ok, err)
	}
}

func TestNewRecordRejectsSensitivePayload(t *testing.T) {
	_, err := NewRecord("default", "dev", "200171", "room.create", "req-1", "bad", map[string]any{
		"accessToken": "secret",
	}, nil, time.Unix(1000, 0), time.Minute)
	if err == nil || !strings.Contains(err.Error(), "token-like") {
		t.Fatalf("err = %v", err)
	}
}

func TestNewAccountRecordUsesExplicitAccountScope(t *testing.T) {
	now := time.Unix(1000, 0)
	record, err := NewAccountRecord("default", "dev", "home.create", "req-1", "创建家庭", map[string]any{
		"name": "新家",
	}, nil, now, time.Minute)
	if err != nil {
		t.Fatalf("NewAccountRecord error: %v", err)
	}
	if !IsAccountScope(record.HouseID) {
		t.Fatalf("record house scope = %q", record.HouseID)
	}
	if err := record.Verify(now); err != nil {
		t.Fatalf("Verify error: %v", err)
	}
}

func TestR3RecordRequiresLocalApproval(t *testing.T) {
	now := time.Unix(1000, 0)
	store := NewStore(t.TempDir() + "/pending_plans.json").WithClock(func() time.Time { return now })
	_, err := NewRecordWithRisk("default", "dev", "200171", "device.remove", "req-1", "删除设备", RiskR3, "", map[string]any{
		"deviceId": "device-1",
	}, nil, now, time.Minute)
	if err == nil || !strings.Contains(err.Error(), "approval challenge") {
		t.Fatalf("challenge err = %v", err)
	}
	record, err := NewRecordWithRisk("default", "dev", "200171", "device.remove", "req-1", "删除设备", RiskR3, "DELETE device.remove device-1", map[string]any{
		"deviceId": "device-1",
	}, nil, now, time.Minute)
	if err != nil {
		t.Fatalf("NewRecordWithRisk error: %v", err)
	}
	if record.Risk != RiskR3 || !record.ApprovalRequired || record.ApprovalChallenge == "" {
		t.Fatalf("record = %#v", record)
	}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if _, err := store.MarkApproved(record.ID, "wrong"); err == nil || !strings.Contains(err.Error(), "challenge mismatch") {
		t.Fatalf("wrong challenge err = %v", err)
	}
	approved, err := store.MarkApproved(record.ID, "DELETE device.remove device-1")
	if err != nil {
		t.Fatalf("MarkApproved error: %v", err)
	}
	if approved.ApprovedAt != now.Unix() {
		t.Fatalf("approved = %#v", approved)
	}
	loaded, ok, err := store.Load(record.ID)
	if err != nil || !ok || loaded.ApprovedAt != now.Unix() {
		t.Fatalf("loaded = %#v ok=%v err=%v", loaded, ok, err)
	}
}

func TestStoreMarkCanceledBlocksFutureVerify(t *testing.T) {
	now := time.Unix(1000, 0)
	store := NewStore(t.TempDir() + "/pending_plans.json").WithClock(func() time.Time { return now })
	record, err := NewRecord("default", "dev", "200171", "room.create", "req-1", "创建房间 测试", map[string]any{
		"houseId": float64(200171),
		"name":    "测试",
	}, nil, now, time.Minute)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	if err := store.Save(record); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	canceled, err := store.MarkCanceled(record.ID)
	if err != nil {
		t.Fatalf("MarkCanceled error: %v", err)
	}
	if canceled.Status != StatusCanceled || canceled.CanceledAt != now.Unix() {
		t.Fatalf("canceled = %#v", canceled)
	}
	loaded, ok, err := store.Load(record.ID)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if !ok || loaded.Status != StatusCanceled {
		t.Fatalf("loaded = %#v ok=%v", loaded, ok)
	}
	if err := loaded.Verify(now); err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("Verify error = %v", err)
	}
}

func TestRecordVerifyRejectsHashMismatchAndExpiredPlan(t *testing.T) {
	now := time.Unix(1000, 0)
	record, err := NewRecord("default", "dev", "200171", "room.create", "req-1", "创建房间 测试", map[string]any{
		"houseId": float64(200171),
		"name":    "测试",
	}, nil, now, time.Minute)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	record.Payload["name"] = "篡改"
	if err := record.Verify(now); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("hash err = %v", err)
	}

	record, err = NewRecord("default", "dev", "200171", "room.create", "req-1", "创建房间 测试", map[string]any{
		"houseId": float64(200171),
		"name":    "测试",
	}, nil, now, time.Minute)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	if err := record.Verify(now.Add(2 * time.Minute)); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired err = %v", err)
	}
}

func TestStoreCompactsExpiredAndOldTerminalPlans(t *testing.T) {
	now := time.Unix(10_000, 0)
	store := NewStore(t.TempDir()+"/pending_plans.json").
		WithClock(func() time.Time { return now }).
		WithRetention(time.Hour, time.Hour, 3)

	oldPending, err := NewRecord("default", "dev", "200171", "room.create", "req-old-pending", "旧待确认", map[string]any{"name": "old"}, nil, now.Add(-3*time.Hour), time.Minute)
	if err != nil {
		t.Fatalf("old pending: %v", err)
	}
	oldCommitted, err := NewRecord("default", "dev", "200171", "room.create", "req-old-committed", "旧已提交", map[string]any{"name": "old-committed"}, nil, now.Add(-3*time.Hour), time.Minute)
	if err != nil {
		t.Fatalf("old committed: %v", err)
	}
	oldCommitted.Status = StatusCommitted
	oldCommitted.CommittedAt = now.Add(-2 * time.Hour).Unix()

	active, err := NewRecord("default", "dev", "200171", "room.create", "req-active", "活跃待确认", map[string]any{"name": "active"}, nil, now, time.Hour)
	if err != nil {
		t.Fatalf("active: %v", err)
	}
	recentCommitted, err := NewRecord("default", "dev", "200171", "room.create", "req-recent", "最近已提交", map[string]any{"name": "recent"}, nil, now.Add(-10*time.Minute), time.Minute)
	if err != nil {
		t.Fatalf("recent committed: %v", err)
	}
	recentCommitted.Status = StatusCommitted
	recentCommitted.CommittedAt = now.Add(-5 * time.Minute).Unix()
	recentCanceled, err := NewRecord("default", "dev", "200171", "room.create", "req-canceled", "最近已取消", map[string]any{"name": "canceled"}, nil, now.Add(-9*time.Minute), time.Minute)
	if err != nil {
		t.Fatalf("recent canceled: %v", err)
	}
	recentCanceled.Status = StatusCanceled
	recentCanceled.CanceledAt = now.Add(-4 * time.Minute).Unix()

	for _, record := range []Record{oldPending, oldCommitted, active, recentCommitted, recentCanceled} {
		if err := store.Save(record); err != nil {
			t.Fatalf("Save %s: %v", record.ID, err)
		}
	}

	if _, ok, err := store.Load(oldPending.ID); err != nil || ok {
		t.Fatalf("old pending load ok=%v err=%v", ok, err)
	}
	if _, ok, err := store.Load(oldCommitted.ID); err != nil || ok {
		t.Fatalf("old committed load ok=%v err=%v", ok, err)
	}
	for _, record := range []Record{active, recentCommitted, recentCanceled} {
		if _, ok, err := store.Load(record.ID); err != nil || !ok {
			t.Fatalf("expected %s kept, ok=%v err=%v", record.ID, ok, err)
		}
	}
}
