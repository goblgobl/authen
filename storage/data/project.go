package data

type Project struct {
	Id       string `json:"id"`
	Issuer   string `json:"issuer"`
	MaxUsers uint32 `json:"max_users"`
}
