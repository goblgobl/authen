package loginLogs

import (
	"github.com/valyala/fasthttp"
	"src.goblgobl.com/authen"
	"src.goblgobl.com/utils"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/validation"

	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/authen/storage/data"
	_ "src.goblgobl.com/authen/tests"
)

var (
	listValidation = validation.Object().
		Field("user_id", userIdValidation).
		Field("page", validation.Int().Min(1).Default(1)).
		Field("perpage", validation.Int().Min(1).Max(100).Default(10))
)

func List(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
	validator := env.Validator
	input, ok := listValidation.ValidateArgs(conn.QueryArgs(), validator)
	if !ok {
		return http.Validation(validator), nil
	}

	limit, offset := utils.Paging(input.Int("perpage"), input.Int("page"), 10)

	res, err := storage.DB.LoginLogGet(data.LoginLogGet{
		Limit:     limit,
		Offset:    offset,
		ProjectId: env.Project.Id,
		UserId:    input.String("user_id"),
	})
	if err != nil {
		return nil, err
	}

	return http.Ok(struct {
		Results []data.LoginLogRecord `json:"results"`
	}{
		Results: res.Records,
	}), nil
}
