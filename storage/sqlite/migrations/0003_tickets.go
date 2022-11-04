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

	return nil
}
