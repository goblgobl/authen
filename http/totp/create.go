package totp

import (
	"encoding/base64"

	"src.goblgobl.com/authen"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"

	qrcode "github.com/skip2/go-qrcode"
	"github.com/valyala/fasthttp"
	"github.com/xlzd/gotp"
)

var (
	createValidation = validation.Input().
		Field(validation.String("id").Required().Length(1, 100)).
		Field(validation.String("key").Required().Length(32, 32)).
		Field(validation.String("account").Required().Length(1, 100)).
		Field(validation.String("issuer").Length(1, 100))
)

func Create(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
	data, err := typed.Json(conn.PostBody())
	if err != nil {
		return http.InvalidJSON, nil
	}

	validator := env.Validator
	if !createValidation.Validate(data, validator) {
		return http.Validation(validator), nil
	}

	account := data.String("account")
	issuer := data.String("issuer")
	if issuer == "" {
		issuer = env.Project.Issuer
	}
	secret := gotp.RandomSecret(int(authen.Config.TOTP.SecretLength))
	url := gotp.NewDefaultTOTP(secret).ProvisioningUri(account, issuer)

	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		return nil, err
	}

	// encrypted := encryption.Encrypt()

	return http.Ok(struct {
		Secret string `json:"secret"`
		QR     string `json:"qr"`
	}{
		Secret: secret,
		QR:     base64.RawStdEncoding.EncodeToString(png),
	}), nil
}
