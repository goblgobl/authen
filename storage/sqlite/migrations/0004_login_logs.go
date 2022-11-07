package migrations

import (
	"fmt"

	"src.goblgobl.com/utils/sqlite"
)

// called from within a transaction
func Migrate_0004(conn sqlite.Conn) error {
	if err := conn.Exec(`
		create table authen_login_logs (
			id text not null primary key,
			project_id text not null,
			user_id text not null,
			status int not null,
			meta blob null,
			created int not null default(unixepoch())
	)`); err != nil {
		return fmt.Errorf("sqlite 0004 authen_login_logs - %w", err)
	}

	if err := conn.Exec(`
		create index authen_login_logs_project_id_user_id on authen_login_logs(project_id, user_id)
	`); err != nil {
		return fmt.Errorf("sqlite 0004 migration authen_login_logs_project_id_user_id - %w", err)
	}
	return nil
}
