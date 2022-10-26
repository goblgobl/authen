package totp

import (
	"time"

	"github.com/valyala/fasthttp"
	"github.com/xlzd/gotp"
	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/utils"
	"src.goblgobl.com/utils/encryption"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"
)

var (
	confirmValidation = validation.Input().
				Field(keyValidation).
				Field(typeValidation).
				Field(codeValidation).
				Field(userIdValidation)

	resNotFound      = http.StaticError(400, codes.RES_TOTP_NOT_FOUND, "TOTP not found")
	resIncorrectKey  = http.StaticError(400, codes.RES_TOTP_INCORRECT_KEY, "key is not correct")
	resIncorrectCode = http.StaticError(400, codes.RES_TOTP_INCORRECT_CODE, "code is not correct")
	resOK            = http.Ok(nil)
)

func Confirm(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
	input, err := typed.Json(conn.PostBody())
	if err != nil {
		return http.InvalidJSON, nil
	}

	validator := env.Validator
	if !confirmValidation.Validate(input, validator) {
		return http.Validation(validator), nil
	}

	projectId := env.Project.Id
	tpe := input.String("type")
	userId := input.String("user_id")

	result, err := storage.DB.TOTPGet(data.TOTPGet{
		Type:      tpe,
		Pending:   true,
		UserId:    userId,
		ProjectId: projectId,
	})
	if err != nil {
		return nil, err
	}
	if result.Status == data.TOTP_GET_NOT_FOUND {
		return resNotFound, nil
	}

	encrypted := result.Secret
	key := *(*[32]byte)(input.Bytes("key"))
	secret, ok := encryption.Decrypt(key, encrypted)
	if !ok {
		return resIncorrectKey, nil
	}

	totp := gotp.NewDefaultTOTP(utils.B2S(secret))
	if !totp.VerifyTime(input.String("code"), time.Now()) {
		return resIncorrectCode, nil
	}

	_, err = storage.DB.TOTPCreate(data.TOTPCreate{
		UserId:    userId,
		Type:      tpe,
		ProjectId: projectId,
		Secret:    encrypted,
	})

	return resOK, err
}
