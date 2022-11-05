package storage

import (
	"strings"
	"time"

	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/authen/storage/pg"
	"src.goblgobl.com/authen/storage/sqlite"
	"src.goblgobl.com/utils/log"
)

// singleton
var DB Storage

type Storage interface {
	// health check the storage, returns nil if everything is ok
	Ping() error

	// return information about the storage
	Info() (any, error)

	// clean any data that needs cleaning
	Clean() error

	EnsureMigrations() error

	GetProject(id string) (*data.Project, error)
	GetUpdatedProjects(timestamp time.Time) ([]*data.Project, error)

	TOTPGet(opts data.TOTPGet) (data.TOTPGetResult, error)
	TOTPCreate(opts data.TOTPCreate) (data.TOTPCreateResult, error)
	TOTPDelete(opts data.TOTPGet) (int, error)

	TicketUse(opts data.TicketUse) (data.TicketUseResult, error)
	TicketDelete(opts data.TicketUse) (data.TicketUseResult, error)
	TicketCreate(opts data.TicketCreate) (data.TicketCreateResult, error)
}

func Configure(config Config) (err error) {
	switch strings.ToLower(config.Type) {
	case "postgres":
		DB, err = pg.New(config.Postgres, "postgres")
	case "cockroach":
		DB, err = pg.New(config.Cockroach, "cockroach")
	case "sqlite":
		DB, err = sqlite.New(config.Sqlite)
	default:
		err = log.Errf(codes.ERR_INVALID_STORAGE_TYPE, "storage.type is invalid. Should be one of: postgres, cockroach or sqlite")
	}
	return
}
