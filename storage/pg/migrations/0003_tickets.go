package migrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func Migrate_0003(tx pgx.Tx) error {
	bg := context.Background()
	if _, err := tx.Exec(bg, `
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

	if _, err := tx.Exec(bg, `
		create index authen_tickets_expires on authen_tickets(expires) where expires is not null
	`); err != nil {
		return fmt.Errorf("pg 0003 migration authen_tickets_expires - %w", err)
	}

	if _, err := tx.Exec(bg, `
		create index authen_tickets_uses on authen_tickets(uses) where uses is not null
	`); err != nil {
		return fmt.Errorf("pg 0003 migration authen_tickets_uses - %w", err)
	}

	return nil
}
