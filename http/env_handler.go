package http

/*
Creates an *authen.Env which is a project-specific context under which this
request will be processed. This wraps the endpoint action, injecting the env and
dealing with the response.

Two important part of handling the response is to deal with unhandled errors and
writing a log of the request/response.
*/

import (
	"time"

	"src.goblgobl.com/utils"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/uuid"

	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/codes"

	"github.com/valyala/fasthttp"
)

var (
	resEnvTimeout           = http.StaticUnavailableError(codes.RES_ENV_TIMEOUT)
	resMissingProjectHeader = http.StaticError(400, codes.RES_MISSING_PROJECT_HEADER, "Gobl-Project header required")
	resProjectNotFound      = http.StaticError(400, codes.RES_PROJECT_NOT_FOUND, "unknown project id")
)

// The action that we'll call is a standard fasthttp action plus our env.
type Next func(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error)

// The routeName is just used for logging, it gives a canonical name to every
// action, which means we won't have to parse/group raw URLs
func envHandler(routeName string, next Next) func(ctx *fasthttp.RequestCtx) {
	return func(conn *fasthttp.RequestCtx) {
		start := time.Now()

		project := loadProject(conn)
		if project == nil {
			// loadProject will write appropriate error
			return
		}

		env := authen.NewEnv(project)
		if env == nil {
			// only way we can't get an env is if the project's env pool
			// blocked for too long
			resEnvTimeout.Write(conn)
			return
		}
		defer env.Release()

		header := &conn.Response.Header
		header.SetContentTypeBytes([]byte("application/json"))
		header.SetBytesK([]byte("RequestId"), env.RequestId)

		r, err := next(conn, env)

		var logger log.Logger
		if err == nil {
			logger = env.Info("req")
		} else {
			// We could log the error directly in the req log
			// but this could contain sensitive or private information
			// (e.g. it could come from a 3rd party library, say, a failed
			// smtp request, and for all we know, it includes the email + smtp password)
			// So we log a separate env-free error (no pid or rid)
			// and tie this error to the req log via the errorId.
			errorId := uuid.String()
			header.SetBytesK([]byte("Error-Id"), errorId)

			logger = env.Error("env_handler_err").String("eid", errorId).Err(err)

			r = http.GenericServerError
		}

		r.Write(conn)

		r.EnhanceLog(logger).
			String("route", routeName).
			Int64("ms", time.Now().Sub(start).Milliseconds()).
			Log()
	}
}

func loadProject(conn *fasthttp.RequestCtx) *authen.Project {
	projectId := conn.Request.Header.PeekBytes([]byte("Gobl-Project"))
	if projectId == nil {
		resMissingProjectHeader.Write(conn)
		return nil
	}
	projectIdString := utils.B2S(projectId)
	project, err := authen.Projects.Get(projectIdString)

	if err != nil {
		log.Error("env_handler_projects_get").
			String("pid", projectIdString).
			Err(err).
			Log()
		http.GenericServerError.Write(conn)
		return nil
	}

	if project == nil {
		resProjectNotFound.Write(conn)
		return nil
	}

	return project
}
