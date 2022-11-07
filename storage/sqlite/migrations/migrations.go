package migrations

import (
	"src.goblgobl.com/utils/sqlite"
)

func Run(conn sqlite.Conn) error {
	migrations := []sqlite.Migration{
		sqlite.Migration{1, Migrate_0001},
		sqlite.Migration{2, Migrate_0002},
		sqlite.Migration{3, Migrate_0003},
		sqlite.Migration{4, Migrate_0004},
	}
	return sqlite.MigrateAll(conn, migrations)
}
