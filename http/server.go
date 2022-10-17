package http

import (
	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/config"
	"src.goblgobl.com/authen/http/misc"
	"src.goblgobl.com/authen/http/totp"
	"src.goblgobl.com/utils/http"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"src.goblgobl.com/utils/log"
)

var (
	resNotFoundPath = http.StaticNotFound(codes.RES_UNKNOWN_ROUTE)
)

func Listen(config config.HTTP) {
	listen := config.Listen
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

	r.POST("/v1/totp", envHandler("totp_create", totp.Create))

	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		resNotFoundPath.Write(ctx)
	}

	return r.Handler
}
