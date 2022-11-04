package migrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func Migrate_0003(tx pgx.Tx) error {
	if _, err := tx.Exec(context.Background(), `
		create table authen_tickets (
			project_id text not null,
			ticket bytea not null,
			expires timestamptz null,
			uses int null,
			payload bytea null,
			created timestamptz not null default now(),
			primary key (project_id, ticket)

		)`); err != nil {
		return fmt.Errorf("pg 0003 migration authen_tickets - %w", err)
	}

	return nil
}
