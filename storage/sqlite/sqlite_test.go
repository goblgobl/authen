package sqlite

import (
	"testing"
	"time"

	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/utils/typed"
)

func Test_Ping(t *testing.T) {
	withTestDB(func(conn Conn) {
		assert.Nil(t, conn.Ping())
	})
}

func Test_GetProject_Unknown(t *testing.T) {
	withTestDB(func(conn Conn) {
		p, err := conn.GetProject("unknown")
		assert.Nil(t, err)
		assert.Nil(t, p)
	})
}

func Test_GetProject_Success(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec("insert into authen_projects (id, issuer, max_users) values ('p1', 'is1', 93)")
		p, err := conn.GetProject("p1")
		assert.Nil(t, err)
		assert.Equal(t, p.Id, "p1")
		assert.Equal(t, p.Issuer, "is1")
		assert.Equal(t, p.MaxUsers, 93)
	})
}

func Test_GetUpdatedProjects_None(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec("insert into authen_projects (id, issuer, max_users, updated) values ('p1', '2is', 93, 0)")
		updated, err := conn.GetUpdatedProjects(time.Now())
		assert.Nil(t, err)
		assert.Equal(t, len(updated), 0)
	})
}

func Test_GetUpdatedProjects_Success(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec(`
			insert into authen_projects (id, issuer, max_users, updated) values
			('p1', '', 1, unixepoch() - 500),
			('p2', '', 2, unixepoch() - 200),
			('p3', '', 3, unixepoch() - 100),
			('p4', '', 4, unixepoch() - 10)
		`)
		updated, err := conn.GetUpdatedProjects(time.Now().Add(time.Second * -105))
		assert.Nil(t, err)
		assert.Equal(t, len(updated), 2)

		// order isn't deterministic
		id1, id2 := updated[0].Id, updated[1].Id
		assert.True(t, id1 != id2)
		assert.True(t, id1 == "p3" || id1 == "p4")
		assert.True(t, id2 == "p3" || id2 == "p4")
	})
}

func withTestDB(fn func(conn Conn)) {
	conn, err := New(typed.Typed{"path": ":memory:"})
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	if err := conn.EnsureMigrations(); err != nil {
		panic(err)
	}
	fn(conn)
}
