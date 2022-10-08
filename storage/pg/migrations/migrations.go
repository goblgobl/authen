package migrations

import (
	"src.goblgobl.com/utils/pg"
)

func Run(db pg.DB) error {
	migrations := []pg.Migration{
		pg.Migration{1, Migrate_0000},
	}
	return pg.MigrateAll(db, "authen", migrations)
}

func GetCurrent(db pg.DB) (int, error) {
	return pg.GetCurrentMigrationVersion(db, "authen")
}
