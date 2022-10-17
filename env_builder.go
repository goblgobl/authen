//go:build !release

// Used as a factory for tests only

package authen

import (
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/validation"
)

func TestEnv() *Env {
	return &Env{
		Logger:    log.Noop{},
		Validator: validation.NewResult(10),
	}
}
