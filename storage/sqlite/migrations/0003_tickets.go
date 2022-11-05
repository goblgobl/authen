package migrations

import (
	"fmt"

	"src.goblgobl.com/utils/sqlite"
)

// called from within a transaction
func Migrate_0003(conn sqlite.Conn) error {
	if err := conn.Exec(`
		create table authen_tickets (
			project_id text not null,
			ticket blob not null,
			expires int null,
			uses int null,
			payload blob null,
			created int not null default(unixepoch()),
			primary key (project_id, ticket)
	)`); err != nil {
		return fmt.Errorf("sqlite 0003 authen_tickets - %w", err)
	}

	if err := conn.Exec(`
		create index authen_tickets_expires on authen_tickets(expires) where expires is not null
	`); err != nil {
		return fmt.Errorf("sqlite 0003 migration authen_tickets_expires - %w", err)
	}

	if err := conn.Exec(`
		create index authen_tickets_uses on authen_tickets(uses) where uses is not null
	`); err != nil {
		return fmt.Errorf("sqlite 0003 migration authen_tickets_uses - %w", err)
	}

	return nil
}
