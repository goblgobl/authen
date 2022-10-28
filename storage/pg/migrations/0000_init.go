package migrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func Migrate_0000(tx pgx.Tx) error {
	if err := createProjects(tx); err != nil {
		return err
	}
	if err := createTOTP(tx); err != nil {
		return err
	}
	return nil
}

func createProjects(tx pgx.Tx) error {
	bg := context.Background()
	if _, err := tx.Exec(bg, `
		create table authen_projects (
			id uuid not null primary key,
			totp_max int not null,
			totp_issuer text not null,
			totp_setup_ttl int not null,
			totp_secret_length int not null,
			created timestamptz not null default now(),
			updated timestamptz not null default now()
		)`); err != nil {
		return fmt.Errorf("pg 0000 migration authen_projects - %w", err)
	}

	if _, err := tx.Exec(bg, `
		create index authen_projects_updated on authen_projects(updated)
	`); err != nil {
		return fmt.Errorf("pg 0000 migration authen_projects_updated - %w", err)
	}
	return nil
}

func createTOTP(tx pgx.Tx) error {
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
		return fmt.Errorf("pg 0000 migration authen_totps - %w", err)
	}

	return nil
}
