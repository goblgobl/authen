package tests

import (
	"src.goblgobl.com/authen/storage"
	f "src.goblgobl.com/tests/factory"
	"src.goblgobl.com/utils/uuid"
)

type factory struct {
	Project f.Table
}

var (
	// need to be created in init, after we've loaded out storage
	// engine, since the factories can change slightly based on
	// on the storage engine (e.g. how placeholders work)
	Factory factory
)

func init() {
	f.DB = storage.DB.(f.SQLStorage)
	Factory.Project = f.NewTable("authen_projects", func(args f.KV) f.KV {
		return f.KV{
			"id":        args.UUID("id", uuid.String()),
			"max_users": args.Int("max_users", 100),
		}
	})
}
