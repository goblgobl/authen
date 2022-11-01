package misc

import (
	"fmt"

	"github.com/valyala/fasthttp"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/utils/http"
)

func Ping(conn *fasthttp.RequestCtx) (http.Response, error) {
	if err := storage.DB.Ping(); err != nil {
		return nil, fmt.Errorf("ping store - %w", err)
	}

	return http.OkBytes([]byte(`{"ok":true}`)), nil
}
