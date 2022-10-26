package totp

import (
	"github.com/valyala/fasthttp"
	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"
)

var (
	deleteValidation = validation.Input().
		Field(typeValidation).
		Field(userIdValidation)
)

func Delete(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
	input, err := typed.Json(conn.PostBody())
	if err != nil {
		return http.InvalidJSON, nil
	}

	validator := env.Validator
	if !deleteValidation.Validate(input, validator) {
		return http.Validation(validator), nil
	}

	err = storage.DB.DeleteTOTP(data.GetTOTP{
		ProjectId: env.Project.Id,
		Type:      input.String("type"),
		UserId:    input.String("user_id"),
	})
	return resOK, err
}
