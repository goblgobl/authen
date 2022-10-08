package config

import (
	"os"

	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/utils/json"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"
)

type Config struct {
	InstanceId uint8             `json:"instance_id"`
	HTTP       HTTP              `json:"http"`
	Log        log.Config        `json:"log"`
	Storage    typed.Typed       `json:"storage"`
	Validation validation.Config `json:"validation"`
}

type HTTP struct {
	Listen string `json:"listen"`
}

func Configure(filePath string) (Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Config{}, log.Err(codes.ERR_READ_CONFIG, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return config, log.Err(codes.ERR_PARSE_CONFIG, err)
	}

	if err := log.Configure(config.Log); err != nil {
		return config, err
	}

	if err := validation.Configure(config.Validation); err != nil {
		return config, err
	}

	if err := storage.Configure(config.Storage); err != nil {
		return config, err
	}

	return config, nil
}
