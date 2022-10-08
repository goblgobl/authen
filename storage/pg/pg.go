package pg

import (
	"context"
	"time"

	"src.goblgobl.com/utils/pg"
	"src.goblgobl.com/utils/typed"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/authen/storage/pg/migrations"
)

type DB struct {
	pg.DB
}

func New(config typed.Typed) (DB, error) {
	db, err := pg.New(config.String("url"))
	if err != nil {
		return DB{}, err
	}
	return DB{db}, nil
}

func (db DB) Ping() error {
	_, err := db.Exec(context.Background(), "select 1")
	return err
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

func (db DB) GetProject(id string) (*data.Project, error) {
	row := db.QueryRow(context.Background(), `
		select id, max_users
		from authen_projects
		where id = $1
	`, id)

	project, err := scanProject(row)
	if err == pg.ErrNoRows {
		return nil, nil
	}
	return project, err
}

func (db DB) GetUpdatedProjects(timestamp time.Time) ([]*data.Project, error) {

	// Not sure fetching the count upfront really makes much sense.
	// But we do expect this to be 0 almost every time that it's called, so most
	// of the time we're going to be doing a single DB call (either to get the count
	// which returns 0, or to get an empty result set).
	count, err := pg.Scalar[int](db.DB, "select count(*) from authen_projects where updated > $1", timestamp)
	if count == 0 || err != nil {
		return nil, err
	}

	rows, err := db.Query(context.Background(), "select id, max_users from authen_projects where updated > $1", timestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]*data.Project, 0, count)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, rows.Err()
}

func scanProject(row pg.Row) (*data.Project, error) {
	var id string
	var maxUsers int
	if err := row.Scan(&id, &maxUsers); err != nil {
		return nil, err
	}

	return &data.Project{
		Id:       id,
		MaxUsers: uint32(maxUsers),
	}, nil
}
