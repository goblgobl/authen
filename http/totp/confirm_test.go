package totp

import (
	"strings"
	"testing"
	"time"

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
			"key":  tests.HexKey(),
			"code": "123456",
		}).
		Post(Confirm).
		ExpectInvalid(102_006)
}

func Test_Confirm_Expired(t *testing.T) {
	env := authen.BuildEnv().Env()
	userId := tests.String(1, 100)
	key, hexKey := tests.Key()
	secret := gotp.RandomSecret(int(16))
	totp := gotp.NewDefaultTOTP(secret)

	tests.Factory.TOTP.Insert("project_id", env.Project.Id, "user_id", userId, "secret", secret, "key", key, "pending", true, "expires", time.Now().Add(-time.Second))
	request.ReqT(t, env).
		Body(map[string]any{
			"id":   userId,
			"key":  hexKey,
			"code": totp.Now(),
		}).
		Post(Confirm).ExpectInvalid(102_006)
}

func Test_Confirm_WrongKey(t *testing.T) {
	env := authen.BuildEnv().Env()
	userId := tests.String(1, 100)
	tests.Factory.TOTP.Insert("project_id", env.Project.Id, "user_id", userId, "secret", "a", "pending", true, "expires", time.Now().Add(time.Minute))
	request.ReqT(t, env).
		Body(map[string]any{
			"id":   userId,
			"code": "123456",
			"key":  tests.HexKey(),
		}).
		Post(Confirm).
		ExpectInvalid(102_007)
}

func Test_Confirm_WrongCode(t *testing.T) {
	env := authen.BuildEnv().Env()
	userId := tests.String(1, 100)

	key, hexKey := tests.Key()
	secret := gotp.RandomSecret(int(16))

	tests.Factory.TOTP.Insert("project_id", env.Project.Id, "user_id", userId, "secret", secret, "key", key, "pending", true, "expires", time.Now().Add(time.Minute))
	request.ReqT(t, env).
		Body(map[string]any{
			"id":   userId,
			"key":  hexKey,
			"code": "123456",
		}).
		Post(Confirm).
		ExpectInvalid(102_008)
}

// do this twice, with the same user, to confirm that
// upsert works
func Test_Confirm_Without_Type(t *testing.T) {
	env := authen.BuildEnv().Env()
	userId := tests.String(1, 100)

	key, hexKey := tests.Key()
	for i := 0; i < 2; i++ {
		secret := gotp.RandomSecret(int(16))
		totp := gotp.NewDefaultTOTP(secret)

		tests.Factory.TOTP.Insert("project_id", env.Project.Id, "user_id", userId, "secret", secret, "key", key, "pending", true, "expires", time.Now().Add(time.Minute))
		request.ReqT(t, env).
			Body(map[string]any{
				"id":   userId,
				"key":  hexKey,
				"code": totp.Now(),
			}).
			Post(Confirm).OK()

		row := tests.Row("select * from authen_totps where user_id = $1 and pending", userId)
		assert.Nil(t, row)

		row = tests.Row("select * from authen_totps where user_id = $1", userId)
		assert.Nowish(t, row.Time("created"))
		assert.False(t, row.Bool("pending"))
		assert.Equal(t, row.String("type"), "")
		assert.Nil(t, row["expires"])
		assert.Equal(t, row.String("project_id"), env.Project.Id)

		dbSecret, ok := encryption.Decrypt(key, row.Bytes("secret"))
		assert.True(t, ok)
		assert.Equal(t, string(dbSecret), secret)
	}
}

// do this twice, with the same user, to confirm that
// upsert works
func Test_Confirm_With_Type(t *testing.T) {
	env := authen.BuildEnv().Env()
	userId := tests.String(1, 100)

	key, hexKey := tests.Key()
	for i := 0; i < 2; i++ {
		secret := gotp.RandomSecret(int(16))
		totp := gotp.NewDefaultTOTP(secret)

		tests.Factory.TOTP.Insert("project_id", env.Project.Id, "user_id", userId, "secret", secret, "type", "t1x", "key", key, "pending", true, "expires", time.Now().Add(time.Minute))
		request.ReqT(t, env).
			Body(map[string]any{
				"id":   userId,
				"key":  hexKey,
				"type": "t1x",
				"code": totp.Now(),
			}).
			Post(Confirm).OK()

		row := tests.Row("select * from authen_totps where user_id = $1 and pending", userId)
		assert.Nil(t, row)

		row = tests.Row("select * from authen_totps where user_id = $1", userId)
		assert.Nowish(t, row.Time("created"))
		assert.False(t, row.Bool("pending"))
		assert.Equal(t, row.String("type"), "t1x")
		assert.Nil(t, row["expires"])
		assert.Equal(t, row.String("project_id"), env.Project.Id)

		dbSecret, ok := encryption.Decrypt(key, row.Bytes("secret"))
		assert.True(t, ok)
		assert.Equal(t, string(dbSecret), secret)
	}
}
