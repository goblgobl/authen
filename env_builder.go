//go:build !release

// Used as a factory for tests only

package authen

import (
	"time"

	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/uuid"
	"src.goblgobl.com/utils/validation"
)

type EnvBuilder struct {
	project   *Project
	logger    log.Logger
	validator *validation.Result
}

func BuildEnv() *EnvBuilder {
	project := &Project{Id: uuid.String(), TOTPSecretLength: 16}
	return &EnvBuilder{
		project: project,
	}
}

func (eb *EnvBuilder) ProjectId(id string) *EnvBuilder {
	eb.project.Id = id
	return eb
}

func (eb *EnvBuilder) TOTPMax(max int) *EnvBuilder {
	eb.project.TOTPMax = max
	return eb
}

func (eb *EnvBuilder) TOTPSetupTTL(max int) *EnvBuilder {
	eb.project.TOTPSetupTTL = time.Duration(max) * time.Second
	return eb
}

func (eb *EnvBuilder) TicketMax(max int) *EnvBuilder {
	eb.project.TicketMax = max
	return eb
}

func (eb *EnvBuilder) TicketMaxPayloadLength(max int) *EnvBuilder {
	eb.project.TicketMaxPayloadLength = max
	return eb
}

func (eb *EnvBuilder) LoginLogMax(max int) *EnvBuilder {
	eb.project.LoginLogMax = max
	return eb
}

func (eb *EnvBuilder) LoginLogMaxPayloadLength(max int) *EnvBuilder {
	eb.project.LoginLogMaxPayloadLength = max
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
