package tests

// This _needs_ to be called "0tests", because we need the init
// in this file to execute before the init in any other file
// (awful)

import (
	"math/rand"
	"regexp"
	"time"

	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/tests"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"
)

var generator tests.Generator

func init() {
	rand.Seed(time.Now().UnixNano())
	err := validation.Configure(validation.Config{
		PoolSize:  1,
		MaxErrors: 10,
	})

	if err != nil {
		panic(err)
	}

	if err := storage.Configure(tests.StorageConfig()); err != nil {
		panic(err)
	}

	if err := storage.DB.EnsureMigrations(); err != nil {
		panic(err)
	}
}

func String(constraints ...int) string {
	return generator.String(constraints...)
}

func UUID() string {
	return generator.UUID()
}

type TestableDB interface {
	Placeholder(i int) string
	RowToMap(sql string, args ...any) (typed.Typed, error)
}

var PlaceholderPattern = regexp.MustCompile(`\$(\d+)`)

func Row(sql string, args ...any) typed.Typed {
	db := storage.DB.(TestableDB)
	// no one's going to like this, but not sure how else to deal with it
	if db.Placeholder(0) == "?1" {
		sql = PlaceholderPattern.ReplaceAllString(sql, "?$1")
	}
	row, err := db.RowToMap(sql, args...)
	if err != nil {
		panic(err)
	}
	return row
}
