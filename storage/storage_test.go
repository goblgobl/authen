package storage

import (
	"testing"

	"src.goblgobl.com/authen/storage/pg"
	"src.goblgobl.com/authen/storage/sqlite"
	"src.goblgobl.com/tests"
	"src.goblgobl.com/tests/assert"
)

func Test_Configure_InvalidType(t *testing.T) {
	err := Configure(Config{Type: "invalid"})
	assert.Equal(t, err.Error(), "code: 103003 - storage.type is invalid. Should be one of: postgres, cockroach or sqlite")
}

func Test_Configure_Sqlite(t *testing.T) {
	config := Config{
		Type:   "sqlite",
		Sqlite: sqlite.Config{Path: ":memory:"},
	}
	err := Configure(config)
	assert.Nil(t, err)
	_, ok := DB.(sqlite.Conn)
	assert.True(t, ok)
}

func Test_Configure_PG(t *testing.T) {
	if tests.StorageType() != "postgres" {
		return
	}
	config := Config{
		Type:     "postgres",
		Postgres: pg.Config{URL: tests.PG()},
	}
	err := Configure(config)
	assert.Nil(t, err)
	_, ok := DB.(pg.DB)
	assert.True(t, ok)
}
