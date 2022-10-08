package misc

import (
	"testing"

	_ "src.goblgobl.com/authen/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/tests/request"
)

func Test_Ping_Ok(t *testing.T) {
	conn := request.Req(t).Conn()
	Ping(conn)
	res := request.Res(t, conn).OK()
	assert.Equal(t, res.Body, `{"ok":true}`)
}
