package totp

import "src.goblgobl.com/utils/validation"

var (
	idValidation   = validation.String("id").Required().Length(1, 100)
	keyValidation  = validation.String("key").Required().Length(32, 32)
	codeValidation = validation.String("code").Required().Length(6, 6)
)
