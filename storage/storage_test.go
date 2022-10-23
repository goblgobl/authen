package storage

import (
	"testing"

	"src.goblgobl.com/authen/storage/pg"
	"src.goblgobl.com/authen/storage/sqlite"
	"src.goblgobl.com/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/utils/typed"
)

func Test_Configure_InvalidType(t *testing.T) {
	err := Configure(typed.Typed{"type": "invalid"})
	assert.Equal(t, err.Error(), "code: 103003 - storage.type is invalid. Should be one of: pg, cr or sqlite")
}

func Test_Configure_Sqlite(t *testing.T) {
	err := Configure(typed.Typed{"type": "sqlite", "path": ":memory:"})
	assert.Nil(t, err)
	_, ok := DB.(sqlite.Conn)
	assert.True(t, ok)
}

func Test_Configure_PG(t *testing.T) {
	err := Configure(typed.Typed{"type": "pg", "url": tests.PG()})
	assert.Nil(t, err)
	_, ok := DB.(pg.DB)
	assert.True(t, ok)
}
