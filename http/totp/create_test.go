package totp

import (
	"strings"
	"testing"

	"src.goblgobl.com/authen"
	"src.goblgobl.com/tests/request"
)

func Test_Create_InvalidBody(t *testing.T) {
	request.ReqT(t, authen.TestEnv()).
		Body("nope").
		Post(Create).
		ExpectInvalid(2004)
}

func Test_Create_InvalidData(t *testing.T) {
	request.ReqT(t, authen.TestEnv()).
		Body("{}").
		Post(Create).
		ExpectValidation("id", 1001, "key", 1001, "account", 1001).
		ExpectNoValidation("issuer")

	request.ReqT(t, authen.TestEnv()).
		Body(map[string]any{
			"id":      "",
			"key":     "",
			"account": "",
		}).
		Post(Create).
		ExpectValidation("id", 1003, "key", 1003, "account", 1003)

	request.ReqT(t, authen.TestEnv()).
		Body(map[string]any{
			"id":      strings.Repeat("a", 101),
			"key":     strings.Repeat("b", 33),
			"account": strings.Repeat("c", 101),
			"issuer":  strings.Repeat("d", 101),
		}).
		Post(Create).
		ExpectValidation("id", 1003, "key", 1003, "account", 1003, "issuer", 1003)

	// key has to be 32 exactly, so let's test under this also
	// (previous test was 33)
	request.ReqT(t, authen.TestEnv()).
		Body(map[string]any{"key": strings.Repeat("b", 31)}).
		Post(Create).
		ExpectValidation("key", 1003)
}

func Test_Create_CreateRecord(t *testing.T) {
	// res := request.ReqT(t, authen.TestEnv()).
	// 	Body(map[string]any{
	// 		"id": ""
	// 	}).
	// 	Post(Create).OK().JSON()
	// 	fmt.Println(res)
}
