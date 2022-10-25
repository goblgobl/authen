package data

type Project struct {
	Id           string `json:"id"`
	Issuer       string `json:"issuer"`
	TOTPMax      uint32 `json:"totp_max"`
	TOTPSetupTTL uint32 `json:"totp_setup_ttl`
}
