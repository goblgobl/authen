package totp

import (
	"github.com/valyala/fasthttp"
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
	changeKeyValidation = validation.Input().
		Field(typeValidation).
		Field(userIdValidation).
		Field(keyValidation).
		Field(newKeyValidation)
)

func ChangeKey(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
	input, err := typed.Json(conn.PostBody())
	if err != nil {
		return http.InvalidJSON, nil
	}

	validator := env.Validator
	if !changeKeyValidation.Validate(input, validator) {
		return http.Validation(validator), nil
	}

	tpe := input.String("type")
	userId := input.String("user_id")
	projectId := env.Project.Id
	result, err := storage.DB.TOTPGet(data.TOTPGet{
		Type:      tpe,
		UserId:    userId,
		Pending:   false,
		ProjectId: projectId,
	})

	if err != nil {
		return nil, err
	}
	if result.Status == data.TOTP_GET_NOT_FOUND {
		return resNotFound, nil
	}

	key := *(*[32]byte)(input.Bytes("key"))
	secret, ok := encryption.Decrypt(key, result.Secret)
	if !ok {
		return resIncorrectKey, nil
	}

	newKey := *(*[32]byte)(input.Bytes("new_key"))
	encrypted, err := encryption.Encrypt(newKey, utils.B2S(secret))
	if err != nil {
		return nil, err
	}

	_, err = storage.DB.TOTPCreate(data.TOTPCreate{
		UserId:    userId,
		Type:      tpe,
		ProjectId: projectId,
		Secret:    encrypted,
	})

	return resOK, err
}
