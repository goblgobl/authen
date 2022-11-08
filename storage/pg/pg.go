package pg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/pg"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/authen/storage/pg/migrations"
)

type Config struct {
	URL string `json:"url"`
}

type DB struct {
	pg.DB
	tpe string
}

func New(config Config, tpe string) (DB, error) {
	db, err := pg.New(config.URL)
	if err != nil {
		return DB{}, fmt.Errorf("PG.New - %w", err)
	}
	return DB{db, tpe}, nil
}

func (db DB) Ping() error {
	_, err := db.Exec(context.Background(), "select 1")
	if err != nil {
		return fmt.Errorf("PG.Ping - %w", err)
	}
	return nil
}

func (db DB) Clean() error {
	_, err := db.Exec(context.Background(), `
		delete from authen_totps
		where expires < now()
	`)
	if err != nil {
		return fmt.Errorf("PG.clean (totp) - %w", err)
	}

	_, err = db.Exec(context.Background(), `
		delete from authen_tickets
		where uses = 0 or expires < now()
	`)
	if err != nil {
		return fmt.Errorf("PG.clean (tickets) - %w", err)
	}

	return nil
}

func (db DB) EnsureMigrations() error {
	return migrations.Run(db.DB)
}

func (db DB) Info() (any, error) {
	migration, err := migrations.GetCurrent(db.DB)
	if err != nil {
		return nil, err
	}

	return struct {
		Type      string `json:"type"`
		Migration int    `json:"migration"`
	}{
		Type:      db.tpe,
		Migration: migration,
	}, nil
}

func (db DB) GetProject(id string) (*data.Project, error) {
	row := db.QueryRow(context.Background(), `
		select id,
			totp_issuer, totp_max, totp_setup_ttl, totp_secret_length,
			ticket_max, ticket_max_payload_length,
			login_log_max, login_log_max_payload_length
		from authen_projects
		where id = $1
	`, id)

	project, err := scanProject(row)
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("PG.GetProject - %w", err)
	}
	return project, nil
}

func (db DB) GetUpdatedProjects(timestamp time.Time) ([]*data.Project, error) {
	// Not sure fetching the count upfront really makes much sense.
	// But we do expect this to be 0 almost every time that it's called, so most
	// of the time we're going to be doing a single DB call (either to get the count
	// which returns 0, or to get an empty result set).
	count, err := pg.Scalar[int](db.DB, "select count(*) from authen_projects where updated > $1", timestamp)
	if count == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("PG.GetUpdatedProjects (count) - %w", err)
	}

	rows, err := db.Query(context.Background(), `
		select id,
			totp_issuer, totp_max, totp_setup_ttl, totp_secret_length,
			ticket_max, ticket_max_payload_length,
			login_log_max, login_log_max_payload_length
		from authen_projects where updated > $1
	`, timestamp)
	if err != nil {
		return nil, fmt.Errorf("PG.GetUpdatedProjects (select) - %w", err)
	}
	defer rows.Close()

	projects := make([]*data.Project, 0, count)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, rows.Err()
}

func (db DB) TOTPCreate(opts data.TOTPCreate) (data.TOTPCreateResult, error) {
	max := opts.Max
	tpe := opts.Type
	secret := opts.Secret
	userId := opts.UserId
	expires := opts.Expires
	pending := expires != nil
	projectId := opts.ProjectId

	var result data.TOTPCreateResult

	// Since we check first, then add the user (outside of a transaction)
	// concurrent calls to this might result in going a little over max
	// but I'm ok with that in the name of minimizing the DB calls
	// we need to make inside a transaction.
	canAdd, err := db.canAddTOTP(projectId, userId, tpe, max)
	if err != nil {
		return result, err
	}

	if !canAdd {
		result.Status = data.TOTP_CREATE_MAX
		return result, nil
	}

	err = db.Transaction(func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), `
			insert into authen_totps (project_id, user_id, type, pending, secret, expires)
			values ($1, $2, $3, $4, $5, $6)
			on conflict (project_id, user_id, type, pending) do update set secret = $5, expires = $6
		`, projectId, userId, tpe, pending, secret, expires)
		if err != nil {
			return fmt.Errorf("PG.TOTPCreate (upsert) - %w", err)
		}

		if pending {
			return nil
		}

		// We just inserted a non-pending TOTP, we should delete the
		// pending one for this user+type since it's now confirmed.
		// (Even though these are auto-cleaned up, keeping this around
		// longer than necessary would allow it to be re-used, which,
		// at the very least, is not expected.)

		_, err = tx.Exec(context.Background(), `
			delete from authen_totps
			where project_id = $1 and user_id = $2 and type = $3 and pending
		`, projectId, userId, tpe)

		if err != nil {
			return fmt.Errorf("PG.TOTPCreate (delete) - %w", err)
		}

		return nil
	})

	return result, err
}

