package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func Migrate_0000(tx pgx.Tx) error {
	if _, err := tx.Exec(context.Background(), `
		create table authen_projects (
			id text not null primary key,
			max_users int not null
	)`); err != nil {
		return err
	}

	return nil
}
