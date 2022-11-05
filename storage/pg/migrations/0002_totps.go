package migrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func Migrate_0002(tx pgx.Tx) error {
	bg := context.Background()

	if _, err := tx.Exec(bg, `
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

	if _, err := tx.Exec(bg, `
		create index authen_totps_expires on authen_totps(expires) where expires is not null
	`); err != nil {
		return fmt.Errorf("pg 0002 migration authen_totps_expires - %w", err)
	}

	return nil
}
