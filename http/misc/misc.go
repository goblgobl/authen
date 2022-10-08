package misc

import (
	"github.com/valyala/fasthttp"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/log"
)

func Ping(conn *fasthttp.RequestCtx) {
	conn.SetContentTypeBytes([]byte("application/json"))

	if err := storage.DB.Ping(); err != nil {
		log.Error("ping_store").Err(err).Log()
		http.GenericServerError.Write(conn)
	} else {
		conn.SetStatusCode(200)
		conn.Response.SetBody([]byte(`{"ok":true}`))
	}
}
