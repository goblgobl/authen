package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func Migrate_0000(tx pgx.Tx) error {
	bg := context.Background()
	if _, err := tx.Exec(bg, `
		create table authen_projects (
			id text not null primary key,
			max_users int not null,
			created timestamptz not null default now(),
			updated timestamptz not null default now()
		)`); err != nil {
		return err
	}

	if _, err := tx.Exec(bg, `
		create index authen_projects_updated on authen_projects(updated)
	`); err != nil {
		return err
	}

	return nil
}
