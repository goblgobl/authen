package totp

import (
	"time"

	"github.com/valyala/fasthttp"
	"github.com/xlzd/gotp"
	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/utils"
	"src.goblgobl.com/utils/encryption"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"
)

var (
	verifyValidation = validation.Input().
		Field(idValidation).
		Field(keyValidation).
		Field(typeValidation).
		Field(codeValidation)
)

func Verify(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
	input, err := typed.Json(conn.PostBody())
	if err != nil {
		return http.InvalidJSON, nil
	}

	validator := env.Validator
	if !verifyValidation.Validate(input, validator) {
		return http.Validation(validator), nil
	}

	project := env.Project
	result, err := storage.DB.GetTOTP(data.GetTOTP{
		Type:      input.String("type"),
		UserId:    input.String("id"),
		Pending:   false,
		ProjectId: project.Id,
	})
	if err != nil {
		return nil, err
	}
	if result.Status == data.GET_TOTP_NOT_FOUND {
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
	return resOK, nil
}
