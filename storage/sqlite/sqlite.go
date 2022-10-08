package sqlite

import (
	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/authen/storage/sqlite/migrations"
	"src.goblgobl.com/utils/sqlite"
	usqlite "src.goblgobl.com/utils/sqlite"
	"src.goblgobl.com/utils/typed"
)

type Conn struct {
	usqlite.Conn
}

func New(config typed.Typed) (Conn, error) {
	filePath := config.String("path")
	conn, err := usqlite.New(filePath, true)
	if err != nil {
		return Conn{}, err
	}
	return Conn{conn}, nil
}

func (c Conn) Ping() error {
	return c.Exec("select 1")
}

func (c Conn) GetProject(id string) (*data.Project, error) {
	row := c.Row("select max_users from authen_projects where id = ?1", id)

	var maxUsers int
	exists, err := row.Scan(&maxUsers)
	if !exists || err != nil {
		return nil, err
	}

	return &data.Project{
		MaxUsers: uint32(maxUsers),
	}, nil
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
