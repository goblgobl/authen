package totps

import (
	"encoding/hex"

	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"
)

var (
	typeValidation   = validation.String().Length(0, 100)
	codeValidation   = validation.String().Required().Length(6, 6)
	userIdValidation = validation.String().Required().Length(1, 100)

	// 32 bits hex encoded
	keyValidation = validation.String().Required().Length(64, 64).Convert(validateKey)

	resNotFound      = http.StaticError(400, codes.RES_TOTP_NOT_FOUND, "TOTP not found")
	resIncorrectKey  = http.StaticError(400, codes.RES_TOTP_INCORRECT_KEY, "key is not correct")
	resIncorrectCode = http.StaticError(400, codes.RES_TOTP_INCORRECT_CODE, "code is not correct")
	resOK            = http.Ok(nil)
)

func validateKey(field validation.Field, value string, _object typed.Typed, _input typed.Typed, res *validation.Result) any {
	if key, err := hex.DecodeString(value); err == nil {
		return key
	}

	res.AddInvalidField(field, validation.Invalid{
		Code:  codes.VAL_NON_HEX_KEY,
		Error: "key must be a 32-byte hex encoded value",
	})
	return nil
}
