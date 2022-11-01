package misc

import (
	_ "embed"
	"fmt"
	"runtime"

	"github.com/valyala/fasthttp"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/utils/http"
)

//go:generate make commit.txt
//go:embed commit.txt
var commit string

func Info(conn *fasthttp.RequestCtx) (http.Response, error) {
	storageInfo, err := storage.DB.Info()
	if err != nil {
		return nil, fmt.Errorf("storage info - %w", err)
	}

	return http.Ok(struct {
		Go      string `json:"go"`
		Commit  string `json:"commit"`
		Storage any    `json:"storage"`
	}{
		Commit:  commit,
		Go:      runtime.Version(),
		Storage: storageInfo,
	}), nil
}
