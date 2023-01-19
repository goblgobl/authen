package misc

import (
	"runtime"
	"testing"

	_ "src.goblgobl.com/authen/tests"
	"src.goblgobl.com/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/tests/request"
	"src.goblgobl.com/utils/log"
)

func Test_Info_Ok(t *testing.T) {
	conn := request.Req(t).Conn()
	res, err := Info(conn)
	assert.Nil(t, err)

	res.Write(conn, log.Noop{})
	body := request.Res(t, conn).OK().JSON()
	assert.Equal(t, body.String("commit"), commit)
	assert.Equal(t, body.String("go"), runtime.Version())
	assert.Equal(t, body.Object("storage").String("type"), tests.StorageType())
}
