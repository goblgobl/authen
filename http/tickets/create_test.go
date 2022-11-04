package tickets

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

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
			"ttl":  "four",
			"uses": "five",
		}).
		Post(Create).
		ExpectValidation("ttl", 1005, "uses", 1005)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"ttl":  -1,
			"uses": -2,
		}).
		Post(Create).
		ExpectValidation("ttl", 1006, "uses", 1006)
}

func Test_Create_Minimal(t *testing.T) {
	env := authen.BuildEnv().Env()

	res := request.ReqT(t, env).
		Post(Create).
		OK().Json

	ticket, err := base64.RawStdEncoding.DecodeString(res.String("ticket"))
	assert.Nil(t, err)

	ticketHash := sha256.Sum256(ticket)

	row := tests.Row("select * from authen_tickets where project_id = $1 and ticket = $2", env.Project.Id, ticketHash[:])
	assert.Nil(t, row["payload"])
	assert.Timeish(t, row.Time("expires"), time.Now().Add(time.Minute))
	assert.Equal(t, row.Int("uses"), 1)
	assert.Nowish(t, row.Time("created"))
}

func Test_Create_Everything(t *testing.T) {
	env := authen.BuildEnv().Env()

	res := request.ReqT(t, env).
		Body(map[string]any{
			"uses":    4,
			"ttl":     120,
			"payload": map[string]int{"over": 9000},
		}).
		Post(Create).
		OK().Json

	ticket, err := base64.RawStdEncoding.DecodeString(res.String("ticket"))
	assert.Nil(t, err)
	ticketHash := sha256.Sum256(ticket)

	row := tests.Row("select * from authen_tickets where project_id = $1 and ticket = $2", env.Project.Id, ticketHash[:])
	assert.Bytes(t, row.Bytes("payload"), []byte(`{"over":9000}`))
	assert.Timeish(t, row.Time("expires"), time.Now().Add(time.Minute*2))
	assert.Equal(t, row.Int("uses"), 4)
	assert.Nowish(t, row.Time("created"))
}

func Test_Create_Max(t *testing.T) {
	projectId := tests.UUID()
	env := authen.BuildEnv().ProjectId(projectId).TicketMax(2).Env()
	tests.Factory.Ticket.Insert("project_id", projectId)
	tests.Factory.Ticket.Insert("project_id", projectId)

	request.ReqT(t, env).
		Post(Create).ExpectInvalid(102009)
}

func Test_Create_Payload_Length(t *testing.T) {
	env := authen.BuildEnv().TicketMaxPayloadLength(10).Env()

	request.ReqT(t, env).
		Body(map[string]any{
			"uses":    4,
			"ttl":     120,
			"payload": map[string]int{"over": 9000},
		}).
		Post(Create).
		ExpectInvalid(102_010)

	request.ReqT(t, env).
		Body(map[string]any{
			"uses":    4,
			"ttl":     120,
			"payload": map[string]int{"over": 9},
		}).
		Post(Create).OK()
}
