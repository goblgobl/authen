package storage

import (
	"src.goblgobl.com/authen/storage/pg"
	"src.goblgobl.com/authen/storage/sqlite"
)

type Config struct {
	Type      string        `json:"type"`
	Sqlite    sqlite.Config `json:"sqlite"`
	Postgres  pg.Config     `json:"postgres"`
	Cockroach pg.Config     `json:"cockroach"`
}
