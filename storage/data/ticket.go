package data

import "time"

type TicketUseStatus int
type TicketCreateStatus int

const (
	TICKET_CREATE_OK TicketCreateStatus = iota
	TICKET_CREATE_MAX

	TICKET_USE_OK TicketUseStatus = iota
	TICKET_USE_NOT_FOUND
)

type TicketCreate struct {
	Max       int
	ProjectId string
	Ticket    []byte
	Payload   []byte
	Uses      *int
	Expires   *time.Time
}

type TicketCreateResult struct {
	Status TicketCreateStatus
}

type TicketUse struct {
	Ticket    []byte
	ProjectId string
}

type TicketUseResult struct {
	Status  TicketUseStatus
	Payload *[]byte
	Uses    *int
}
