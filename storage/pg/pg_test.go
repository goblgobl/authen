package pg

import (
	"testing"
	"time"

	"src.goblgobl.com/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/utils/typed"
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
	p, err := db.GetProject("unknown")
	assert.Nil(t, err)
	assert.Nil(t, p)
}

func Test_GetProject_Success(t *testing.T) {
	db.MustExec("truncate table authen_projects")
	db.MustExec("insert into authen_projects (id, issuer, max_users) values ('p1qa', 'goblgobl.com', 84)")
	p, err := db.GetProject("p1qa")
	assert.Nil(t, err)
	assert.Equal(t, p.Id, "p1qa")
	assert.Equal(t, p.MaxUsers, 84)
	assert.Equal(t, p.Issuer, "goblgobl.com")
}

func Test_GetUpdatedProjects_None(t *testing.T) {
	db.MustExec("truncate table authen_projects")
	db.MustExec("insert into authen_projects (id, issuer, max_users, updated) values ('p3', '', 11, now() - interval '1 second')")
	updated, err := db.GetUpdatedProjects(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(updated), 0)
}

func Test_GetUpdatedProjects_Success(t *testing.T) {
	db.MustExec("truncate table authen_projects")
	db.MustExec(`
			insert into authen_projects (id, issuer, max_users, updated) values
			('p1', '', 1, now() - interval '500 second'),
			('p2', '', 2, now() - interval '200 second'),
			('p3', '', 3, now() - interval '100 second'),
			('p4', '', 4, now() - interval '10 second')
		`)
	updated, err := db.GetUpdatedProjects(time.Now().Add(time.Second * -105))
	assert.Nil(t, err)
	assert.Equal(t, len(updated), 2)

	// order isn't deterministic
	id1, id2 := updated[0].Id, updated[1].Id
	assert.True(t, id1 != id2)
	assert.True(t, id1 == "p3" || id1 == "p4")
	assert.True(t, id2 == "p3" || id2 == "p4")
}
