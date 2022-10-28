package pg

import (
	"fmt"
	"os"
	"testing"
	"time"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/utils/pg"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/uuid"
)

var db DB

func shouldRunTests() bool {
	tpe := tests.StorageType()
	return tpe == "cockroach" || tpe == "postgres"
}

func TestMain(m *testing.M) {
	if !shouldRunTests() {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func init() {
	if !shouldRunTests() {
		return
	}
	url := tests.PG()
	tpe := tests.StorageType()

	if tpe == "cockroach" {
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
	db.MustExec(`
		insert into authen_projects (id, totp_issuer, totp_max, totp_setup_ttl, totp_secret_length)
		values ($1, 'goblgobl.com', 84, 124, 38)
	`, id)

	p, err := db.GetProject(id)
	assert.Nil(t, err)
	assert.Equal(t, p.Id, id)
	assert.Equal(t, p.TOTPMax, 84)
	assert.Equal(t, p.TOTPSetupTTL, 124)
	assert.Equal(t, p.TOTPSecretLength, 38)
	assert.Equal(t, p.TOTPIssuer, "goblgobl.com")
}

func Test_GetUpdatedProjects_None(t *testing.T) {
	id := uuid.String()
	db.MustExec("truncate table authen_projects")
	db.MustExec(`
		insert into authen_projects (id, totp_issuer, totp_max, totp_setup_ttl, totp_secret_length, updated)
		values ($1, '', 11, 12, 13, now() - interval '1 second')
	`, id)
	updated, err := db.GetUpdatedProjects(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(updated), 0)
}

func Test_GetUpdatedProjects_Success(t *testing.T) {
	id1, id2, id3, id4 := uuid.String(), uuid.String(), uuid.String(), uuid.String()
	db.MustExec("truncate table authen_projects")
	db.MustExec(`
			insert into authen_projects (id, totp_issuer, totp_max, totp_setup_ttl, totp_secret_length, updated) values
			($1, '', 1, 11, 21, now() - interval '500 second'),
			($2, '', 2, 12, 22, now() - interval '200 second'),
			($3, '', 3, 13, 23, now() - interval '100 second'),
			($4, '', 4, 14, 24, now() - interval '10 second')
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

func Test_TOTPCreate(t *testing.T) {
	now := time.Now()
	projectId1, projectId2 := uuid.String(), uuid.String()

	db.MustExec(`
			insert into authen_totps (project_id, user_id, type, pending, secret) values
			($1, 'u1', 't1', false, 'sec1'),
			($2, 'u2', 't2', true, 'sec2')
		`, projectId1, projectId1)

	// Adds more when less than max
	for i, expires := range []*time.Time{nil, &now} {
		secret := []byte{byte(i), byte(i)}
		tpe := fmt.Sprintf("t-%d", i)
		res, err := db.TOTPCreate(data.TOTPCreate{
			Max:       4,
			UserId:    "u1",
			Type:      tpe,
			Secret:    secret,
			Expires:   expires,
			ProjectId: projectId1,
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TOTP_CREATE_OK)

		row, _ := db.RowToMap("select * from authen_totps where project_id = $1 and user_id = $2 and type = $3 and pending = $4", projectId1, "u1", tpe, expires != nil)
		assert.Nowish(t, row.Time("created"))
		assert.Equal(t, row.Bool("pending"), expires != nil)
		if expires == nil {
			assert.Nil(t, row["expires"])
		} else {
			assert.Timeish(t, row.Time("expires"), *expires)
		}

		assert.Bytes(t, row.Bytes("secret"), secret)
	}

	// can't add any more, pending or not:
	for _, expires := range []*time.Time{nil, &now} {
		res, err := db.TOTPCreate(data.TOTPCreate{
			Max:       4,
			UserId:    "u4",
			Type:      "",
			Expires:   expires,
			ProjectId: projectId1,
			Secret:    []byte{13, 14},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TOTP_CREATE_MAX)
	}

	// 0 == no limit
	{
		res, err := db.TOTPCreate(data.TOTPCreate{
			Max:       0,
			UserId:    "u4",
			Type:      "t4",
			ProjectId: projectId1,
			Secret:    []byte{23, 24},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TOTP_CREATE_OK)
		row, _ := db.RowToMap("select * from authen_totps where project_id = $1 and user_id = $2 and type = $3", projectId1, "u4", "t4")
		assert.Nowish(t, row.Time("created"))
		assert.Nil(t, row["expires"])
		assert.Bytes(t, row.Bytes("secret"), []byte{23, 24})
	}

	// limits are per project (there's no other totp for project2)
	{
		res, err := db.TOTPCreate(data.TOTPCreate{
			Max:       1,
			UserId:    "u4",
			Type:      "",
			ProjectId: projectId2,
			Secret:    []byte{23, 24},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TOTP_CREATE_OK)
		row, _ := db.RowToMap("select * from authen_totps where project_id = $1 and user_id = $2", projectId2, "u4")
		assert.Nowish(t, row.Time("created"))
		assert.Bytes(t, row.Bytes("secret"), []byte{23, 24})
	}

	// existing users+type don't increment count
	for _, expires := range []*time.Time{nil, &now} {
		res, err := db.TOTPCreate(data.TOTPCreate{
			Max:       1,
			UserId:    "u1",
			Type:      "t1",
			Expires:   expires,
			ProjectId: projectId1,
			Secret:    []byte{33, 34},
		})

		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TOTP_CREATE_OK)
		row, _ := db.RowToMap("select * from authen_totps where project_id = $1 and user_id = $2 and type = $3 and pending = $4", projectId1, "u1", "t1", expires != nil)
		assert.Nowish(t, row.Time("created"))
		assert.Bytes(t, row.Bytes("secret"), []byte{33, 34})
	}

	// existing users DO increment count for a different type
	for _, expires := range []*time.Time{nil, &now} {
		res, err := db.TOTPCreate(data.TOTPCreate{
			Max:       1,
			UserId:    "u1",
			Type:      "t-new",
			Expires:   expires,
			ProjectId: projectId1,
			Secret:    []byte{33, 34},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TOTP_CREATE_MAX)
	}
}

func Test_TOTPCreate_NonPending_DeletesPending(t *testing.T) {
	projectId1 := uuid.String()

	db.MustExec(`
			insert into authen_totps (project_id, user_id, type, pending, secret) values
			($1, 'u1', 't1', true, 'sec1')
		`, projectId1)

	res, err := db.TOTPCreate(data.TOTPCreate{
		Type:      "t1",
		UserId:    "u1",
		ProjectId: projectId1,
		Secret:    []byte{99, 98},
	})
	assert.Nil(t, err)
	assert.Equal(t, res.Status, data.TOTP_CREATE_OK)

	rows, _ := db.RowsToMap("select * from authen_totps where project_id = $1", projectId1)
	assert.Equal(t, len(rows), 1)
	row := rows[0]
	assert.Nil(t, row["expires"])
	assert.False(t, row.Bool("pending"))
}

func Test_TOTPGet(t *testing.T) {
	projectId1, projectId2 := uuid.String(), uuid.String()
	db.MustExec(`
			insert into authen_totps (project_id, user_id, type, pending, expires, secret) values
			($1, 'u1', 't1', true, now() - interval '1 second', 'sec1'),
			($1, 'u2', 't2', true, now() + interval '5 second', 'sec2'),
			($1, 'u2', 't4', true, now() + interval '5 second', 'sec3'),
			($1, 'u2', 't2', false, null, 'sec4'),
			($2, 'u2', 't3', false, null, 'sec5')
		`, projectId1, projectId2)

	assertNotFound := func(opts data.TOTPGet) {
		result, err := db.TOTPGet(opts)
		assert.Nil(t, err)
		assert.Equal(t, result.Status, data.TOTP_GET_NOT_FOUND)
	}

	assertSecret := func(opts data.TOTPGet, secret string) {
		result, err := db.TOTPGet(opts)
		assert.Nil(t, err)
		assert.Equal(t, result.Status, data.TOTP_GET_OK)
		assert.Bytes(t, result.Secret, []byte(secret))
	}

	// expired
	assertNotFound(data.TOTPGet{
		Type:      "t1",
		UserId:    "u1",
		ProjectId: projectId1,
		Pending:   true,
	})

	// user doesn't have this type
	assertNotFound(data.TOTPGet{
		Type:      "t9",
		UserId:    "u1",
		ProjectId: projectId1,
		Pending:   false,
	})

	// user doesn't have this type in non-setup
	assertNotFound(data.TOTPGet{
		Type:      "t4",
		UserId:    "u1",
		ProjectId: projectId1,
		Pending:   false,
	})

	// wrong project
	assertNotFound(data.TOTPGet{
		Type:      "t3",
		UserId:    "u2",
		ProjectId: projectId1,
		Pending:   false,
	})

	// not expired
	assertSecret(data.TOTPGet{
		Type:      "t2",
		UserId:    "u2",
		ProjectId: projectId1,
		Pending:   true,
	}, "sec2")

	// non-setup
	assertSecret(data.TOTPGet{
		Type:      "t2",
		UserId:    "u2",
		ProjectId: projectId1,
		Pending:   false,
	}, "sec4")
}

func Test_TOTPDelete(t *testing.T) {
	projectId1, projectId2 := uuid.String(), uuid.String()
	projectIds := []string{projectId1, projectId2}
	assertCount := func(expected int, args ...string) {
		actual := 0
		var err error

		switch len(args) {
		case 0:
			// count of all, to make sure we didn't over-delete
			actual, err = pg.Scalar[int](db.DB, "select count(*) from authen_totps where project_id = any($1)", projectIds)
		case 2:
			// count of all fo user
			actual, err = pg.Scalar[int](db.DB, "select count(*) from authen_totps where project_id = $1 and user_id = $2", args[0], args[1])
		case 3:
			// count for user+type
			actual, err = pg.Scalar[int](db.DB, "select count(*) from authen_totps where project_id = $1 and user_id = $2 and type = $3", args[0], args[1], args[2])
		}
		if err != nil {
			panic(err)
		}
		assert.Equal(t, actual, expected)
	}

	db.MustExec(`
			insert into authen_totps (project_id, user_id, type, pending, expires, secret) values
			($1, 'u1', 't1', true, now(), 'sec1'),
			($1, 'u2', 't2', true, now(), 'sec2'),
			($1, 'u2', 't4', false, now(), 'sec3'),
			($1, 'u2', 't2', false, null, 'sec4'),
			($2, 'u2', 't3', true, null, 'sec5'),
			($1, 'u3', 't1', true, null, 'sec5')
		`, projectId1, projectId2)

	// specific type
	err := db.TOTPDelete(data.TOTPGet{
		Type:      "t1",
		UserId:    "u1",
		ProjectId: projectId1,
	})
	assert.Nil(t, err)
	assertCount(5)
	assertCount(0, projectId1, "u1", "t1")

	// all types for the user
	err = db.TOTPDelete(data.TOTPGet{
		UserId:    "u2",
		ProjectId: projectId1,
	})
	assert.Nil(t, err)
	assertCount(2)
	assertCount(0, projectId1, "u2")
}
