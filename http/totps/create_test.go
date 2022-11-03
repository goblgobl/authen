package totps

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/tests/request"
	"src.goblgobl.com/utils/encryption"
)

func Test_Create_InvalidBody(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body("nope").
		Post(Create).
		ExpectInvalid(2003)
}

func Test_Create_InvalidData(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		Body("{}").
		Post(Create).
		ExpectValidation("user_id", 1001, "key", 1001, "account", 1001).
		ExpectNoValidation("issuer")

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"user_id": "",
			"key":     "",
			"account": "",
		}).
		Post(Create).
		ExpectValidation("user_id", 1003, "key", 1003, "account", 1003)

	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{
			"type":    strings.Repeat("a", 101),
			"user_id": strings.Repeat("a", 101),
			"key":     strings.Repeat("b", 33),
			"account": strings.Repeat("c", 101),
			"issuer":  strings.Repeat("d", 101),
		}).
		Post(Create).
		ExpectValidation("type", 1003, "user_id", 1003, "key", 1003, "account", 1003, "issuer", 1003)

	// key has to be 32 exactly, so let's test under this also
	// (previous test was 33)
	request.ReqT(t, authen.BuildEnv().Env()).
		Body(map[string]any{"key": strings.Repeat("b", 31)}).
		Post(Create).
		ExpectValidation("key", 1003)
}

// This is a big tests that does everything. We run it twice,
// for the same user, to make sure a given user can only have
// a single active totp setup
func Test_Create_TOTP_Without_Type(t *testing.T) {
	userId := tests.String(1, 100)
	env := authen.BuildEnv().Env()

	for i := 0; i < 2; i++ {
		key, hexKey := tests.Key()

		res := request.ReqT(t, env).
			Body(map[string]any{
				"key":     hexKey,
				"user_id": userId,
				"issuer":  "test-issuer",
				"account": "test-account",
			}).
			Post(Create).OK().JSON()

		secret := res.String("secret")
		assert.Equal(t, len(secret), 26)

		row := tests.Row("select * from authen_totps where user_id = $1", userId)
		assert.Equal(t, row.Bool("pending"), true)
		assert.Nowish(t, row.Time("created"))
		assert.Equal(t, row.String("type"), "")
		assert.Equal(t, row.String("project_id"), env.Project.Id)
		dbSecret, ok := encryption.Decrypt(key, row.Bytes("secret"))
		assert.True(t, ok)
		assert.Equal(t, string(dbSecret), secret)

		raw, err := base64.RawStdEncoding.DecodeString(res.String("qr"))
		assert.Nil(t, err)

		// if zbarimg isn't installed, we won't assert the QR code
		cmd := exec.Command("zbarimg", "--quiet", "-")
		cmd.Stdin = bytes.NewBuffer(raw)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		if err := cmd.Run(); err != nil {
			if !errors.Is(err, exec.ErrNotFound) {
				t.Errorf("zbarimg error output:\n%v", out.String())
				assert.Nil(t, err)
			}
			return
		}

		assert.Equal(t, out.String(), "QR-Code:otpauth://totp/test-issuer:test-account?issuer=test-issuer&secret="+secret+"\n")
	}
}

func Test_Create_TOTP_With_Type(t *testing.T) {
	userId := tests.String(1, 100)
	env := authen.BuildEnv().TOTPSetupTTL(22).Env()

	for i := 0; i < 2; i++ {

		res := request.ReqT(t, env).
			Body(map[string]any{
				"key":     tests.HexKey(),
				"user_id": userId,
				"type":    "t1",
				"issuer":  "test-issuer",
				"account": "test-account",
			}).
			Post(Create).OK().JSON()

		secret := res.String("secret")
		assert.Equal(t, len(secret), 26)

		row := tests.Row("select * from authen_totps where user_id = $1", userId)
		assert.Equal(t, row.String("type"), "t1")
	}
}

func Test_TOTP_CREATE_MAXTOTPs(t *testing.T) {
	projectId := tests.UUID()
	env := authen.BuildEnv().ProjectId(projectId).TOTPMax(2).Env()

	tests.Factory.TOTP.Insert("project_id", projectId)
	tests.Factory.TOTP.Insert("project_id", projectId)

	request.ReqT(t, env).
		Body(map[string]any{
			"issuer":  "test-issuer",
			"account": "test-account",
			"key":     tests.HexKey(),
			"user_id": tests.String(1, 100),
		}).
		Post(Create).ExpectInvalid(102_005)
}
