package tickets

import (
	"encoding/base64"
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
		Post(Delete).
		ExpectValidation("ticket", 1001)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{"ticket": ""}).
		Post(Delete).
		ExpectValidation("ticket", 1003)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{"ticket": "x"}).
		Post(Delete).
		ExpectValidation("ticket", 101_002)
}

func Test_Delete_Not_Found(t *testing.T) {
	projectId := tests.UUID()
	env := authen.BuildEnv().ProjectId(projectId).Env()

	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t1")
	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t2", "uses", 0)
	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t3", "expires", time.Now().Add(-time.Second))

	json := request.ReqT(t, env).
		Body(map[string]any{"ticket": "nope"}).
		Post(Delete).OK().Json
	assert.Equal(t, json.Int("deleted"), 0)

	// wrong project
	json = request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t1"))}).
		Post(Delete).OK().Json
	assert.Equal(t, json.Int("deleted"), 0)

	// no more use
	json = request.ReqT(t, env).
		Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t2"))}).
		Post(Delete).OK().Json
	assert.Equal(t, json.Int("deleted"), 0)

	// expired
	json = request.ReqT(t, env).
		Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t3"))}).
		Post(Delete).OK().Json
	assert.Equal(t, json.Int("deleted"), 0)
}

func Test_Delete_Found(t *testing.T) {
	projectId := tests.UUID()
	env := authen.BuildEnv().ProjectId(projectId).Env()

	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t1")
	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t2", "uses", 3)
	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t3", "expires", time.Now().Add(time.Second*10))

	json := request.
		ReqT(t, env).Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t1"))}).
		Post(Delete).OK().Json
	assert.Nil(t, json["uses"])
	assert.Equal(t, json.Int("deleted"), 1)

	json = request.
		ReqT(t, env).Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t2"))}).
		Post(Delete).OK().Json
	assert.Equal(t, json.Int("uses"), 3)
	assert.Equal(t, json.Int("deleted"), 1)

	// future expiration
	json = request.
		ReqT(t, env).Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t3"))}).
		Post(Delete).OK().Json
	assert.Nil(t, json["uses"])
	assert.Equal(t, json.Int("deleted"), 1)

	// again, already deleted
	json = request.
		ReqT(t, env).Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t3"))}).
		Post(Delete).OK().Json
	assert.Nil(t, json["uses"])
	assert.Equal(t, json.Int("deleted"), 0)
}
