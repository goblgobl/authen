package migrations

import (
	"context"

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
			issuer text not null,
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

func createTOTP(tx pgx.Tx) error {
	bg := context.Background()
	if _, err := tx.Exec(bg, `
		create table authen_totp_setups (
			project_id uuid not null,
			user_id text not null,
			nonce bytea not null,
			secret bytea not null,
			created timestamptz not null default now(),
			primary key (project_id, user_id)
		)`); err != nil {
		return err
	}

	if _, err := tx.Exec(bg, `
		create table authen_totps (
			project_id uuid not null,
			user_id text not null,
			nonce bytea not null,
			secret bytea not null,
			created timestamptz not null default now(),
			primary key (project_id, user_id)
		)`); err != nil {
		return err
	}

	return nil
}
