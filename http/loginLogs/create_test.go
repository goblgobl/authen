package loginLogs

import (
	"strings"
	"testing"

	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/tests/request"
)

func Test_Create_InvalidBody(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body("nope").
		Post(Create).
		ExpectInvalid(2003)
}

func Test_Create_InvalidData(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"user_id": 3,
			"status":  "nope",
		}).
		Post(Create).
		ExpectValidation("user_id", 1002, "status", 1005)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"user_id": strings.Repeat("a", 101),
		}).
		Post(Create).
		ExpectValidation("user_id", 1003)
}

func Test_Create_Max(t *testing.T) {
	projectId := tests.UUID()
	env := authen.BuildEnv().ProjectId(projectId).LoginLogMax(2).Env()
	tests.Factory.LoginLog.Insert("project_id", projectId)
	tests.Factory.LoginLog.Insert("project_id", projectId)

	request.ReqT(t, env).
		Body(map[string]any{
			"status":  1,
			"user_id": "1",
		}).
		Post(Create).ExpectInvalid(102012)
}

func Test_Create_Payload_Length(t *testing.T) {
	env := authen.BuildEnv().LoginLogMaxPayloadLength(10).Env()

	request.ReqT(t, env).
		Body(map[string]any{
			"user_id": "id",
			"status":  1,
			"payload": map[string]int{"over": 9000},
		}).
		Post(Create).
		ExpectInvalid(102_013)

	request.ReqT(t, env).
		Body(map[string]any{
			"user_id": "id",
			"status":  1,
			"payload": map[string]int{"over": 9},
		}).
		Post(Create).OK()
}

func Test_Create_NoPayload(t *testing.T) {
	env := authen.BuildEnv().Env()

	res := request.ReqT(t, env).
		Body(map[string]any{
			"status":  99,
			"user_id": "user_id_1",
		}).
		Post(Create).OK().Json

	id := res.String("id")
	assert.Equal(t, len(id), 36)

	row := tests.Row("select * from authen_login_logs where id = $1", id)
	assert.Nil(t, row["payload"])
	assert.Nowish(t, row.Time("created"))
	assert.Equal(t, row.Int("status"), 99)
	assert.Equal(t, row.String("user_id"), "user_id_1")
	assert.Equal(t, row.String("project_id"), env.Project.Id)
}
