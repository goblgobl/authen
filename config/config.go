package config

import (
	"os"

	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/utils/json"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/validation"
)

type Config struct {
	InstanceId             uint8             `json:"instance_id"`
	MultiTenancy           bool              `json:"multi_tenancy"`
	HTTP                   HTTP              `json:"http"`
	TOTP                   *TOTP             `json:"totp"`
	Log                    log.Config        `json:"log"`
	Storage                storage.Config    `json:"storage"`
	Validation             validation.Config `json:"validation"`
	ProjectUpdateFrequency uint16            `json:"project_update_frequency"`
}

type HTTP struct {
	Listen string `json:"listen"`
}

type TOTP struct {
	Max          int    `json:"max"`
	Issuer       string `json:"issuer"`
	SetupTTL     int    `json:"setup_ttl"`
	SecretLength int    `json:"secret_length"`
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

	totp := config.TOTP
	if !config.MultiTenancy && (totp == nil || totp.Issuer == "") {
		return config, log.Errf(codes.ERR_MULTITENANCY_TOTP_CONFIG, "totp.issuer must be set")
	}

	if config.MultiTenancy && totp != nil {
		log.Warn("multi_tenancy_totp").String("details", "'totp' configuration settings are ignored when multi_tenancy=true").Log()
	}

	if totp != nil {
		if totp.SetupTTL == 0 {
			totp.SetupTTL = 300
		}
		if totp.SecretLength == 0 {
			totp.SecretLength = 16
		}
	}

	return config, nil
}
