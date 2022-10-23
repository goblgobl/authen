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
	row := c.Row("select id, issuer, max_users from authen_projects where id = ?1", id)

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

	rows := c.Rows("select id, issuer, max_users from authen_projects where updated > ?1", timestamp)
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
	secret := opts.Secret
	userId := opts.UserId
	projectId := opts.ProjectId

	err := c.Transaction(func() error {
		err := c.Exec(`
			insert or replace into authen_totps (project_id, user_id, secret)
			values (?1, ?2, ?3)
		`, projectId, userId, secret)
		if err != nil {
			return fmt.Errorf("Sqlite.CreateTOTP (upsert) - %w", err)
		}

		err = c.Exec(`
			delete from authen_totp_setups
			where project_id = ?1 and user_id = ?2
		`, projectId, userId)
		if err != nil {
			return fmt.Errorf("Sqlite.CreateTOTP (delete) - %w", err)
		}

		return nil

	})

	return data.CreateTOTPResult{
		Status: data.CREATE_TOTP_OK,
	}, err
}

func (c Conn) CreateTOTPSetup(opts data.CreateTOTP) (data.CreateTOTPResult, error) {
	secret := opts.Secret
	userId := opts.UserId
	maxUsers := opts.MaxUsers
	projectId := opts.ProjectId

	var result data.CreateTOTPResult

	// Since we check first, then add the user (outside of a transaction)
	// concurrent calls to this might result in going a little over maxUsers
	// but I'm ok with that in the name of minimizing the DB calls
	// we need to make inside a transaction.
	canAdd, err := c.canAddUser(projectId, userId, maxUsers)
	if err != nil {
		return result, err
	}

	if !canAdd {
		result.Status = data.CREATE_TOTP_MAX_USERS
		return result, nil
	}

	err = c.Exec(`
		insert or replace into authen_totp_setups (project_id, user_id, secret)
		values (?1, ?2, ?3)
	`, projectId, userId, secret)

	if err != nil {
		return result, fmt.Errorf("Sqlite.CreateTOTPSetup - %w", err)
	}

	result.Status = data.CREATE_TOTP_OK
	return result, nil
}

func (c Conn) GetTOTPSetup(opts data.GetTOTPSetup) (data.GetTOTPSetupResult, error) {
	userId := opts.UserId
	projectId := opts.ProjectId
	var result data.GetTOTPSetupResult

	row := c.Row(`
		select secret
		from authen_totp_setups
		where project_id = ?1 and user_id = ?2
	`, projectId, userId)

	var secret []byte
	if err := row.Scan(&secret); err != nil {
		if err == sqlite.ErrNoRows {
			result.Status = data.GET_TOTP_SETUP_NOT_FOUND
			return result, nil
		}
		return result, fmt.Errorf("Sqlite.GetTOTPSetup - %w", err)
	}

	return data.GetTOTPSetupResult{
		Secret: secret,
		Status: data.GET_TOTP_SETUP_OK,
	}, nil
}

func (c Conn) canAddUser(projectId string, userId string, maxUsers uint32) (bool, error) {
	if maxUsers == 0 {
		return true, nil
	}

	// if the user already exists, then we aren't adding a user
	// and thus cannot be over any limit
	exists, err := sqlite.Scalar[bool](c.Conn, "select exists (select 1 from authen_totps where project_id = ?1 and user_id = ?2)", projectId, userId)
	if exists || err != nil {
		return exists, err
	}

	count, err := sqlite.Scalar[int](c.Conn, "select count(*) from authen_totps where project_id = ?1", projectId)
	return count < int(maxUsers), err
}

func scanProject(scanner sqlite.Scanner) (*data.Project, error) {
	var id, issuer string
	var maxUsers int

	if err := scanner.Scan(&id, &issuer, &maxUsers); err != nil {
		return nil, err
	}

	return &data.Project{
		Id:       id,
		Issuer:   issuer,
		MaxUsers: uint32(maxUsers),
	}, nil
}