func (db DB) TOTPGet(opts data.TOTPGet) (data.TOTPGetResult, error) {
	tpe := opts.Type
	userId := opts.UserId
	pending := opts.Pending
	projectId := opts.ProjectId
	var result data.TOTPGetResult

	row := db.QueryRow(context.Background(), `
		select secret
		from authen_totps
		where project_id = $1
			and user_id = $2
			and type = $3
			and pending = $4
			and (not pending or expires > now())
	`, projectId, userId, tpe, pending)

	var secret []byte
	if err := row.Scan(&secret); err != nil {
		if err == pg.ErrNoRows {
			result.Status = data.TOTP_GET_NOT_FOUND
			return result, nil
		}
		return result, fmt.Errorf("PG.TOTPGet - %w", err)
	}

	return data.TOTPGetResult{
		Secret: secret,
		Status: data.TOTP_GET_OK,
	}, nil
}

func (db DB) TOTPDelete(opts data.TOTPGet) (int, error) {
	tpe := opts.Type
	userId := opts.UserId
	allTypes := opts.AllTypes
	projectId := opts.ProjectId

	cmd, err := db.Exec(context.Background(), `
		delete from authen_totps
		where project_id = $1
			and user_id = $2
			and (type = $3 or $4)
	`, projectId, userId, tpe, allTypes)

	if err != nil {
		return 0, fmt.Errorf("PG.TOTPDelete - %w", err)
	}
	return int(cmd.RowsAffected()), nil
}

func (db DB) TicketCreate(opts data.TicketCreate) (data.TicketCreateResult, error) {
	max := opts.Max
	uses := opts.Uses
	ticket := opts.Ticket
	expires := opts.Expires
	payload := opts.Payload
	projectId := opts.ProjectId

	var result data.TicketCreateResult

	canAdd, err := db.ticketCanAdd(projectId, max)
	if err != nil {
		return result, err
	}

	if !canAdd {
		result.Status = data.TICKET_CREATE_MAX
		return result, nil
	}

	_, err = db.Exec(context.Background(), `
		insert into authen_tickets (project_id, ticket, expires, uses, payload)
		values ($1, $2, $3, $4, $5)
	`, projectId, ticket, expires, uses, payload)

	if err != nil {
		return result, fmt.Errorf("PG.TicketCreate - %w", err)
	}

	result.Status = data.TICKET_CREATE_OK
	return result, nil
}

func (db DB) TicketUse(opts data.TicketUse) (data.TicketUseResult, error) {
	ticket := opts.Ticket
	projectId := opts.ProjectId

	var result data.TicketUseResult

	row := db.QueryRow(context.Background(), `
		update authen_tickets
		set uses = uses - 1
		where project_id = $1
			and ticket = $2
			and (uses is null or uses > 0)
			and (expires is null or expires > now())
		returning uses, payload
	`, projectId, ticket)

	var uses *int
	var payload *[]byte
	if err := row.Scan(&uses, &payload); err != nil {
		if err == pg.ErrNoRows {
			result.Status = data.TICKET_USE_NOT_FOUND
			return result, nil
		}
		return result, fmt.Errorf("PG.TicketUse - %w", err)
	}

	result.Status = data.TICKET_USE_OK
	result.Payload = payload
	result.Uses = uses
	return result, nil
}

func (db DB) TicketDelete(opts data.TicketUse) (data.TicketUseResult, error) {
	ticket := opts.Ticket
	projectId := opts.ProjectId

	var result data.TicketUseResult

	row := db.QueryRow(context.Background(), `
		delete from authen_tickets
		where project_id = $1
			and ticket = $2
			and (uses is null or uses > 0)
			and (expires is null or expires > now())
		returning uses
	`, projectId, ticket)

	var uses *int
	if err := row.Scan(&uses); err != nil {
		if err == pg.ErrNoRows {
			result.Status = data.TICKET_USE_NOT_FOUND
			return result, nil
		}
		return result, fmt.Errorf("PG.TicketDelete - %w", err)
	}

	result.Status = data.TICKET_USE_OK
	result.Uses = uses
	return result, nil
}

func (db DB) LoginLogCreate(opts data.LoginLogCreate) (data.LoginLogCreateResult, error) {
	id := opts.Id
	max := opts.Max
	payload := opts.Payload
	userId := opts.UserId
	status := opts.Status
	projectId := opts.ProjectId

	var result data.LoginLogCreateResult

	canAdd, err := db.loginLogCanAdd(projectId, max)
	if err != nil {
		return result, err
	}

	if !canAdd {
		result.Status = data.LOGIN_LOG_CREATE_MAX
		return result, nil
	}

	_, err = db.Exec(context.Background(), `
		insert into authen_login_logs (id, project_id, user_id, status, payload)
		values ($1, $2, $3, $4, $5)
	`, id, projectId, userId, status, payload)

	if err != nil {
		return result, fmt.Errorf("PG.LoginLogCreate - %w", err)
	}

	result.Status = data.LOGIN_LOG_CREATE_OK
	return result, nil
}

