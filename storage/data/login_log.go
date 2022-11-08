package data

import "time"

type LoginLogGetStatus int
type LoginLogCreateStatus int

const (
	LOGIN_LOG_CREATE_OK LoginLogCreateStatus = iota
	LOGIN_LOG_CREATE_MAX

	LOGIN_LOG_GET_OK LoginLogGetStatus = iota
)

type LoginLogCreate struct {
	Id        string
	Max       int
	ProjectId string
	UserId    string
	Status    int
	Payload   []byte
}

type LoginLogCreateResult struct {
	Status LoginLogCreateStatus
}

type LoginLogGet struct {
	UserId    string
	ProjectId string
	Limit     int
	Offset    int
}

type LoginLogRecord struct {
	Id      string    `json:"id"`
	Status  int       `json:"status"`
	Payload any       `json:"payload",omitempty`
	Created time.Time `json:"created"`
}

type LoginLogGetResult struct {
	Status  LoginLogGetStatus
	Records []LoginLogRecord
}
