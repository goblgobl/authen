package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"src.goblgobl.com/utils/json"

	"src.goblgobl.com/authen/storage"
	f "src.goblgobl.com/tests/factory"
	"src.goblgobl.com/utils/encryption"
	"src.goblgobl.com/utils/uuid"
)

type factory struct {
	Project f.Table
	TOTP    f.Table
	Ticket  f.Table
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
			"id":                        args.UUID("id", uuid.String()),
			"totp_max":                  args.Int("totp_max", 100),
			"totp_issuer":               args.String("totp_issuer", ""),
			"totp_setup_ttl":            args.Int("totp_setup_ttl", 120),
			"totp_secret_length":        args.Int("totp_secret_length", 32),
			"ticket_max":                args.Int("ticket_max", 100),
			"ticket_max_payload_length": args.Int("ticket_max_payload_length", 128),
			"created":                   args.Time("created", time.Now()),
			"updated":                   args.Time("updated", time.Now()),
		}
	})

	Factory.TOTP = f.NewTable("authen_totps", func(args f.KV) f.KV {
		encryptedSecret := []byte{1}
		if secret := args.String("secret", "").(string); secret != "" {
			var err error
			var key [32]byte
			switch t := args["key"].(type) {
			case [32]byte:
				key = t
			case string:
				slice, err := hex.DecodeString(t)
				if err != nil {
					panic(err)
				}
				key = *(*[32]byte)(slice)
			default:
				key = [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}
			}
			encryptedSecret, err = encryption.Encrypt(key, secret)
			if err != nil {
				panic(err)
			}
		}

		return f.KV{
			"project_id": args.UUID("project_id", uuid.String()),
			"user_id":    args.String("user_id", uuid.String()),
			"type":       args.String("type", ""),
			"pending":    args.Bool("pending", false),
			"secret":     encryptedSecret,
			"expires":    args.Time("expires"),
			"created":    args.Time("created", time.Now()),
		}
	})

	Factory.Ticket = f.NewTable("authen_tickets", func(args f.KV) f.KV {
		var payload *[]byte
		if p, ok := args["payload"]; ok {
			p, err := json.Marshal(p)
			if err != nil {
				panic(err)
			}
			payload = &p
		}

		ticket := args.String("ticket", uuid.String()).(string)
		ticketHash := sha256.Sum256([]byte(ticket))

		return f.KV{
			"project_id": args.UUID("project_id", uuid.String()),
			"ticket":     ticketHash[:],
			"expires":    args.Time("expires"),
			"uses":       args.Int("uses"),
			"payload":    payload,
			"created":    args.Time("created", time.Now()),
		}
	})
}
