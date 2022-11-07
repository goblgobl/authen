package migrations

import (
	"src.goblgobl.com/utils/pg"
)

func Run(db pg.DB) error {
	migrations := []pg.Migration{
		pg.Migration{1, Migrate_0001},
		pg.Migration{2, Migrate_0002},
		pg.Migration{3, Migrate_0003},
		pg.Migration{4, Migrate_0004},
	}
	return pg.MigrateAll(db, "authen", migrations)
}

func GetCurrent(db pg.DB) (int, error) {
	return pg.GetCurrentMigrationVersion(db, "authen")
}
