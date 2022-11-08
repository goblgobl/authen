package migrations

import (
	"fmt"

	"src.goblgobl.com/utils/sqlite"
)

// called from within a transaction
func Migrate_0001(conn sqlite.Conn) error {
	err := conn.Exec(`
		create table authen_projects (
			id text not null primary key,
			totp_max int not null,
			totp_issuer text not null,
			totp_setup_ttl int not null,
			totp_secret_length int not null,
			ticket_max int not null,
			ticket_max_payload_length int not null,
			login_log_max int not null,
			login_log_max_payload_length int not null,
			created int not null default(unixepoch()),
			updated int not null default(unixepoch())
	)`)

	if err != nil {
		return fmt.Errorf("sqlite 0001 authen_projects - %w", err)
	}

	return nil
}
