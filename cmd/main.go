package main

import (
	"flag"

	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/config"
	"src.goblgobl.com/authen/http"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/utils/log"
)

func main() {
	configPath := flag.String("config", "config.json", "full path to config file")
	migrations := flag.Bool("migrations", false, "only run migrations and exit")
	flag.Parse()

	config, err := config.Configure(*configPath)
	if err != nil {
		log.Fatal("load_config").String("path", *configPath).Err(err).Log()
		return
	}

	if err := authen.Init(config); err != nil {
		log.Fatal("authen_init").Err(err).Log()
		return
	}

	if *migrations || config.Migrations == nil || *config.Migrations == true {
		if err := storage.DB.EnsureMigrations(); err != nil {
			log.Fatal("authen_migrations").Err(err).Log()
			return
		}
	} else {
		log.Info("migrations_skip").Log()
	}

	if *migrations {
		return
	}

	http.Listen()
}
