package data

type Project struct {
	Id                     string `json:"id"`
	TOTPMax                int    `json:"totp_max"`
	TOTPIssuer             string `json:"totp_issuer"`
	TOTPSetupTTL           int    `json:"totp_setup_ttl`
	TOTPSecretLength       int    `json:"totp_secret_length"`
	TicketMax              int    `json:"ticket_max"`
	TicketMaxPayloadLength int    `json:"ticket_max_payload_length"`
	LoginLogMax            int    `json:"login_log_max"`
	LoginLogMaxMetaLength  int    `json:"login_log_max_meta_length"`
}
