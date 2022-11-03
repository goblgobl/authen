package totps

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

func Test_ChangeKey_InvalidBody(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body("nope").
		Post(ChangeKey).
		ExpectInvalid(2003)
}

func Test_ChangeKey_InvalidData(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body("{}").
		Post(ChangeKey).
		ExpectValidation("user_id", 1001, "key", 1001, "new_key", 1001)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"user_id": "",
			"key":     "",
			"new_key": "",
		}).
		Post(ChangeKey).
		ExpectValidation("user_id", 1003, "key", 1003, "new_key", 1003)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"type":    strings.Repeat("a", 101),
			"user_id": strings.Repeat("a", 101),
			"key":     strings.Repeat("b", 33),
			"new_key": strings.Repeat("c", 33),
		}).
		Post(ChangeKey).
		ExpectValidation("type", 1003, "user_id", 1003, "key", 1003, "new_key", 1003)

	// key has to be 32 exactly  so let's test under this also
	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"key":     strings.Repeat("b", 31),
			"new_key": strings.Repeat("z", 31),
		}).
		Post(ChangeKey).
		ExpectValidation("key", 1003, "new_key", 1003)
}

func Test_ChangeKey_UnknownId(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"user_id": tests.String(1, 100),
			"key":     tests.HexKey(),
			"new_key": tests.HexKey(),
		}).
		Post(ChangeKey).
		ExpectInvalid(102_006)
}

func Test_ChangeKey_Fails_ForPending(t *testing.T) {
	env := authen.BuildEnv().Env()
	userId := tests.String(1, 100)

	key, hexKey := tests.Key()
	secret := gotp.RandomSecret(16)

	tests.Factory.TOTP.Insert("project_id", env.Project.Id, "user_id", userId, "secret", secret, "key", key, "pending", true)
	request.ReqT(t, env).
		Body(map[string]any{
			"user_id": userId,
			"key":     hexKey,
			"new_key": tests.HexKey(),
		}).
		Post(ChangeKey).
		ExpectInvalid(102_006)
}

func Test_ChangeKey_WrongKey(t *testing.T) {
	env := authen.BuildEnv().Env()
	userId := tests.String(1, 100)
	tests.Factory.TOTP.Insert("project_id", env.Project.Id, "user_id", userId, "secret", "a")
	request.ReqT(t, env).
		Body(map[string]any{
			"user_id": userId,
			"key":     tests.HexKey(),
			"new_key": tests.HexKey(),
		}).
		Post(ChangeKey).
		ExpectInvalid(102_007)
}

func Test_ChangeKey(t *testing.T) {
	env := authen.BuildEnv().Env()

	for _, tpe := range []string{"", "txp1"} {
		key, hexKey := tests.Key()
		userId := tests.String(1, 100)
		newKey, newHexKey := tests.Key()
		secret := gotp.RandomSecret(16)

		tests.Factory.TOTP.Insert("project_id", env.Project.Id, "user_id", userId, "type", tpe, "secret", secret, "key", key)
		request.ReqT(t, env).
			Body(map[string]any{
				"user_id": userId,
				"type":    tpe,
				"key":     hexKey,
				"new_key": newHexKey,
			}).
			Post(ChangeKey).OK()

		rows := tests.Rows("select * from authen_totps where user_id = $1", userId)
		assert.Equal(t, len(rows), 1) // only 1, as it replaced the existing

		row := rows[0]
		assert.Equal(t, row.Bool("pending"), false)
		assert.Equal(t, row.String("type"), tpe)
		assert.Equal(t, row.String("project_id"), env.Project.Id)
		dbSecret, ok := encryption.Decrypt(newKey, row.Bytes("secret"))
		assert.True(t, ok)
		assert.Equal(t, string(dbSecret), secret)
	}
}
