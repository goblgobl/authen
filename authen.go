package authen

import (
	"src.goblgobl.com/authen/config"
)

var InstanceId uint8

func Init(config config.Config) error {
	InstanceId = config.InstanceId
	return nil
}
