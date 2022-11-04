package tickets

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
	"src.goblgobl.com/utils/validation"

	_ "src.goblgobl.com/authen/tests"
)

var (
	useValidation = validation.Input().Field(ticketValidation)
	resNotFound   = http.StaticError(404, codes.RES_TICKET_NOT_FOUND, "ticket not found")
)

func Use(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
	body := conn.PostBody()
	input, err := typed.Json(body)
	if err != nil {
		return http.InvalidJSON, nil
	}

	validator := env.Validator
	if !useValidation.Validate(input, validator) {
		return http.Validation(validator), nil
	}

	project := env.Project
	res, err := storage.DB.TicketUse(data.TicketUse{
		Ticket:    input.Bytes("ticket"),
		ProjectId: project.Id,
	})
	if err != nil {
		return nil, err
	}

	if res.Status == data.TICKET_USE_NOT_FOUND {
		return resNotFound, nil
	}

	var payload any
	if p := res.Payload; p != nil {
		if err := json.Unmarshal(*p, &payload); err != nil {
			// very weird, as we've been able to deal with this as json so far
			log.Error("ticket_use_payload").Err(err).String("body", string(body)).Log()
			return nil, err
		}
	}

	return http.Ok(struct {
		Uses    *int `json:"uses"`
		Payload any  `json:"payload"`
	}{
		Uses:    res.Uses,
		Payload: payload,
	}), nil
}
