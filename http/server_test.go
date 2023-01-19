package http

import (
	"errors"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/config"
	"src.goblgobl.com/authen/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/tests/request"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/log"
)

var projectId string

func init() {
	projectId = tests.Factory.Project.Insert().String("id")
}

func Test_Server_MultiTenancy_Missing_Project_Header(t *testing.T) {
	conn := request.Req(t).Conn()
	http.Handler("", loadMultiTenancyEnv, func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		assert.Fail(t, "next should not be called")
		return nil, nil
	})(conn)
	request.Res(t, conn).ExpectInvalid(102002)
}

func Test_Server_MultiTenancy_Unknown_Project(t *testing.T) {
	conn := request.Req(t).ProjectId("6429C13A-DBB2-4FF2-ADDA-571C601B91E6").Conn()
	http.Handler("", loadMultiTenancyEnv, func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		assert.Fail(t, "next should not be called")
		return nil, nil
	})(conn)
	request.Res(t, conn).ExpectInvalid(102003)
}

func Test_Server_MultiTenancy_CallsHandlerWithProject(t *testing.T) {
	conn := request.Req(t).ProjectId(projectId).Conn()
	http.Handler("", loadMultiTenancyEnv, func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		assert.Equal(t, env.Project.Id, projectId)
		return http.Ok(map[string]int{"over": 9000}), nil
	})(conn)

	res := request.Res(t, conn).OK()
	assert.Equal(t, res.Json.Int("over"), 9000)
}

func Test_Server_MultiTenancy_RequestId(t *testing.T) {
	conn := request.Req(t).ProjectId(projectId).Conn()

	var id1, id2 string
	http.Handler("", loadMultiTenancyEnv, func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		id1 = env.RequestId()
		return http.Ok(nil), nil
	})(conn)

	http.Handler("", loadMultiTenancyEnv, func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		id2 = env.RequestId()
		return http.Ok(nil), nil
	})(conn)

	assert.Equal(t, len(id1), 8)
	assert.Equal(t, len(id2), 8)
	assert.NotEqual(t, id1, id2)
}

func Test_Server_MultiTenancy_LogsResponse(t *testing.T) {
	var requestId string
	conn := request.Req(t).ProjectId(projectId).Conn()

	logged := tests.CaptureLog(func() {
		http.Handler("test-route", loadMultiTenancyEnv, func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
			requestId = env.RequestId()
			return http.StaticNotFound(9001), nil
		})(conn)
	})

	reqLog := log.KvParse(logged)
	assert.Equal(t, reqLog["pid"], projectId)
	assert.Equal(t, reqLog["rid"], requestId)
	assert.Equal(t, reqLog["l"], "req")
	assert.Equal(t, reqLog["status"], "404")
	assert.Equal(t, reqLog["res"], "33")
	assert.Equal(t, reqLog["code"], "9001")
	assert.Equal(t, reqLog["c"], "test-route")
}

func Test_Server_MultiTenancy_LogsError(t *testing.T) {
	var requestId string
	conn := request.Req(t).ProjectId(projectId).Conn()
	logged := tests.CaptureLog(func() {
		http.Handler("test2", loadMultiTenancyEnv, func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
			requestId = env.RequestId()
			return nil, errors.New("Not Over 9000!")
		})(conn)
	})

	res := request.Res(t, conn).ExpectCode(2001)
	assert.Equal(t, res.Status, 500)

	errorId := res.Headers["Error-Id"]

	assert.Equal(t, len(errorId), 36)
	assert.Equal(t, res.Json.String("error_id"), errorId)

	reqLog := log.KvParse(logged)
	assert.Equal(t, reqLog["pid"], projectId)
	assert.Equal(t, reqLog["rid"], requestId)
	assert.Equal(t, reqLog["l"], "req")
	assert.Equal(t, reqLog["status"], "500")
	assert.Equal(t, reqLog["res"], "95")
	assert.Equal(t, reqLog["code"], "2001")
	assert.Equal(t, reqLog["c"], "test2")
	assert.Equal(t, reqLog["eid"], errorId)
	assert.Equal(t, reqLog["err"], `"Not Over 9000!"`)
}

func Test_Server_SingleTenancy_CallsHandlerWithProject(t *testing.T) {
	loader := createSingleTenancyLoader(config.Config{
		TOTP: &config.TOTP{
			Max:          1,
			Issuer:       "test-issuer",
			SetupTTL:     22,
			SecretLength: 8,
		},
		Ticket: &config.Ticket{
			Max:              99,
			MaxPayloadLength: 177,
		},
		LoginLog: &config.LoginLog{
			Max:              101,
			MaxPayloadLength: 202,
		},
	})

	conn := request.Req(t).Conn()
	http.Handler("", loader, func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		p := env.Project
		assert.Equal(t, p.Id, "00000000-00000000-00000000-00000000")
		assert.Equal(t, p.TOTPMax, 1)
		assert.Equal(t, p.TOTPIssuer, "test-issuer")
		assert.Equal(t, p.TOTPSecretLength, 8)
		assert.Equal(t, p.TOTPSetupTTL, time.Duration(22)*time.Second)
		assert.Equal(t, p.TicketMax, 99)
		assert.Equal(t, p.TicketMaxPayloadLength, 177)
		assert.Equal(t, p.LoginLogMax, 101)
		assert.Equal(t, p.LoginLogMaxPayloadLength, 202)

		return http.Ok(map[string]int{"over": 9001}), nil
	})(conn)

	res := request.Res(t, conn).OK()
	assert.Equal(t, res.Json.Int("over"), 9001)
}
