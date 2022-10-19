package pg

import (
	"testing"
	"time"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/utils/encryption"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/uuid"
)

var db DB

func init() {
	var err error
	db, err = New(typed.Typed{"url": tests.PG()})
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

func Test_CreateTOTP(t *testing.T) {
	projectId1, projectId2 := uuid.String(), uuid.String()

	db.MustExec(`
		insert into authen_totps (project_id, user_id, nonce, secret) values
		($1, 'u1', '', ''),
		($2, 'u2', '', '')
	`, projectId1, projectId1)

	// can add 1 more
	res, err := db.CreateTOTP(data.CreateTOTP{
		MaxUsers:  3,
		UserId:    "u3",
		ProjectId: projectId1,
		Value:     encryption.Value{Nonce: []byte{1, 2}, Data: []byte{3, 4}},
	})
	assert.Nil(t, err)
	assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
	row, _ := db.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId1, "u3")
	assert.Nowish(t, row.Time("created"))
	assert.Bytes(t, row["nonce"].([]byte), []byte{1, 2})
	assert.Bytes(t, row["secret"].([]byte), []byte{3, 4})

	// can't add any more
	res, err = db.CreateTOTP(data.CreateTOTP{
		MaxUsers:  2,
		UserId:    "u4",
		ProjectId: projectId1,
		Value:     encryption.Value{Nonce: []byte{11, 12}, Data: []byte{13, 14}},
	})
	assert.Nil(t, err)
	assert.Equal(t, res.Status, data.CREATE_TOTP_MAX_USERS)

	// 0 == no limit
	res, err = db.CreateTOTP(data.CreateTOTP{
		MaxUsers:  0,
		UserId:    "u4",
		ProjectId: projectId1,
		Value:     encryption.Value{Nonce: []byte{11, 12}, Data: []byte{13, 14}},
	})
	assert.Nil(t, err)
	assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
	row, _ = db.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId1, "u4")
	assert.Nowish(t, row.Time("created"))
	assert.Bytes(t, row["nonce"].([]byte), []byte{11, 12})
	assert.Bytes(t, row["secret"].([]byte), []byte{13, 14})

	// limits are per project
	res, err = db.CreateTOTP(data.CreateTOTP{
		MaxUsers:  1,
		UserId:    "u4",
		ProjectId: projectId2,
		Value:     encryption.Value{Nonce: []byte{21, 22}, Data: []byte{23, 24}},
	})
	assert.Nil(t, err)
	assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
	row, _ = db.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId2, "u4")
	assert.Nowish(t, row.Time("created"))
	assert.Bytes(t, row["nonce"].([]byte), []byte{21, 22})
	assert.Bytes(t, row["secret"].([]byte), []byte{23, 24})

	// existing users don't increment count
	res, err = db.CreateTOTP(data.CreateTOTP{
		MaxUsers:  1,
		UserId:    "u1",
		ProjectId: projectId1,
		Value:     encryption.Value{Nonce: []byte{31, 32}, Data: []byte{33, 34}},
	})
	assert.Nil(t, err)
	assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
	row, _ = db.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId1, "u1")
	assert.Nowish(t, row.Time("created"))
	assert.Bytes(t, row["nonce"].([]byte), []byte{31, 32})
	assert.Bytes(t, row["secret"].([]byte), []byte{33, 34})
}
