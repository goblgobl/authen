package tickets

import (
	"crypto/sha256"
	"encoding/base64"

	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"
)

var (
	ticketValidation = validation.String("ticket").
		Required().Length(1, 200).Convert(decodeTicket)
)

func decodeTicket(field string, value string, _input typed.Typed, res *validation.Result) any {
	ticket, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		res.InvalidField(field, validation.Meta{
			Code:  codes.VAL_NON_BASE64_TICKET,
			Error: "Ticket must be a base64 encoded value",
		}, nil)
	}
	ticketHash := sha256.Sum256(ticket)
	return ticketHash[:]
}
