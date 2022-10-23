package pg

import (
	"testing"
	"time"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/uuid"
)

var db DB

func init() {
	url := tests.PG()
	tpe := tests.StorageType()

	if tpe == "cr" {
		url = tests.CR()
	}

	var err error
	db, err = New(typed.Typed{"url": url}, tpe)
	if err != nil {
		panic(err)
	}
	if err := db.EnsureMigrations(); err != nil {
		panic(err)
	}
}

func Test_Ping(t *testing.T) {
	assert.Nil(t, db.Ping())
}

func Test_GetProject_Unknown(t *testing.T) {
	p, err := db.GetProject("76FBFC33-7CB1-447D-8786-C9D370737AA6")
	assert.Nil(t, err)
	assert.Nil(t, p)
}

func Test_GetProject_Success(t *testing.T) {
	id := uuid.String()
	db.MustExec("truncate table authen_projects")
	db.MustExec("insert into authen_projects (id, issuer, max_users) values ($1, 'goblgobl.com', 84)", id)
	p, err := db.GetProject(id)
	assert.Nil(t, err)
	assert.Equal(t, p.Id, id)
	assert.Equal(t, p.MaxUsers, 84)
	assert.Equal(t, p.Issuer, "goblgobl.com")
}

func Test_GetUpdatedProjects_None(t *testing.T) {
	id := uuid.String()
	db.MustExec("truncate table authen_projects")
	db.MustExec("insert into authen_projects (id, issuer, max_users, updated) values ($1, '', 11, now() - interval '1 second')", id)
	updated, err := db.GetUpdatedProjects(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(updated), 0)
}

func Test_GetUpdatedProjects_Success(t *testing.T) {
	id1, id2, id3, id4 := uuid.String(), uuid.String(), uuid.String(), uuid.String()
	db.MustExec("truncate table authen_projects")
	db.MustExec(`
			insert into authen_projects (id, issuer, max_users, updated) values
			($1, '', 1, now() - interval '500 second'),
			($2, '', 2, now() - interval '200 second'),
			($3, '', 3, now() - interval '100 second'),
			($4, '', 4, now() - interval '10 second')
		`, id1, id2, id3, id4)
	updated, err := db.GetUpdatedProjects(time.Now().Add(time.Second * -105))
	assert.Nil(t, err)
	assert.Equal(t, len(updated), 2)

	// order isn't deterministic
	actual1, actual2 := updated[0].Id, updated[1].Id
	assert.True(t, actual1 != actual2)
	assert.True(t, actual1 == id3 || actual1 == id4)
	assert.True(t, actual2 == id3 || actual2 == id4)
}

func Test_CreateTOTPSetup(t *testing.T) {
	projectId1, projectId2 := uuid.String(), uuid.String()

	db.MustExec(`
		insert into authen_totps (project_id, user_id, secret) values
		($1, 'u1', ''),
		($2, 'u2', '')
	`, projectId1, projectId1)

	// can add 1 more
	res, err := db.CreateTOTPSetup(data.CreateTOTP{
		MaxUsers:  3,
		UserId:    "u3",
		ProjectId: projectId1,
		Secret:    []byte{3, 4},
	})
	assert.Nil(t, err)
	assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
	row, _ := db.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId1, "u3")
	assert.Nowish(t, row.Time("created"))
	assert.Bytes(t, row.Bytes("secret"), []byte{3, 4})

	// can't add any more
	res, err = db.CreateTOTPSetup(data.CreateTOTP{
		MaxUsers:  2,
		UserId:    "u4",
		ProjectId: projectId1,
		Secret:    []byte{13, 14},
	})
	assert.Nil(t, err)
	assert.Equal(t, res.Status, data.CREATE_TOTP_MAX_USERS)

	// 0 == no limit
	res, err = db.CreateTOTPSetup(data.CreateTOTP{
		MaxUsers:  0,
		UserId:    "u4",
		ProjectId: projectId1,
		Secret:    []byte{15, 16},
	})
	assert.Nil(t, err)
	assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
	row, _ = db.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId1, "u4")
	assert.Nowish(t, row.Time("created"))
	assert.Bytes(t, row.Bytes("secret"), []byte{15, 16})

	// limits are per project
	res, err = db.CreateTOTPSetup(data.CreateTOTP{
		MaxUsers:  1,
		UserId:    "u4",
		ProjectId: projectId2,
		Secret:    []byte{23, 24},
	})
	assert.Nil(t, err)
	assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
	row, _ = db.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId2, "u4")
	assert.Nowish(t, row.Time("created"))
	assert.Bytes(t, row.Bytes("secret"), []byte{23, 24})

	// existing users don't increment count
	res, err = db.CreateTOTPSetup(data.CreateTOTP{
		MaxUsers:  1,
		UserId:    "u1",
		ProjectId: projectId1,
		Secret:    []byte{33, 34},
	})
	assert.Nil(t, err)
	assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
	row, _ = db.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId1, "u1")
	assert.Nowish(t, row.Time("created"))
	assert.Bytes(t, row.Bytes("secret"), []byte{33, 34})
}

func Test_CreateTOTP(t *testing.T) {
	userId, projectId1 := uuid.String(), uuid.String()

	// test upsert
	for i := byte(1); i < 3; i++ {
		db.MustExec(`
			insert into authen_totp_setups (project_id, user_id, secret) values
			($1, $2, '2b')
		`, projectId1, userId)

		res, err := db.CreateTOTP(data.CreateTOTP{
			UserId:    userId,
			ProjectId: projectId1,
			Secret:    []byte{i, i},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.CREATE_TOTP_OK)

		row, _ := db.RowToMap("select * from authen_totp_setups where user_id = $1", userId)
		assert.Nil(t, row)

		row, _ = db.RowToMap("select * from authen_totps where user_id = $1", userId)
		assert.Nowish(t, row.Time("created"))
		assert.Equal(t, row.String("project_id"), projectId1)
		assert.Bytes(t, row.Bytes("secret"), []byte{i, i})
	}
}

func Test_GetTOTPSetup_NotFound(t *testing.T) {
	result, err := db.GetTOTPSetup(data.GetTOTPSetup{
		UserId:    "u1",
		ProjectId: uuid.String(),
	})
	assert.Nil(t, err)
	assert.Equal(t, result.Status, data.GET_TOTP_SETUP_NOT_FOUND)
}

func Test_GetTOTPSetup_Found(t *testing.T) {
	projectId, userId := uuid.String(), uuid.String()
	db.MustExec(`
		insert into authen_totp_setups (project_id, user_id, secret) values
		($1, $2, 'bbb2')
	`, projectId, userId)

	result, err := db.GetTOTPSetup(data.GetTOTPSetup{
		UserId:    userId,
		ProjectId: projectId,
	})
	assert.Nil(t, err)
	assert.Equal(t, result.Status, data.GET_TOTP_SETUP_OK)
	assert.Bytes(t, result.Secret, []byte("bbb2"))
}
