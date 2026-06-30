package storage

import (
	"context"
	"strings"
	"testing"
)

func TestMemoryMigrationsDoNotCreateTokenColumns(t *testing.T) {
	for _, migration := range MemoryMigrations() {
		sql := strings.ToLower(migration.SQL)
		for _, forbidden := range []string{"token", "authorization", "cookie", "secret"} {
			if strings.Contains(sql, forbidden) {
				t.Fatalf("migration %s contains forbidden term %q", migration.ID, forbidden)
			}
		}
	}
}

func TestMemoryMigrationsIncludePrivacyConsentAndPreferenceTables(t *testing.T) {
	sql := strings.ToLower(JoinMigrationSQL(MemoryMigrations()))

	for _, table := range []string{
		"privacy_consents",
		"explicit_preferences",
		"interaction_events",
		"recommendations",
	} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Fatalf("missing table %s in migrations:\n%s", table, sql)
		}
	}
	if strings.Contains(sql, "implicit_preference_candidates") {
		t.Fatalf("runtime storage must not create subjective implicit memory candidate tables:\n%s", sql)
	}
}

func TestApplyMigrationsRunsStatementsInOrder(t *testing.T) {
	execer := &recordingExecer{}
	migrations := []Migration{
		{ID: "001", SQL: "create table a(id text);"},
		{ID: "002", SQL: "create table b(id text);"},
	}

	if err := ApplyMigrations(context.Background(), execer, migrations); err != nil {
		t.Fatalf("ApplyMigrations error: %v", err)
	}

	if len(execer.statements) != 2 {
		t.Fatalf("statements = %#v", execer.statements)
	}
	if !strings.Contains(execer.statements[0], "create table a") || !strings.Contains(execer.statements[1], "create table b") {
		t.Fatalf("statements out of order: %#v", execer.statements)
	}
}

type recordingExecer struct {
	statements []string
}

func (execer *recordingExecer) ExecContext(_ context.Context, statement string, _ ...any) (any, error) {
	execer.statements = append(execer.statements, statement)
	return nil, nil
}
