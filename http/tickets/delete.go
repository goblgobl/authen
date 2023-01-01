package tickets

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
	deleteValidation = validation.Object().Field("ticket", ticketValidation)
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

	project := env.Project
	res, err := storage.DB.TicketDelete(data.TicketUse{
		Ticket:    input.Bytes("ticket"),
		ProjectId: project.Id,
	})

	if err != nil {
		return nil, err
	}

	var deleted int
	if res.Status == data.TICKET_USE_OK {
		deleted = 1
	}

	return http.Ok(struct {
		Uses    *int `json:"uses"`
		Deleted int  `json:"deleted"`
	}{
		Uses:    res.Uses,
		Deleted: deleted,
	}), nil
}
