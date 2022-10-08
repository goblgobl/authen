package main

import (
	"flag"

	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/config"
	"src.goblgobl.com/authen/http"
	"src.goblgobl.com/utils/log"
)

func main() {
	configPath := flag.String("config", "config.json", "full path to config file")
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
	http.Listen(config.HTTP)
}
