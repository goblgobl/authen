package data

import "src.goblgobl.com/utils/encryption"

type CreateTOTPStatus int
type GetTOTPSetupStatus int

const (
	CREATE_TOTP_OK CreateTOTPStatus = iota
	CREATE_TOTP_MAX_USERS

	GET_TOTP_SETUP_OK GetTOTPSetupStatus = iota
	GET_TOTP_SETUP_NOT_FOUND
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

type GetTOTPSetup struct {
	ProjectId string
	UserId    string
}

type GetTOTPSetupResult struct {
	Status GetTOTPSetupStatus
	Value  encryption.Value
}
