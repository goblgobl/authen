//go:build !release

// Used as a factory for tests only

package authen

import (
	"time"

	"src.goblgobl.com/authen/config"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/uuid"
	"src.goblgobl.com/utils/validation"
)

func init() {
	// awful
	Config.TOTP = config.TOTP{SecretLength: 16}
}

type EnvBuilder struct {
	project   *Project
	logger    log.Logger
	validator *validation.Result
}

func BuildEnv() *EnvBuilder {
	project := &Project{Id: uuid.String()}
	return &EnvBuilder{
		project: project,
	}
}

func (eb *EnvBuilder) ProjectId(id string) *EnvBuilder {
	eb.project.Id = id
	return eb
}

func (eb *EnvBuilder) TOTPMax(max uint32) *EnvBuilder {
	eb.project.TOTPMax = max
	return eb
}

func (eb *EnvBuilder) TOTPSetupTTL(max uint32) *EnvBuilder {
	eb.project.TOTPSetupTTL = time.Duration(max) * time.Second
	return eb
}

func (eb *EnvBuilder) Env() *Env {
	project := eb.project
	if project == nil {
		project = &Project{
			Id: uuid.String(),
		}
	}

	logger := eb.logger
	if logger == nil {
		logger = log.Noop{}
	}

	validator := eb.validator
	if validator == nil {
		validator = validation.NewResult(10)
	}

	return &Env{
		Logger:    logger,
		Project:   project,
		Validator: validator,
	}
}
