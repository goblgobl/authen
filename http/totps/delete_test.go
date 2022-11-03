package totps

import (
	"strings"
	"testing"
	"time"

	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/tests/request"
)

func Test_Delete_InvalidBody(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body("nope").
		Post(Delete).
		ExpectInvalid(2003)
}

func Test_Delete_InvalidData(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body("{}").
		Post(Delete).
		ExpectValidation("user_id", 1001)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"user_id": "",
		}).
		Post(Delete).
		ExpectValidation("user_id", 1003)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"type":    strings.Repeat("a", 101),
			"user_id": strings.Repeat("a", 101),
		}).
		Post(Delete).
		ExpectValidation("user_id", 1003, "type", 1003)
}

func Test_Deletes_Specific_Type(t *testing.T) {
	now := time.Now()
	env := authen.BuildEnv().Env()
	projectId := env.Project.Id
	userId1, userId2 := tests.String(1, 100), tests.String(1, 100)

	tests.Factory.TOTP.Insert("project_id", projectId, "user_id", userId1, "type", "t1", "created", now.Add(time.Second*10))
	tests.Factory.TOTP.Insert("project_id", projectId, "user_id", userId1, "type", "t2", "created", now.Add(time.Second*20))
	tests.Factory.TOTP.Insert("project_id", projectId, "user_id", userId2, "type", "t1", "created", now.Add(time.Second*30))

	body := request.ReqT(t, env).
		Body(map[string]any{
			"type":    "t1",
			"user_id": userId1,
		}).Post(Delete).OK().Json

	assert.Equal(t, body.Int("deleted"), 1)

	rows := tests.Rows("select * from authen_totps where project_id = $1 order by created", projectId)
	assert.Equal(t, len(rows), 2)
	assert.Equal(t, rows[0].String("type"), "t2")
	assert.Equal(t, rows[0].String("user_id"), userId1)

	assert.Equal(t, rows[1].String("type"), "t1")
	assert.Equal(t, rows[1].String("user_id"), userId2)
}

func Test_Deletes_Specific_All_Types(t *testing.T) {
	env := authen.BuildEnv().Env()
	projectId := env.Project.Id
	userId1, userId2 := tests.String(1, 100), tests.String(1, 100)

	tests.Factory.TOTP.Insert("project_id", projectId, "user_id", userId1, "type", "t1")
	tests.Factory.TOTP.Insert("project_id", projectId, "user_id", userId1, "type", "t2")
	tests.Factory.TOTP.Insert("project_id", projectId, "user_id", userId2, "type", "t1")

	body := request.ReqT(t, env).
		Body(map[string]any{
			"user_id":   userId1,
			"all_types": true,
		}).Post(Delete).OK().Json

	assert.Equal(t, body.Int("deleted"), 2)

	rows := tests.Rows("select * from authen_totps where project_id = $1", projectId)
	assert.Equal(t, len(rows), 1)
	assert.Equal(t, rows[0].String("type"), "t1")
	assert.Equal(t, rows[0].String("user_id"), userId2)
}
