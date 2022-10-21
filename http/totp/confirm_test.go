package totp

import (
	"strings"
	"testing"

	"github.com/xlzd/gotp"
	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/tests/request"
	"src.goblgobl.com/utils/encryption"
)

func Test_Confirm_InvalidBody(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body("nope").
		Post(Confirm).
		ExpectInvalid(2004)
}

func Test_Confirm_InvalidData(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body("{}").
		Post(Confirm).
		ExpectValidation("id", 1001, "key", 1001, "code", 1001)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"id":   "",
			"key":  "",
			"code": "",
		}).
		Post(Confirm).
		ExpectValidation("id", 1003, "key", 1003, "code", 1003)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"id":   strings.Repeat("a", 101),
			"key":  strings.Repeat("b", 33),
			"code": strings.Repeat("c", 7),
		}).
		Post(Confirm).
		ExpectValidation("id", 1003, "key", 1003, "code", 1003)

	// key has to be 32 exactly and code exactly 6,
	// so let's test under this also (previous test was 33 and 7)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"key":  strings.Repeat("b", 31),
			"code": strings.Repeat("z", 5),
		}).
		Post(Confirm).
		ExpectValidation("key", 1003, "code", 1003)
}

func Test_Confirm_UnknownId(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"id":   tests.String(1, 100),
			"key":  tests.String(32),
			"code": "123456",
		}).
		Post(Confirm).
		ExpectInvalid(101_006)
}

func Test_Confirm_WrongKey(t *testing.T) {
	env := authen.BuildEnv().Env()
	userId := tests.String(1, 100)
	tests.Factory.TOTPSetup.Insert("project_id", env.Project.Id, "user_id", userId, "secret", "a")
	request.ReqT(t, env).
		Body(map[string]any{
			"id":   userId,
			"code": "123456",
			"key":  tests.String(32),
		}).
		Post(Confirm).
		ExpectInvalid(101_007)
}

func Test_Confirm_WrongCode(t *testing.T) {
	env := authen.BuildEnv().Env()
	userId := tests.String(1, 100)

	key := tests.String(32)
	secret := gotp.RandomSecret(int(16))

	tests.Factory.TOTPSetup.Insert("project_id", env.Project.Id, "user_id", userId, "secret", secret, "key", key)
	request.ReqT(t, env).
		Body(map[string]any{
			"id":   userId,
			"key":  key,
			"code": "123456",
		}).
		Post(Confirm).
		ExpectInvalid(101_008)
}

// do this twice, with the same user, to confirm that
// upsert works
func Test_Confirm_Success(t *testing.T) {
	env := authen.BuildEnv().Env()
	userId := tests.String(1, 100)

	key := tests.String(32)

	for i := 0; i < 2; i++ {
		secret := gotp.RandomSecret(int(16))
		totp := gotp.NewDefaultTOTP(secret)

		tests.Factory.TOTPSetup.Insert("project_id", env.Project.Id, "user_id", userId, "secret", secret, "key", key)
		request.ReqT(t, env).
			Body(map[string]any{
				"id":   userId,
				"key":  key,
				"code": totp.Now(),
			}).
			Post(Confirm).OK()

		row := tests.Row("select * from authen_totp_setups where user_id = $1", userId)
		assert.Nil(t, row)

		row = tests.Row("select * from authen_totps where user_id = $1", userId)
		assert.Nowish(t, row.Time("created"))
		assert.Equal(t, row.String("project_id"), env.Project.Id)

		dbSecret, err := encryption.Decrypt([]byte(key), row["nonce"].([]byte), row["secret"].([]byte))
		assert.Nil(t, err)
		assert.Equal(t, string(dbSecret), secret)
	}
}
