package migrations

import (
	"fmt"

	"src.goblgobl.com/utils/sqlite"
)

// called from within a transaction
func Migrate_0002(conn sqlite.Conn) error {
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
		return fmt.Errorf("sqlite 0002 authen_totps - %w", err)
	}

	return nil
}
