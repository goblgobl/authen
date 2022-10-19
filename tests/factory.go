package tests

import (
	"time"

	"src.goblgobl.com/authen/storage"
	f "src.goblgobl.com/tests/factory"
	"src.goblgobl.com/utils/uuid"
)

type factory struct {
	Project   f.Table
	TOTP      f.Table
	TOTPSetup f.Table
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
			"issuer":    args.String("issuer", ""),
			"max_users": args.Int("max_users", 100),
			"created":   args.Time("created", time.Now()),
			"updated":   args.Time("updated", time.Now()),
		}
	})

	Factory.TOTP = f.NewTable("authen_totps", func(args f.KV) f.KV {
		return f.KV{
			"project_id": args.UUID("project_id", uuid.String()),
			"user_id":    args.String("issuer", uuid.String()),
			"nonce":      args.String("nonce", ""),
			"secret":     args.String("secret", ""),
			"created":    args.Time("created", time.Now()),
		}
	})

	Factory.TOTPSetup = f.NewTable("authen_totp_setups", func(args f.KV) f.KV {
		return f.KV{
			"project_id": args.UUID("project_id", uuid.String()),
			"user_id":    args.String("issuer", uuid.String()),
			"nonce":      args.String("nonce", ""),
			"secret":     args.String("secret", ""),
			"created":    args.Time("created", time.Now()),
		}
	})
}
