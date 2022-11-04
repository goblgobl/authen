package tickets

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"time"

	"github.com/valyala/fasthttp"
	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/codes"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/utils/http"
	"src.goblgobl.com/utils/json"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/validation"

	_ "src.goblgobl.com/authen/tests"
)

var (
	createValidation = validation.Input().
				Field(validation.Int("ttl").Min(0).Default(60)).
				Field(validation.Int("uses").Min(0).Default(1))

	resMax              = http.StaticError(400, codes.RES_TICKET_MAX, "maximum number of tickets reached")
	resMaxPayloadLength = http.StaticError(400, codes.RES_TICKET_MAX_PAYLOAD_LENGTH, "payload length is exceeds maximum allowed size")
)

func Create(conn *fasthttp.RequestCtx, env *authen.Env) (http.Response, error) {
	body := conn.PostBody()
	input, err := typed.Json(body)
	if err != nil {
		return http.InvalidJSON, nil
	}

	validator := env.Validator
	if !createValidation.Validate(input, validator) {
		return http.Validation(validator), nil
	}

	project := env.Project

	var payload []byte
	if p, ok := input["payload"]; ok {
		pb, err := json.Marshal(p)
		if err != nil {
			// since this unmarshal'd, it should marshal, this is weird
			// body could have sensitive information...but we do currently store the
			// payload in plain text. Still not great, but this shouldnt' happen and
			// if it does, I really want to understand what's goin gon.
			log.Error("ticket_create_payload").Err(err).String("body", string(body)).Log()
			return nil, err
		}
		if m := project.TicketMaxPayloadLength; m > 0 && len(pb) > m {
			return resMaxPayloadLength, nil
		}
		payload = pb
	}

	var uses *int
	if n, ok := input.IntIf("uses"); ok {
		uses = &n
	}

	var expires *time.Time
	if n, ok := input.IntIf("ttl"); ok {
		e := time.Now().Add(time.Duration(n) * time.Second)
		expires = &e
	}

	var ticket [20]byte
	ticketSlice := ticket[:]
	if _, err := io.ReadFull(rand.Reader, ticketSlice); err != nil {
		return nil, err
	}
	ticketHash := sha256.Sum256(ticketSlice)

	result, err := storage.DB.TicketCreate(data.TicketCreate{
		Uses:      uses,
		Payload:   payload,
		Expires:   expires,
		ProjectId: project.Id,
		Ticket:    ticketHash[:],
		Max:       project.TicketMax,
	})
	if err != nil {
		return nil, err
	}

	if result.Status == data.TICKET_CREATE_MAX {
		return resMax, nil
	}

	return http.Ok(struct {
		Ticket string `json:"ticket"`
	}{
		Ticket: base64.RawStdEncoding.EncodeToString(ticketSlice),
	}), nil
}
