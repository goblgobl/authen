package misc

import (
	_ "embed"
	"runtime"

	"github.com/valyala/fasthttp"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/json"
	"src.goblgobl.com/utils/log"
)

//go:generate make commit.txt
//go:embed commit.txt
var commit string

func Info(conn *fasthttp.RequestCtx) {
	conn.SetContentTypeBytes([]byte("application/json"))

	storageInfo, err := storage.DB.Info()

	if err != nil {
		log.Error("storage_info	").Err(err).Log()
		http.GenericServerError.Write(conn)
		return
	}

	data, err := json.Marshal(struct {
		Go      string `json:"go"`
		Commit  string `json:"commit"`
		Storage any    `json:"storage"`
	}{
		Commit:  commit,
		Go:      runtime.Version(),
		Storage: storageInfo,
	})

	if err != nil {
		log.Error("info_serialize").Err(err).Log()
		http.GenericServerError.Write(conn)
	} else {
		conn.SetStatusCode(200)
		conn.Response.SetBody(data)
	}
}
