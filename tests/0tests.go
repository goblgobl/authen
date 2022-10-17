package tests

// This _needs_ to be called "0tests", because we need the init
// in this file to execute before the init in any other file
// (awful)

import (
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/tests"
	"src.goblgobl.com/utils/validation"
)

func init() {
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
