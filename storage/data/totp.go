package data

import "src.goblgobl.com/utils/encryption"

type CreateTOTPStatus int

const (
	CREATE_TOTP_OK CreateTOTPStatus = iota
	CREATE_TOTP_MAX_USERS
)

type CreateTOTP struct {
	ProjectId string
	UserId    string
	MaxUsers  uint32
	Value     encryption.Value
}

type CreateTOTPResult struct {
	Status CreateTOTPStatus
}
