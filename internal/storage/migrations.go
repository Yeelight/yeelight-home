package storage

import (
	"context"
	"strings"
)

type Migration struct {
	ID  string
	SQL string
}

type Execer interface {
	ExecContext(ctx context.Context, statement string, args ...any) (any, error)
}

func MemoryMigrations() []Migration {
	return []Migration{
		{
			ID: "001_memory_core",
			SQL: `
create table if not exists privacy_consents (
  profile text not null,
  house_id text not null,
  consent_version text not null,
  learning_enabled integer not null default 0,
  created_at integer not null,
  updated_at integer not null,
  primary key (profile, house_id)
);

create table if not exists explicit_preferences (
  id text primary key,
  profile text not null,
  house_id text not null,
  scope_type text not null,
  scope_ref text not null,
  preference_type text not null,
  preference_value text not null,
  evidence_ref text,
  created_at integer not null,
  updated_at integer not null
);

create table if not exists interaction_events (
  id text primary key,
  profile text not null,
  house_id text not null,
  event_type text not null,
  intent text not null,
  target_summary text,
  evidence_ref text,
  created_at integer not null
);

create table if not exists recommendations (
  id text primary key,
  profile text not null,
  house_id text not null,
  recommendation_type text not null,
  explanation text not null,
  evidence_ref text not null,
  status text not null default 'pending',
  created_at integer not null,
  updated_at integer not null
);
`,
		},
	}
}

func JoinMigrationSQL(migrations []Migration) string {
	var builder strings.Builder
	for _, migration := range migrations {
		builder.WriteString(migration.SQL)
		builder.WriteString("\n")
	}
	return builder.String()
}

func ApplyMigrations(ctx context.Context, execer Execer, migrations []Migration) error {
	for _, migration := range migrations {
		if _, err := execer.ExecContext(ctx, migration.SQL); err != nil {
			return err
		}
	}
	return nil
}
