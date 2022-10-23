package pg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"src.goblgobl.com/utils/pg"
	"src.goblgobl.com/utils/typed"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/authen/storage/pg/migrations"
)

type DB struct {
	pg.DB
}

func New(config typed.Typed) (DB, error) {
	db, err := pg.New(config.String("url"))
	if err != nil {
		return DB{}, fmt.Errorf("PG.New - %w", err)
	}
	return DB{db}, nil
}

func (db DB) Ping() error {
	_, err := db.Exec(context.Background(), "select 1")
	if err != nil {
		return fmt.Errorf("PG.Ping - %w", err)
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
		Type:      "pg",
		Migration: migration,
	}, nil
}

func (db DB) GetProject(id string) (*data.Project, error) {
	row := db.QueryRow(context.Background(), `
		select id, issuer, max_users
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

	rows, err := db.Query(context.Background(), "select id, issuer, max_users from authen_projects where updated > $1", timestamp)
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

func (db DB) CreateTOTPSetup(opts data.CreateTOTP) (data.CreateTOTPResult, error) {
	secret := opts.Secret
	userId := opts.UserId
	maxUsers := opts.MaxUsers
	projectId := opts.ProjectId

	var result data.CreateTOTPResult

	// Since we check first, then add the user (outside of a transaction)
	// concurrent calls to this might result in going a little over maxUsers
	// but I'm ok with that in the name of minimizing the DB calls
	// we need to make inside a transaction.
	canAdd, err := db.canAddUser(projectId, userId, maxUsers)
	if err != nil {
		return result, err
	}
	if !canAdd {
		result.Status = data.CREATE_TOTP_MAX_USERS
		return result, nil
	}

	_, err = db.Exec(context.Background(), `
		insert into authen_totp_setups (project_id, user_id, secret)
		values ($1, $2, $3)
		on conflict (project_id, user_id) do update set secret = $3
	`, projectId, userId, secret)

	if err != nil {
		return result, fmt.Errorf("PG.CreateTOTPSetup - %w", err)
	}

	result.Status = data.CREATE_TOTP_OK
	return result, nil
}

func (db DB) CreateTOTP(opts data.CreateTOTP) (data.CreateTOTPResult, error) {
	secret := opts.Secret
	userId := opts.UserId
	projectId := opts.ProjectId

	err := db.Transaction(func(tx pgx.Tx) error {
		_, err := db.Exec(context.Background(), `
			insert into authen_totps (project_id, user_id, secret)
			values ($1, $2, $3)
			on conflict (project_id, user_id) do update set secret = $3
		`, projectId, userId, secret)

		if err != nil {
			return fmt.Errorf("PG.CreateTOTP (upsert) - %w", err)
		}

		_, err = db.Exec(context.Background(), `
			delete from authen_totp_setups
			where project_id = $1 and user_id = $2
		`, projectId, userId)

		if err != nil {
			return fmt.Errorf("PG.CreateTOTP (delete) - %w", err)
		}
		return nil
	})

	return data.CreateTOTPResult{
		Status: data.CREATE_TOTP_OK,
	}, err
}

func (db DB) GetTOTPSetup(opts data.GetTOTPSetup) (data.GetTOTPSetupResult, error) {
	userId := opts.UserId
	projectId := opts.ProjectId
	var result data.GetTOTPSetupResult

	row := db.QueryRow(context.Background(), `
		select secret
		from authen_totp_setups
		where project_id = $1 and user_id = $2
	`, projectId, userId)

	var secret []byte
	if err := row.Scan(&secret); err != nil {
		if err == pg.ErrNoRows {
			result.Status = data.GET_TOTP_SETUP_NOT_FOUND
			return result, nil
		}
		return result, fmt.Errorf("PG.GetTOTPSetup - %w", err)
	}

	return data.GetTOTPSetupResult{
		Secret: secret,
		Status: data.GET_TOTP_SETUP_OK,
	}, nil
}

func (db DB) canAddUser(projectId string, userId string, maxUsers uint32) (bool, error) {
	if maxUsers == 0 {
		return true, nil
	}

	// if the user already exists, then we aren't adding a user
	// and thus cannot be over any limit
	exists, err := pg.Scalar[bool](db.DB, "select exists (select 1 from authen_totps where project_id = $1 and user_id = $2)", projectId, userId)
	if err != nil {
		return false, fmt.Errorf("PG.canAddUser (exists) - %w", err)
	}
	if exists {
		return exists, nil
	}

	count, err := pg.Scalar[uint32](db.DB, "select count(*) from authen_totps where project_id = $1", projectId)
	if err != nil {
		return false, fmt.Errorf("PG.canAddUser (count) - %w", err)
	}
	return count < maxUsers, nil
}

func scanProject(row pg.Row) (*data.Project, error) {
	var id, issuer string
	var maxUsers int
	if err := row.Scan(&id, &issuer, &maxUsers); err != nil {
		return nil, fmt.Errorf("PG.scanProject - %w", err)
	}

	return &data.Project{
		Id:       id,
		Issuer:   issuer,
		MaxUsers: uint32(maxUsers),
	}, nil
}
