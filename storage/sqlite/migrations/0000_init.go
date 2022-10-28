package migrations

import (
	"fmt"

	"src.goblgobl.com/utils/sqlite"
)

// called from within a transaction
func Migrate_0000(conn sqlite.Conn) error {
	if err := createProjects(conn); err != nil {
		return err
	}
	if err := createTOTP(conn); err != nil {
		return err
	}
	return nil
}

func createProjects(conn sqlite.Conn) error {
	err := conn.Exec(`
		create table authen_projects (
			id text not null primary key,
			totp_max int not null,
			totp_issuer text not null,
			totp_setup_ttl int not null,
			totp_secret_length int not null,
			created int not null default(unixepoch()),
			updated int not null default(unixepoch())
	)`)

	if err != nil {
		return fmt.Errorf("sqlite 0000 authen_projects - %w", err)
	}

	return nil
}

func createTOTP(conn sqlite.Conn) error {
	if err := conn.Exec(`
		create table authen_totps (
			project_id text not null,
			user_id text not null,
			type text not null,
			pending int not null,
			secret blob not null,
			expires int null,
			created int not null default(unixepoch()),
			primary key (project_id, user_id, type, pending)
	)`); err != nil {
		return fmt.Errorf("sqlite 0000 authen_totps - %w", err)
	}

	return nil
}
