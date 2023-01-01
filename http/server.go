package http

import (
	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/config"
	"src.goblgobl.com/authen/http/loginLogs"
	"src.goblgobl.com/authen/http/misc"
	"src.goblgobl.com/authen/http/tickets"
	"src.goblgobl.com/authen/http/totps"
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
	r.GET("/v1/ping", http.NoEnvHandler("ping", misc.Ping))
	r.GET("/v1/info", http.NoEnvHandler("info", misc.Info))

	envLoader := loadMultiTenancyEnv
	if !authen.Config.MultiTenancy {
		envLoader = createSingleTenancyLoader(authen.Config)
	}

	// TOTP routes
	r.POST("/v1/totps", http.Handler("totp_create", envLoader, totps.Create))
	r.POST("/v1/totps/verify", http.Handler("totp_verify", envLoader, totps.Verify))
	r.POST("/v1/totps/delete", http.Handler("totp_delete", envLoader, totps.Delete))
	r.POST("/v1/totps/change_key", http.Handler("totp_change_key", envLoader, totps.ChangeKey))

	// Tickets routes
	r.POST("/v1/tickets", http.Handler("tickets_create", envLoader, tickets.Create))
	r.POST("/v1/tickets/use", http.Handler("tickets_use", envLoader, tickets.Use))
	r.POST("/v1/tickets/delete", http.Handler("tickets_delete", envLoader, tickets.Delete))

	r.GET("/v1/login_logs", http.Handler("login_logs_list", envLoader, loginLogs.List))
	r.POST("/v1/login_logs", http.Handler("login_logs_create", envLoader, loginLogs.Create))

	// catch all
	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		resNotFoundPath.Write(ctx)
	}

	return r.Handler
}

func loadMultiTenancyEnv(conn *fasthttp.RequestCtx) (*authen.Env, http.Response, error) {
	projectId := conn.Request.Header.PeekBytes([]byte("Project"))
	if projectId == nil {
		return nil, resMissingProjectHeader, nil
	}
	projectIdString := utils.B2S(projectId)
	project, err := authen.Projects.Get(projectIdString)

	if err != nil {
		return nil, nil, err
	}

	if project == nil {
		return nil, resProjectNotFound, nil
	}
	return authen.NewEnv(project), nil, nil
}

func createSingleTenancyLoader(config config.Config) func(conn *fasthttp.RequestCtx) (*authen.Env, http.Response, error) {
	totp := config.TOTP
	ticket := config.Ticket
	loginLog := config.LoginLog
	project := authen.NewProject(&data.Project{
		Id:                       "00000000-00000000-00000000-00000000",
		TOTPMax:                  totp.Max,
		TOTPIssuer:               totp.Issuer,
		TOTPSetupTTL:             totp.SetupTTL,
		TOTPSecretLength:         totp.SecretLength,
		TicketMax:                ticket.Max,
		TicketMaxPayloadLength:   ticket.MaxPayloadLength,
		LoginLogMax:              loginLog.Max,
		LoginLogMaxPayloadLength: loginLog.MaxPayloadLength,
	}, false)

	return func(conn *fasthttp.RequestCtx) (*authen.Env, http.Response, error) {
		return authen.NewEnv(project), nil, nil
	}
}
