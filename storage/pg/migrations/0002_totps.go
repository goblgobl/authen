package migrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func Migrate_0002(tx pgx.Tx) error {
	if _, err := tx.Exec(context.Background(), `
		create table authen_totps (
			project_id uuid not null,
			user_id text not null,
			type text not null,
			pending bool not null,
			secret bytea not null,
			expires timestamptz null,
			created timestamptz not null default now(),
			primary key (project_id, user_id, type, pending)
		)`); err != nil {
		return fmt.Errorf("pg 0002 migration authen_totps - %w", err)
	}

	return nil
}
