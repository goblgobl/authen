package data

type Project struct {
	Id               string `json:"id"`
	TOTPMax          int    `json:"totp_max"`
	TOTPIssuer       string `json:"totp_issuer"`
	TOTPSetupTTL     int    `json:"totp_setup_ttl`
	TOTPSecretLength int    `json:"totp_secret_length"`
}
