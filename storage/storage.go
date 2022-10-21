package storage

import (
	"time"

	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/authen/storage/pg"
	"src.goblgobl.com/authen/storage/sqlite"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/typed"
)

// singleton
var DB Storage

type Storage interface {
	// health check the storage, returns nil if everything is ok
	Ping() error

	// return information about the storage
	Info() (any, error)

	EnsureMigrations() error

	GetProject(id string) (*data.Project, error)
	GetUpdatedProjects(timestamp time.Time) ([]*data.Project, error)

	CreateTOTP(opts data.CreateTOTP) (data.CreateTOTPResult, error)
	CreateTOTPSetup(opts data.CreateTOTP) (data.CreateTOTPResult, error)
	GetTOTPSetup(opts data.GetTOTPSetup) (data.GetTOTPSetupResult, error)
}

func Configure(config typed.Typed) (err error) {
	switch config.String("type") {
	case "pg":
		DB, err = pg.New(config)
	case "sqlite":
		DB, err = sqlite.New(config)
	default:
		err = log.Errf(codes.ERR_INVALID_STORAGE_TYPE, "storage.type is invalid. Should be one of: pg or sqlite")
	}
	return
}
