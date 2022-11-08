package loginLogs

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/uuid"
	"src.goblgobl.com/utils/validation"

	_ "src.goblgobl.com/authen/tests"
)

var (
	createValidation = validation.Input().
				Field(userIdValidation).
				Field(validation.Int("status"))

	resMax              = http.StaticError(400, codes.RES_LOGIN_LOG_MAX, "maximum number of login logs reached")
	resMaxPayloadLength = http.StaticError(400, codes.RES_LOGIN_LOG_MAX_META_LENGTH, "payload length is exceeds maximum allowed size")
)

func Create(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
	body := conn.PostBody()
	input, err := typed.Json(body)
	if err != nil {
		return http.InvalidJSON, nil
	}

	validator := env.Validator
	if !createValidation.Validate(input, validator) {
		return http.Validation(validator), nil
	}

	project := env.Project
	var payload []byte
	if p, ok := input["payload"]; ok {
		mb, err := json.Marshal(p)
		if err != nil {
			// since this unmarshal'd, it should marshal, this is weird
			// body could have sensitive information...but we do currently store the
			// payload in plain text. Still not great, but this shouldnt' happen and
			// if it does, I really want to understand what's goin gon.
			log.Error("login_log_create_payload").Err(err).String("body", string(body)).Log()
			return nil, err
		}
		if m := project.LoginLogMaxPayloadLength; m > 0 && len(mb) > m {
			return resMaxPayloadLength, nil
		}
		payload = mb
	}

	id := uuid.String()
	result, err := storage.DB.LoginLogCreate(data.LoginLogCreate{
		Id:        id,
		Payload:   payload,
		ProjectId: project.Id,
		Status:    input.Int("status"),
		UserId:    input.String("user_id"),
		Max:       project.LoginLogMax,
	})
	if err != nil {
		return nil, err
	}

	if result.Status == data.LOGIN_LOG_CREATE_MAX {
		return resMax, nil
	}

	return http.Ok(struct {
		Id string `json:"id"`
	}{
		Id: id,
	}), nil
}
