package sqlite

import (
	"encoding/json"
	"fmt"
	"time"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/authen/storage/sqlite/migrations"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/sqlite"
)

type Config struct {
	Path string `json:"path"`
}

type Conn struct {
	sqlite.Conn
}

func New(config Config) (Conn, error) {
	conn, err := sqlite.New(config.Path, true)
	if err != nil {
		return Conn{}, fmt.Errorf("Sqlite.New - %w", err)
	}
	return Conn{conn}, nil
}

func (c Conn) Ping() error {
	err := c.Exec("select 1")
	if err != nil {
		return fmt.Errorf("Sqlite.Ping - %w", err)
	}
	return nil
}

func (c Conn) EnsureMigrations() error {
	return migrations.Run(c.Conn)
}

func (c Conn) Clean() error {
	err := c.Exec(`
		delete from authen_totps
		where expires < unixepoch()
	`)
	if err != nil {
		return fmt.Errorf("Sqlite.clean (totp) - %w", err)
	}

	err = c.Exec(`
		delete from authen_tickets
		where uses = 0 or expires < unixepoch()
	`)
	if err != nil {
		return fmt.Errorf("Sqlite.clean (tickets) - %w", err)
	}

	return nil
}

func (c Conn) Info() (any, error) {
	migration, err := sqlite.GetCurrentMigrationVersion(c.Conn)
	if err != nil {
		return nil, err
	}

	return struct {
		Type      string `json:"type"`
		Migration int    `json:"migration"`
	}{
		Type:      "sqlite",
		Migration: migration,
	}, nil
}

func (c Conn) GetProject(id string) (*data.Project, error) {
	row := c.Row(`
		select id,
			totp_issuer, totp_max, totp_setup_ttl, totp_secret_length,
			ticket_max, ticket_max_payload_length,
			login_log_max, login_log_max_payload_length
		from authen_projects
		where id = ?1
	`, id)

	project, err := scanProject(row)
	if err != nil {
		if err == sqlite.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("Sqlite.GetProject - %w", err)
	}
	return project, nil
}

