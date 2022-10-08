package misc

import (
	"runtime"
	"testing"

	_ "src.goblgobl.com/authen/tests"
	"src.goblgobl.com/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/tests/request"
)

func Test_Info_Ok(t *testing.T) {
	conn := request.Req(t).Conn()
	Info(conn)
	res := request.Res(t, conn).OK().JSON()
	assert.Equal(t, res.String("commit"), commit)
	assert.Equal(t, res.String("go"), runtime.Version())
	assert.Equal(t, res.Object("storage").String("type"), tests.StorageType())
}
