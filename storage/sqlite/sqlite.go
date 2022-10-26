package sqlite

import (
	"fmt"
	"time"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/authen/storage/sqlite/migrations"
	"src.goblgobl.com/utils/sqlite"
	"src.goblgobl.com/utils/typed"
)

type Conn struct {
	sqlite.Conn
}

func New(config typed.Typed) (Conn, error) {
	filePath := config.String("path")
	conn, err := sqlite.New(filePath, true)
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
	row := c.Row("select id, issuer, totp_max, totp_setup_ttl from authen_projects where id = ?1", id)

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

	rows := c.Rows("select id, issuer, totp_max, totp_setup_ttl from authen_projects where updated > ?1", timestamp)
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

func (c Conn) CreateTOTP(opts data.CreateTOTP) (data.CreateTOTPResult, error) {
	max := opts.Max
	tpe := opts.Type
	secret := opts.Secret
	userId := opts.UserId
	expires := opts.Expires
	pending := expires != nil
	projectId := opts.ProjectId

	var result data.CreateTOTPResult

	// Since we check first, then add the user (outside of a transaction)
	// concurrent calls to this might result in going a little over max
	// but I'm ok with that in the name of minimizing the DB calls
	// we need to make inside a transaction.
	canAdd, err := c.canAddTOTP(projectId, userId, tpe, max)
	if err != nil {
		return result, err
	}

	if !canAdd {
		result.Status = data.CREATE_TOTP_MAX
		return result, nil
	}

	err = c.Transaction(func() error {
		err := c.Exec(`
			insert or replace into authen_totps (project_id, user_id, type, pending, secret, expires)
			values (?1, ?2, ?3, ?4, ?5, ?6)
		`, projectId, userId, tpe, pending, secret, expires)

		if err != nil {
			return fmt.Errorf("Sqlite.CreateTOTP (upsert) - %w", err)
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
			return fmt.Errorf("Sqlite.CreateTOTP (delete) - %w", err)
		}

		return nil
	})

	result.Status = data.CREATE_TOTP_OK
	return result, err
}

func (c Conn) GetTOTP(opts data.GetTOTP) (data.GetTOTPResult, error) {
	tpe := opts.Type
	userId := opts.UserId
	pending := opts.Pending
	projectId := opts.ProjectId
	var result data.GetTOTPResult

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
			result.Status = data.GET_TOTP_NOT_FOUND
			return result, nil
		}
		return result, fmt.Errorf("Sqlite.GetTOTP - %w", err)
	}

	return data.GetTOTPResult{
		Secret: secret,
		Status: data.GET_TOTP_OK,
	}, nil
}

func (c Conn) DeleteTOTP(opts data.GetTOTP) error {
	tpe := opts.Type
	userId := opts.UserId
	projectId := opts.ProjectId

	err := c.Exec(`
		delete from authen_totps
		where project_id = ?1
			and user_id = ?2
			and (type = ?3 or ?3 = '')
	`, projectId, userId, tpe)

	if err != nil {
		return fmt.Errorf("Sqlite.DeleteTOTP - %w", err)
	}

	return nil
}

func (c Conn) canAddTOTP(projectId string, userId string, tpe string, max uint32) (bool, error) {
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
		return false, fmt.Errorf("Sqlite.canAddTOTP (exists) - %w", err)
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
		return false, fmt.Errorf("Sqlite.canAddTOTP (count) - %w", err)
	}
	return count < int(max), nil
}

func scanProject(scanner sqlite.Scanner) (*data.Project, error) {
	var id, issuer string
	var totpMax, totpSetupTTL int

	if err := scanner.Scan(&id, &issuer, &totpMax, &totpSetupTTL); err != nil {
		return nil, err
	}

	return &data.Project{
		Id:           id,
		Issuer:       issuer,
		TOTPMax:      uint32(totpMax),
		TOTPSetupTTL: uint32(totpSetupTTL),
	}, nil
}
