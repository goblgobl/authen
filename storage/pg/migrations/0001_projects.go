package migrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func Migrate_0001(tx pgx.Tx) error {
	bg := context.Background()
	if _, err := tx.Exec(bg, `
		create table authen_projects (
			id uuid not null primary key,
			totp_max int not null,
			totp_issuer text not null,
			totp_setup_ttl int not null,
			totp_secret_length int not null,
			ticket_max int not null,
			ticket_max_payload_length int not null,
			login_log_max int not null,
			login_log_max_payload_length int not null,
			created timestamptz not null default now(),
			updated timestamptz not null default now()
		)`); err != nil {
		return fmt.Errorf("pg 0001 migration authen_projects - %w", err)
	}

	if _, err := tx.Exec(bg, `
		create index authen_projects_updated on authen_projects(updated)
	`); err != nil {
		return fmt.Errorf("pg 0001 migration authen_projects_updated - %w", err)
	}
	return nil
}
