package migrations

import (
	"src.goblgobl.com/utils/sqlite"
)

// called from within a transaction
func Migrate_0000(conn sqlite.Conn) error {
	if err := conn.Exec(`
		create table authen_projects (
			id text not null primary key,
			max_users int not null
	)`); err != nil {
		return err
	}

	return nil
}
