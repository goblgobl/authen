package migrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func Migrate_0004(tx pgx.Tx) error {
	bg := context.Background()
	if _, err := tx.Exec(bg, `
		create table authen_login_logs (
			id uuid not null primary key,
			project_id text not null,
			user_id text not null,
			status int not null,
			payload bytea null,
			created timestamptz not null default now()
		)`); err != nil {
		return fmt.Errorf("pg 0004 migration authen_login_logs - %w", err)
	}

	if _, err := tx.Exec(bg, `
		create index authen_login_logs_project_id_user_id on authen_login_logs(project_id, user_id)
	`); err != nil {
		return fmt.Errorf("pg 0004 migration authen_login_logs_project_id_user_id - %w", err)
	}

	return nil
}
