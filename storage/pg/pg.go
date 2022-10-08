package pg

import (
	"context"

	"github.com/jackc/pgx/v5"
	upg "src.goblgobl.com/utils/pg"
	"src.goblgobl.com/utils/typed"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/authen/storage/pg/migrations"
)

type DB struct {
	upg.DB
}

func New(config typed.Typed) (DB, error) {
	db, err := upg.New(config.String("url"))
	if err != nil {
		return DB{}, err
	}
	return DB{db}, nil
}

func (db DB) Ping() error {
	_, err := db.Exec(context.Background(), "select 1")
	return err
}

func (db DB) GetProject(id string) (*data.Project, error) {
	row := db.QueryRow(context.Background(), `
		select max_users
		from authen_projects
		where id = $1
	`, id)

	var maxUsers int
	err := row.Scan(&maxUsers)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &data.Project{
		MaxUsers: uint32(maxUsers),
	}, nil
}

func (db DB) EnsureMigrations() error {
	return migrations.Run(db.DB)
}

func (db DB) Info() (any, error) {
	migration, err := migrations.GetCurrent(db.DB)
	if err != nil {
		return nil, err
	}

	return struct {
		Type      string `json:"type"`
		Migration int    `json:"migration"`
	}{
		Type:      "pg",
		Migration: migration,
	}, nil
}
