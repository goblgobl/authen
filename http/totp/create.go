package totp

import (
	"encoding/base64"
	"time"

	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/utils/encryption"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"

	qrcode "github.com/skip2/go-qrcode"
	"github.com/valyala/fasthttp"
	"github.com/xlzd/gotp"
)

var (
	createValidation = validation.Input().
				Field(idValidation).
				Field(keyValidation).
				Field(typeValidation).
				Field(validation.String("account").Required().Length(1, 100)).
				Field(validation.String("issuer").Length(1, 100))

	resMax = http.StaticError(400, codes.RES_TOTP_MAX, "maximum number of TOTPs reached")
)

func Create(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
	input, err := typed.Json(conn.PostBody())
	if err != nil {
		return http.InvalidJSON, nil
	}

	validator := env.Validator
	if !createValidation.Validate(input, validator) {
		return http.Validation(validator), nil
	}

	issuer := input.String("issuer")
	if issuer == "" {
		issuer = env.Project.Issuer
	}
	account := input.String("account")

	secret := gotp.RandomSecret(int(authen.Config.TOTP.SecretLength))
	url := gotp.NewDefaultTOTP(secret).ProvisioningUri(account, issuer)

	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		return nil, err
	}

	key := *(*[32]byte)(input.Bytes("key"))
	encrypted, err := encryption.Encrypt(key, secret)
	if err != nil {
		return nil, err
	}

	project := env.Project
	expires := time.Now().Add(project.TOTPSetupTTL)
	result, err := storage.DB.CreateTOTP(data.CreateTOTP{
		Secret:    encrypted,
		UserId:    input.String("id"),
		Type:      input.String("type"),
		Expires:   &expires,
		ProjectId: project.Id,
		Max:       project.TOTPMax,
	})
	if err != nil {
		return nil, err
	}

	if result.Status == data.CREATE_TOTP_MAX {
		return resMax, nil
	}

	return http.Ok(struct {
		QR     string `json:"qr"`
		Secret string `json:"secret"`
	}{
		Secret: secret,
		QR:     base64.RawStdEncoding.EncodeToString(png),
	}), nil
}
