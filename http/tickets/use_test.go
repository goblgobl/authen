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

func Test_Use_InvalidBody(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body("nope").
		Post(Use).
		ExpectInvalid(2003)
}

func Test_Use_InvalidData(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Post(Use).
		ExpectValidation("ticket", 1001)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{"ticket": ""}).
		Post(Use).
		ExpectValidation("ticket", 1003)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{"ticket": "x"}).
		Post(Use).
		ExpectValidation("ticket", 101_002)
}

func Test_Use_Not_Found(t *testing.T) {
	projectId := tests.UUID()
	env := authen.BuildEnv().ProjectId(projectId).Env()

	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t1")
	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t2", "uses", 0)
	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t3", "expires", time.Now().Add(-time.Second))

	request.ReqT(t, env).
		Body(map[string]any{"ticket": "nope"}).
		Post(Use).
		ExpectNotFound(102_011)

	// wrong project
	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t1"))}).
		Post(Use).
		ExpectNotFound(102_011)

	// no more uses
	request.ReqT(t, env).
		Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t2"))}).
		Post(Use).
		ExpectNotFound(102_011)

	// expired
	request.ReqT(t, env).
		Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t3"))}).
		Post(Use).
		ExpectNotFound(102_011)
}

func Test_Use_Found(t *testing.T) {
	projectId := tests.UUID()
	env := authen.BuildEnv().ProjectId(projectId).Env()

	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t1")
	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t2", "uses", 3)
	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t3", "expires", time.Now().Add(time.Second*10))

	// unlimited use ticket!
	req := request.ReqT(t, env).Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t1"))})
	for i := 0; i < 100; i++ {
		json := req.Post(Use).OK().Json
		assert.Nil(t, json["uses"])
	}

	// 3 use ticket
	req = request.ReqT(t, env).Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t2"))})
	for i := 0; i < 3; i++ {
		json := req.Post(Use).OK().Json
		assert.Equal(t, json.Int("uses"), 2-i)
	}
	// no more!
	req.Post(Use).ExpectNotFound(102_011)

	// future expiration
	req = request.ReqT(t, env).Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t3"))})
	for i := 0; i < 10; i++ {
		req.Post(Use).OK()
	}
}

func Test_Use_Payload(t *testing.T) {
	projectId := tests.UUID()
	env := authen.BuildEnv().ProjectId(projectId).Env()

	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t1")
	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t2", "payload", 9001)
	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t3", "payload", "over 9000!!")
	tests.Factory.Ticket.Insert("project_id", projectId, "ticket", "t4", "payload", map[string]int{"over": 9000})
	// null payload
	{
		json := request.ReqT(t, env).
			Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t1"))}).
			Post(Use).
			OK().Json
		assert.Nil(t, json["payload"])
	}

	// int payload
	{
		json := request.ReqT(t, env).
			Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t2"))}).
			Post(Use).
			OK().Json
		assert.Equal(t, json.Int("payload"), 9001)
	}

	// string payload
	{

		json := request.ReqT(t, env).
			Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t3"))}).
			Post(Use).
			OK().Json
		assert.Equal(t, json.String("payload"), "over 9000!!")
	}

	// object payload
	{

		json := request.ReqT(t, env).
			Body(map[string]any{"ticket": base64.RawStdEncoding.EncodeToString([]byte("t4"))}).
			Post(Use).
			OK().Json
		assert.Equal(t, json.Object("payload").Int("over"), 9000)
	}

}
