package tickets

import (
	"crypto/sha256"
	"encoding/base64"

	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"
)

var (
	ticketValidation = validation.String().
				Required().Length(1, 200).Convert(decodeTicket)

	ttlValidation  = validation.Int().Min(0).Default(60)
	usesValidation = validation.Int().Min(0).Default(1)
)

func decodeTicket(field validation.Field, value string, _object typed.Typed, _input typed.Typed, res *validation.Result) any {
	ticket, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		res.AddInvalidField(field, validation.Invalid{
			Code:  codes.VAL_NON_BASE64_TICKET,
			Error: "Ticket must be a base64 encoded value",
		})
	}
	ticketHash := sha256.Sum256(ticket)
	return ticketHash[:]
}
