package totp

import (
	"encoding/hex"

	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"
)

var (
	idValidation   = validation.String("id").Required().Length(1, 100)
	codeValidation = validation.String("code").Required().Length(6, 6)
	typeValidation = validation.String("type").Length(1, 100)

	// 32 bits hex encoded
	keyValidation = validation.String("key").Required().Length(64, 64).
			Convert(func(field string, value string, _input typed.Typed, res *validation.Result) any {
			if key, err := hex.DecodeString(value); err == nil {
				return key
			}

			res.InvalidField(field, validation.Meta{
				Code:  codes.VAL_NON_HEX_KEY,
				Error: "key must be a 32-byte hex encoded value",
			}, nil)
			return nil
		})
)