func (db DB) LoginLogGet(opts data.LoginLogGet) (data.LoginLogGetResult, error) {
	userId := opts.UserId
	projectId := opts.ProjectId
	limit := opts.Limit
	offset := opts.Offset

	var result data.LoginLogGetResult

	rows, err := db.Query(context.Background(), `
		select id, status, payload, created
		from authen_login_logs
		where project_id = $1 and user_id = $2
		order by created desc
		limit $3 offset $4
	`, projectId, userId, limit, offset)

	if err != nil {
		return result, fmt.Errorf("PG.LoginLogGet (select) - %w", err)
	}
	defer rows.Close()

	i := 0
	records := make([]data.LoginLogRecord, limit)
	for rows.Next() {
		var payload any
		var payloadBytes *[]byte
		var record data.LoginLogRecord

		rows.Scan(&record.Id, &record.Status, &payloadBytes, &record.Created)
		if payloadBytes != nil {
			//rather not do this here, but it's better for our handler, so...
			if err := json.Unmarshal(*payloadBytes, &payload); err != nil {
				// very weird, as we've been able to deal with this as json so far
				log.Error("login_record_payload").Err(err).String("payload", string(*payloadBytes)).Log()
			}
			record.Payload = payload
		}
		records[i] = record
		i += 1
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("PG.LoginLogGet (scan) - %w", err)
	}

	result.Records = records[:i]
	result.Status = data.LOGIN_LOG_GET_OK
	return result, nil
}

func (db DB) canAddTOTP(projectId string, userId string, tpe string, max int) (bool, error) {
	// no limit
	if max == 0 {
		return true, nil
	}

	// if the user already exists, then we aren't adding a user
	// and thus cannot be over any limit
	exists, err := pg.Scalar[bool](db.DB, `
		select exists (
			select 1
			from authen_totps
			where project_id = $1 and user_id = $2 and type = $3
		)`, projectId, userId, tpe)

	if err != nil {
		return false, fmt.Errorf("PG.canAddTOTP (exists) - %w", err)
	}
	if exists {
		return exists, nil
	}

	count, err := pg.Scalar[int](db.DB, `
		select count(*)
		from authen_totps
		where project_id = $1
	`, projectId)

	if err != nil {
		return false, fmt.Errorf("PG.canAddTOTP (count) - %w", err)
	}
	return count < max, nil
}

func (db DB) ticketCanAdd(projectId string, max int) (bool, error) {
	// no limit
	if max == 0 {
		return true, nil
	}

	count, err := pg.Scalar[int](db.DB, `
		select count(*)
		from authen_tickets
		where project_id = $1
	`, projectId)

	if err != nil {
		return false, fmt.Errorf("PG.ticketCanAdd (count) - %w", err)
	}
	return count < max, nil
}

func (db DB) loginLogCanAdd(projectId string, max int) (bool, error) {
	// no limit
	if max == 0 {
		return true, nil
	}

	count, err := pg.Scalar[int](db.DB, `
		select count(*)
		from authen_login_logs
		where project_id = $1
	`, projectId)

	if err != nil {
		return false, fmt.Errorf("PG.loginLogCanAdd (count) - %w", err)
	}
	return count < max, nil
}

func scanProject(row pg.Row) (*data.Project, error) {
	var id, totpIssuer string
	var totpMax, totpSetupTTL, totpSecretLength int
	var ticketMax, ticketMaxPayloadLength int
	var loginLogMax, loginLogMaxPayloadLength int

	err := row.Scan(&id,
		&totpIssuer, &totpMax, &totpSetupTTL, &totpSecretLength,
		&ticketMax, &ticketMaxPayloadLength,
		&loginLogMax, &loginLogMaxPayloadLength)

	if err != nil {
		return nil, fmt.Errorf("PG.scanProject - %w", err)
	}

	return &data.Project{
		Id:                       id,
		TOTPMax:                  totpMax,
		TOTPIssuer:               totpIssuer,
		TOTPSetupTTL:             totpSetupTTL,
		TOTPSecretLength:         totpSecretLength,
		TicketMax:                ticketMax,
		TicketMaxPayloadLength:   ticketMaxPayloadLength,
		LoginLogMax:              loginLogMax,
		LoginLogMaxPayloadLength: loginLogMaxPayloadLength,
	}, nil
}
