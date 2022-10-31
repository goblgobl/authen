package data

import "time"

type TOTPGetStatus int
type TOTPCreateStatus int

const (
	TOTP_CREATE_OK TOTPCreateStatus = iota
	TOTP_CREATE_MAX

	TOTP_GET_OK TOTPGetStatus = iota
	TOTP_GET_NOT_FOUND
)

type TOTPCreate struct {
	Max       int
	ProjectId string
	UserId    string
	Type      string
	Secret    []byte
	Expires   *time.Time
}

type TOTPCreateResult struct {
	Status TOTPCreateStatus
}

type TOTPGet struct {
	ProjectId string
	UserId    string
	Type      string
	Pending   bool
	AllTypes  bool
}

type TOTPGetResult struct {
	Status TOTPGetStatus
	Secret []byte
}
