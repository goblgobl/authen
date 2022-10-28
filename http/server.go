package http

import (
	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/config"
	"src.goblgobl.com/authen/http/misc"
	"src.goblgobl.com/authen/http/totp"
	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/utils"
	"src.goblgobl.com/utils/http"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"src.goblgobl.com/utils/log"
)

var (
	resNotFoundPath         = http.StaticNotFound(codes.RES_UNKNOWN_ROUTE)
	resMissingProjectHeader = http.StaticError(400, codes.RES_MISSING_PROJECT_HEADER, "Gobl-Project header required")
	resProjectNotFound      = http.StaticError(400, codes.RES_PROJECT_NOT_FOUND, "unknown project id")
)

func Listen() {
	listen := authen.Config.HTTP.Listen
	if listen == "" {
		listen = "127.0.0.1:5200"
	}

	log.Info("server_listening").String("address", listen).Log()

	fast := fasthttp.Server{
		Handler:                      handler(),
		NoDefaultContentType:         true,
		NoDefaultServerHeader:        true,
		SecureErrorLogMessage:        true,
		DisablePreParseMultipartForm: true,
	}
	err := fast.ListenAndServe(listen)
	log.Fatal("http_server_error").Err(err).String("address", listen).Log()
}

func handler() func(ctx *fasthttp.RequestCtx) {
	r := router.New()
	// misc routes
	r.GET("/v1/ping", misc.Ping)
	r.GET("/v1/info", misc.Info)

	envLoader := loadMultiTenancyEnv
	if totp := authen.Config.TOTP; totp != nil {
		envLoader = createSingleTenancyLoader(totp)
	}

	r.POST("/v1/totp", http.Handler("totp_create", envLoader, totp.Create))
	r.POST("/v1/totp/verify", http.Handler("totp_verify", envLoader, totp.Verify))
	r.POST("/v1/totp/confirm", http.Handler("totp_confirm", envLoader, totp.Confirm))
	r.POST("/v1/totp/delete", http.Handler("totp_delete", envLoader, totp.Delete))
	r.POST("/v1/totp/change_key", http.Handler("totp_change_key", envLoader, totp.ChangeKey))

	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		resNotFoundPath.Write(ctx)
	}

	return r.Handler
}

func loadMultiTenancyEnv(conn *fasthttp.RequestCtx) (*authen.Env, bool) {
	projectId := conn.Request.Header.PeekBytes([]byte("Gobl-Project"))
	if projectId == nil {
		resMissingProjectHeader.Write(conn)
		return nil, false
	}
	projectIdString := utils.B2S(projectId)
	project, err := authen.Projects.Get(projectIdString)

	if err != nil {
		log.Error("env_handler_projects_get").
			String("pid", projectIdString).
			Err(err).
			Log()
		http.GenericServerError.Write(conn)
		return nil, false
	}

	if project == nil {
		resProjectNotFound.Write(conn)
		return nil, false
	}
	return authen.NewEnv(project), true
}

func createSingleTenancyLoader(config *config.TOTP) func(conn *fasthttp.RequestCtx) (*authen.Env, bool) {
	project := authen.NewProject(&data.Project{
		Id:               "00000000-00000000-00000000-00000000",
		TOTPMax:          config.Max,
		TOTPIssuer:       config.Issuer,
		TOTPSetupTTL:     config.SetupTTL,
		TOTPSecretLength: config.SecretLength,
	})

	return func(conn *fasthttp.RequestCtx) (*authen.Env, bool) {
		return authen.NewEnv(project), true
	}
}
