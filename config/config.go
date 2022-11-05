package config

import (
	"os"

	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/utils/json"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/validation"
)

var (
	defaultDBCleanFrequency       = uint16(120)
	defaultProjectUpdateFrequency = uint16(120)
)

type Config struct {
	InstanceId             uint8             `json:"instance_id"`
	Migrations             *bool             `json:"migrations"`
	DBCleanFrequency       *uint16           `json:"db_clean_frequency"`
	ProjectUpdateFrequency *uint16           `json:"project_update_frequency"`
	MultiTenancy           bool              `json:"multi_tenancy"`
	HTTP                   HTTP              `json:"http"`
	TOTP                   *TOTP             `json:"totp"`
	Ticket                 *Ticket           `json:"ticket"`
	Log                    log.Config        `json:"log"`
	Storage                storage.Config    `json:"storage"`
	Validation             validation.Config `json:"validation"`
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

type Ticket struct {
	Max              int `json:"max"`
	MaxPayloadLength int `json:"max_payload_length"`
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

	ticket := config.Ticket
	if config.MultiTenancy && ticket != nil {
		log.Warn("multi_tenancy_ticket").String("details", "'ticket' configuration settings are ignored when multi_tenancy=true").Log()
	}
	if !config.MultiTenancy && ticket == nil {
		config.Ticket = new(Ticket)
	}

	if config.DBCleanFrequency == nil {
		config.DBCleanFrequency = &defaultDBCleanFrequency
	}

	if config.ProjectUpdateFrequency == nil {
		config.ProjectUpdateFrequency = &defaultProjectUpdateFrequency
	}

	return config, nil
}