func (c Conn) GetUpdatedProjects(timestamp time.Time) ([]*data.Project, error) {
	// Not sure fetching the count upfront really makes much sense.
	// But we do expect this to be 0 almost every time that it's called, so most
	// of the time we're going to be doing a single DB call (either to get the count
	// which returns 0, or to get an empty result set).
	count, err := sqlite.Scalar[int](c.Conn, "select count(*) from authen_projects where updated > ?1", timestamp)
	if err != nil {
		return nil, fmt.Errorf("Sqlite.GetUpdatedProjects (count) - %w", err)
	}
	if count == 0 {
		return nil, nil
	}

	rows := c.Rows(`
		select id,
			totp_issuer, totp_max, totp_setup_ttl, totp_secret_length,
			ticket_max, ticket_max_payload_length,
			login_log_max, login_log_max_payload_length
		from authen_projects
		where updated > ?1
	`, timestamp)
	defer rows.Close()

	projects := make([]*data.Project, 0, count)
	for rows.Next() {
		project, err := scanProject(&rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	if err := rows.Error(); err != nil {
		return nil, fmt.Errorf("Sqlite.GetUpdatedProjects (select) - %w", err)
	}

	return projects, nil
}

func (c Conn) TOTPCreate(opts data.TOTPCreate) (data.TOTPCreateResult, error) {
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
	canAdd, err := c.totpCanAdd(projectId, userId, tpe, max)
	if err != nil {
		return result, err
	}

	if !canAdd {
		result.Status = data.TOTP_CREATE_MAX
		return result, nil
	}

	err = c.Transaction(func() error {
		err := c.Exec(`
			insert or replace into authen_totps (project_id, user_id, type, pending, secret, expires)
			values (?1, ?2, ?3, ?4, ?5, ?6)
		`, projectId, userId, tpe, pending, secret, expires)

		if err != nil {
			return fmt.Errorf("Sqlite.TOTPCreate (upsert) - %w", err)
		}

		if pending {
			return nil
		}

		// We just inserted a non-pending TOTP, we should delete the
		// pending one for this user+type since it's now confirmed.
		// (Even though these are auto-cleaned up, keeping this around
		// longer than necessary would allow it to be re-used, which,
		// at the very least, is not expected.)
		err = c.Exec(`
			delete from authen_totps
			where project_id = ?1 and user_id = ?2 and type = ?3 and pending
		`, projectId, userId, tpe)

		if err != nil {
			return fmt.Errorf("Sqlite.TOTPCreate (delete) - %w", err)
		}

		return nil
	})

	result.Status = data.TOTP_CREATE_OK
	return result, err
}

func (c Conn) TOTPGet(opts data.TOTPGet) (data.TOTPGetResult, error) {
	tpe := opts.Type
	userId := opts.UserId
	pending := opts.Pending
	projectId := opts.ProjectId
	var result data.TOTPGetResult

	row := c.Row(`
		select secret
		from authen_totps
		where project_id = ?1
			and user_id = ?2
			and type = ?3
			and pending = ?4
			and (pending = 0 or expires > unixepoch())
	`, projectId, userId, tpe, pending)

	var secret []byte
	if err := row.Scan(&secret); err != nil {
		if err == sqlite.ErrNoRows {
			result.Status = data.TOTP_GET_NOT_FOUND
			return result, nil
		}
		return result, fmt.Errorf("Sqlite.TOTPGet - %w", err)
	}

	return data.TOTPGetResult{
		Secret: secret,
		Status: data.TOTP_GET_OK,
	}, nil
}

func (c Conn) TOTPDelete(opts data.TOTPGet) (int, error) {
	tpe := opts.Type
	userId := opts.UserId
	allTypes := opts.AllTypes
	projectId := opts.ProjectId

	err := c.Exec(`
		delete from authen_totps
		where project_id = ?1
			and user_id = ?2
			and (type = ?3 or ?4)
	`, projectId, userId, tpe, allTypes)

	if err != nil {
		return 0, fmt.Errorf("Sqlite.TOTPDelete - %w", err)
	}

	return c.Changes(), nil
}

func (c Conn) TicketCreate(opts data.TicketCreate) (data.TicketCreateResult, error) {
	max := opts.Max
	uses := opts.Uses
	ticket := opts.Ticket
	expires := opts.Expires
	payload := opts.Payload
	projectId := opts.ProjectId

	var result data.TicketCreateResult

	canAdd, err := c.ticketCanAdd(projectId, max)
	if err != nil {
		return result, err
	}

	if !canAdd {
		result.Status = data.TICKET_CREATE_MAX
		return result, nil
	}

	err = c.Exec(`
		insert into authen_tickets (project_id, ticket, expires, uses, payload)
		values (?1, ?2, ?3, ?4, ?5)
	`, projectId, ticket, expires, uses, payload)

	if err != nil {
		return result, fmt.Errorf("Sqlite.TicketCreate - %w", err)
	}

	result.Status = data.TICKET_CREATE_OK
	return result, nil
}

func (c Conn) TicketUse(opts data.TicketUse) (data.TicketUseResult, error) {
	ticket := opts.Ticket
	projectId := opts.ProjectId

	var result data.TicketUseResult

	row := c.Row(`
		update authen_tickets
		set uses = uses - 1
		where project_id = ?1
			and ticket = ?2
			and (uses is null or uses > 0)
			and (expires is null or expires > unixepoch())
		returning uses, payload
	`, projectId, ticket)

	var uses *int
	var payload *[]byte
	if err := row.Scan(&uses, &payload); err != nil {
		if err == sqlite.ErrNoRows {
			result.Status = data.TICKET_USE_NOT_FOUND
			return result, nil
		}
		return result, fmt.Errorf("Sqlite.TicketUse - %w", err)
	}

	result.Status = data.TICKET_USE_OK
	result.Payload = payload
	result.Uses = uses
	return result, nil
}

func (c Conn) TicketDelete(opts data.TicketUse) (data.TicketUseResult, error) {
	ticket := opts.Ticket
	projectId := opts.ProjectId

	var result data.TicketUseResult

	row := c.Row(`
		delete from authen_tickets
		where project_id = ?1 and ticket = ?2
			and (uses is null or uses > 0)
			and (expires is null or expires > unixepoch())
		returning uses
	`, projectId, ticket)

	var uses *int
	if err := row.Scan(&uses); err != nil {
		if err == sqlite.ErrNoRows {
			result.Status = data.TICKET_USE_NOT_FOUND
			return result, nil
		}
		return result, fmt.Errorf("Sqlite.TicketDelete - %w", err)
	}

	result.Status = data.TICKET_USE_OK
	result.Uses = uses
	return result, nil
}

func (c Conn) LoginLogCreate(opts data.LoginLogCreate) (data.LoginLogCreateResult, error) {
	id := opts.Id
	max := opts.Max
	payload := opts.Payload
	userId := opts.UserId
	status := opts.Status
	projectId := opts.ProjectId

	var result data.LoginLogCreateResult

	canAdd, err := c.loginLogCanAdd(projectId, max)
	if err != nil {
		return result, err
	}

	if !canAdd {
		result.Status = data.LOGIN_LOG_CREATE_MAX
		return result, nil
	}

	err = c.Exec(`
		insert into authen_login_logs (id, project_id, user_id, status, payload)
		values (?1, ?2, ?3, ?4, ?5)
	`, id, projectId, userId, status, payload)

	if err != nil {
		return result, fmt.Errorf("Sqlite.LoginLogCreate - %w", err)
	}

	result.Status = data.LOGIN_LOG_CREATE_OK
	return result, nil
}

func (c Conn) LoginLogGet(opts data.LoginLogGet) (data.LoginLogGetResult, error) {
	userId := opts.UserId
	projectId := opts.ProjectId
	limit := opts.Limit
	offset := opts.Offset

	var result data.LoginLogGetResult

	rows := c.Rows(`
		select id, status, payload, created
		from authen_login_logs
		where project_id = ?1 and user_id = ?2
		order by created desc
		limit ?3 offset ?4
	`, projectId, userId, limit, offset)

	if err := rows.Error(); err != nil {
		return result, fmt.Errorf("Sqlite.LoginLogGet (select) - %w", err)
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

	if err := rows.Error(); err != nil {
		return result, fmt.Errorf("Sqlite.LoginLogGet (scan) - %w", err)
	}

	result.Records = records[:i]
	result.Status = data.LOGIN_LOG_GET_OK
	return result, nil
}

func (c Conn) totpCanAdd(projectId string, userId string, tpe string, max int) (bool, error) {
	// no limit
	if max == 0 {
		return true, nil
	}

	// An update doesn't increase our total count, so if this
	// project+user+type already exists, we can "add" since "add"
	// will mean update in this case
	exists, err := sqlite.Scalar[bool](c.Conn, `
		select exists (
			select 1
			from authen_totps
			where project_id = ?1 and user_id = ?2 and type = ?3
		)`, projectId, userId, tpe)

	if err != nil {
		return false, fmt.Errorf("Sqlite.totpCanAdd (exists) - %w", err)
	}

	if exists {
		return exists, nil
	}

	count, err := sqlite.Scalar[int](c.Conn, `
		select count(*)
		from authen_totps
		where project_id = ?1
	`, projectId)

	if err != nil {
		return false, fmt.Errorf("Sqlite.totpCanAdd (count) - %w", err)
	}
	return count < max, nil
}

func (c Conn) ticketCanAdd(projectId string, max int) (bool, error) {
	// no limit
	if max == 0 {
		return true, nil
	}

	count, err := sqlite.Scalar[int](c.Conn, `
		select count(*)
		from authen_tickets
		where project_id = ?1
	`, projectId)

	if err != nil {
		return false, fmt.Errorf("Sqlite.ticketCanAdd (count) - %w", err)
	}
	return count < max, nil
}

func (c Conn) loginLogCanAdd(projectId string, max int) (bool, error) {
	// no limit
	if max == 0 {
		return true, nil
	}

	count, err := sqlite.Scalar[int](c.Conn, `
		select count(*)
		from authen_login_logs
		where project_id = ?1
	`, projectId)

	if err != nil {
		return false, fmt.Errorf("Sqlite.loginLogCanAdd (count) - %w", err)
	}
	return count < max, nil
}

func scanProject(scanner sqlite.Scanner) (*data.Project, error) {
	var id, totpIssuer string
	var totpMax, totpSetupTTL, totpSecretLength int
	var ticketMax, ticketMaxPayloadLength int
	var loginLogMax, loginLogMaxPayloadLength int

	err := scanner.Scan(&id,
		&totpIssuer, &totpMax, &totpSetupTTL, &totpSecretLength,
		&ticketMax, &ticketMaxPayloadLength,
		&loginLogMax, &loginLogMaxPayloadLength)

	if err != nil {
		return nil, err
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
