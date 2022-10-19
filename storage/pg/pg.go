package pg

import (
	"context"
	"time"

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
		return DB{}, err
	}
	return DB{db}, nil
}

func (db DB) Ping() error {
	_, err := db.Exec(context.Background(), "select 1")
	return err
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
	if err == pg.ErrNoRows {
		return nil, nil
	}
	return project, err
}

func (db DB) GetUpdatedProjects(timestamp time.Time) ([]*data.Project, error) {

	// Not sure fetching the count upfront really makes much sense.
	// But we do expect this to be 0 almost every time that it's called, so most
	// of the time we're going to be doing a single DB call (either to get the count
	// which returns 0, or to get an empty result set).
	count, err := pg.Scalar[int](db.DB, "select count(*) from authen_projects where updated > $1", timestamp)
	if count == 0 || err != nil {
		return nil, err
	}

	rows, err := db.Query(context.Background(), "select id, issuer, max_users from authen_projects where updated > $1", timestamp)
	if err != nil {
		return nil, err
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

func (db DB) CreateTOTP(opts data.CreateTOTP) (data.CreateTOTPResult, error) {
	value := opts.Value
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
		insert into authen_totp_setups (project_id, user_id, nonce, secret)
		values ($1, $2, $3, $4)
		on conflict (project_id, user_id) do update set nonce = $3, secret = $4
	`, projectId, userId, value.Nonce, value.Data)

	if err != nil {
		return result, err
	}

	result.Status = data.CREATE_TOTP_OK
	return result, nil
}

func (db DB) canAddUser(projectId string, userId string, maxUsers uint32) (bool, error) {
	if maxUsers == 0 {
		return true, nil
	}

	// if the user already exists, then we aren't adding a user
	// and thus cannot be over any limit
	exists, err := pg.Scalar[bool](db.DB, "select exists (select 1 from authen_totps where project_id = $1 and user_id = $2)", projectId, userId)
	if exists || err != nil {
		return exists, err
	}

	count, err := pg.Scalar[uint32](db.DB, "select count(*) from authen_totps where project_id = $1", projectId)
	return count < maxUsers, err
}

func scanProject(row pg.Row) (*data.Project, error) {
	var id, issuer string
	var maxUsers int
	if err := row.Scan(&id, &issuer, &maxUsers); err != nil {
		return nil, err
	}

	return &data.Project{
		Id:       id,
		Issuer:   issuer,
		MaxUsers: uint32(maxUsers),
	}, nil
}
