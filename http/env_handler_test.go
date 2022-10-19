package http

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
	"src.goblgobl.com/authen"
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

func Test_EnvHandler_Missing_Project_Header(t *testing.T) {
	conn := request.Req(t).Conn()
	envHandler("", func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		assert.Fail(t, "next should not be called")
		return nil, nil
	})(conn)
	request.Res(t, conn).ExpectInvalid(101002)
}

func Test_EnvHandler_Unknown_Project(t *testing.T) {
	conn := request.Req(t).ProjectId("6429C13A-DBB2-4FF2-ADDA-571C601B91E6").Conn()
	envHandler("", func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		assert.Fail(t, "next should not be called")
		return nil, nil
	})(conn)
	request.Res(t, conn).ExpectInvalid(101003)
}

func Test_EnvHandler_CallsHandlerWithProject(t *testing.T) {
	conn := request.Req(t).ProjectId(projectId).Conn()
	envHandler("", func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		assert.Equal(t, env.Project.Id, projectId)
		return http.Ok(map[string]int{"over": 9000}), nil
	})(conn)

	res := request.Res(t, conn).OK()
	assert.Equal(t, res.Json.Int("over"), 9000)
}

func Test_EnvHandler_RequestId(t *testing.T) {
	conn := request.Req(t).ProjectId(projectId).Conn()

	var id1, id2 string
	envHandler("", func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		id1 = env.RequestId
		return http.Ok(nil), nil
	})(conn)

	envHandler("", func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		id2 = env.RequestId
		return http.Ok(nil), nil
	})(conn)

	assert.Equal(t, len(id1), 8)
	assert.Equal(t, len(id2), 8)
	assert.NotEqual(t, id1, id2)
}

func Test_EnvHandler_LogsResponse(t *testing.T) {
	out := &strings.Builder{}
	var logger log.Logger
	defer func() {
		forceLoggerOut(logger, os.Stderr)
	}()

	var requestId string
	conn := request.Req(t).ProjectId(projectId).Conn()
	envHandler("test-route", func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		logger = env.Logger
		forceLoggerOut(logger, out)
		requestId = env.RequestId
		return http.StaticNotFound(9001), nil
	})(conn)

	reqLog := log.KvParse(out.String())
	assert.Equal(t, reqLog["pid"], projectId)
	assert.Equal(t, reqLog["rid"], requestId)
	assert.Equal(t, reqLog["l"], "info")
	assert.Equal(t, reqLog["status"], "404")
	assert.Equal(t, reqLog["route"], "test-route")
	assert.Equal(t, reqLog["res"], "33")
	assert.Equal(t, reqLog["code"], "9001")
	assert.Equal(t, reqLog["c"], "req")
}

func Test_EnvHandler_LogsError(t *testing.T) {
	out := &strings.Builder{}
	var logger log.Logger
	defer func() {
		forceLoggerOut(logger, os.Stderr)
	}()

	var requestId string
	conn := request.Req(t).ProjectId(projectId).Conn()
	envHandler("test2", func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
		logger = env.Logger
		forceLoggerOut(logger, out)
		requestId = env.RequestId
		return nil, errors.New("Not Over 9000!")
	})(conn)

	res := request.Res(t, conn).ExpectCode(2001)
	assert.Equal(t, res.Status, 500)
	reqLog := log.KvParse(out.String())
	assert.Equal(t, reqLog["pid"], projectId)
	assert.Equal(t, reqLog["rid"], requestId)
	assert.Equal(t, reqLog["l"], "error")
	assert.Equal(t, reqLog["status"], "500")
	assert.Equal(t, reqLog["route"], "test2")
	assert.Equal(t, reqLog["res"], "45")
	assert.Equal(t, reqLog["code"], "2001")
	assert.Equal(t, reqLog["c"], "env_handler_err")
	assert.Equal(t, reqLog["eid"], res.Headers["Error-Id"])
	assert.Equal(t, reqLog["err"], `"Not Over 9000!"`)
}

func forceLoggerOut(logger log.Logger, out io.Writer) {
	logger.(*log.KvLogger).SetOut(out)
}
