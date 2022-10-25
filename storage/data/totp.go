package data

import "time"

type GetTOTPStatus int
type CreateTOTPStatus int

const (
	CREATE_TOTP_OK CreateTOTPStatus = iota
	CREATE_TOTP_MAX

	GET_TOTP_OK GetTOTPStatus = iota
	GET_TOTP_NOT_FOUND
)

type CreateTOTP struct {
	Max       uint32
	ProjectId string
	UserId    string
	Type      string
	Secret    []byte
	Expires   *time.Time
}

type CreateTOTPResult struct {
	Status CreateTOTPStatus
}

type GetTOTP struct {
	ProjectId string
	UserId    string
	Type      string
	Pending   bool
}

type GetTOTPResult struct {
	Status GetTOTPStatus
	Secret []byte
}
