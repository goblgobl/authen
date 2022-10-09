package sqlite

import (
	"time"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/authen/storage/sqlite/migrations"
	"src.goblgobl.com/utils/sqlite"
	"src.goblgobl.com/utils/typed"
)

type Conn struct {
	sqlite.Conn
}

func New(config typed.Typed) (Conn, error) {
	filePath := config.String("path")
	conn, err := sqlite.New(filePath, true)
	if err != nil {
		return Conn{}, err
	}
	return Conn{conn}, nil
}

func (c Conn) Ping() error {
	return c.Exec("select 1")
}

func (c Conn) EnsureMigrations() error {
	return migrations.Run(c.Conn)
}

func (c Conn) Info() (any, error) {
	migration, err := sqlite.GetCurrentMigrationVersion(c.Conn)
	if err != nil {
		return nil, err
	}

	return struct {
		Type      string `json:"type"`
		Migration int    `json:"migration"`
	}{
		Type:      "sqlite",
		Migration: migration,
	}, nil
}

func (c Conn) GetProject(id string) (*data.Project, error) {
	row := c.Row("select id, max_users from authen_projects where id = ?1", id)

	project, err := scanProject(row)
	if err == sqlite.ErrNoRows {
		return nil, nil
	}
	return project, err
}

func (c Conn) GetUpdatedProjects(timestamp time.Time) ([]*data.Project, error) {
	// Not sure fetching the count upfront really makes much sense.
	// But we do expect this to be 0 almost every time that it's called, so most
	// of the time we're going to be doing a single DB call (either to get the count
	// which returns 0, or to get an empty result set).
	count, err := sqlite.Scalar[int](c.Conn, "select count(*) from authen_projects where updated > ?1", timestamp)
	if count == 0 || err != nil {
		return nil, err
	}

	rows := c.Rows("select id, max_users from authen_projects where updated > ?1", timestamp)
	defer rows.Close()

	projects := make([]*data.Project, 0, count)
	for rows.Next() {
		project, err := scanProject(&rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, rows.Error()
}

func scanProject(scanner sqlite.Scanner) (*data.Project, error) {
	var id string
	var maxUsers int

	if err := scanner.Scan(&id, &maxUsers); err != nil {
		return nil, err
	}

	return &data.Project{
		Id:       id,
		MaxUsers: uint32(maxUsers),
	}, nil
}
