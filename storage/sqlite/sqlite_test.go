package sqlite

import (
	"fmt"
	"testing"
	"time"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/utils/sqlite"
	"src.goblgobl.com/utils/typed"
	"src.goblgobl.com/utils/uuid"
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
		conn.MustExec("insert into authen_projects (id, issuer, totp_max, totp_setup_ttl) values ('p1', 'is1', 93, 121)")
		p, err := conn.GetProject("p1")
		assert.Nil(t, err)
		assert.Equal(t, p.Id, "p1")
		assert.Equal(t, p.Issuer, "is1")
		assert.Equal(t, p.TOTPMax, 93)
		assert.Equal(t, p.TOTPSetupTTL, 121)
	})
}

func Test_GetUpdatedProjects_None(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec("insert into authen_projects (id, issuer, totp_max, totp_setup_ttl, updated) values ('p1', '2is', 93, 122, 0)")
		updated, err := conn.GetUpdatedProjects(time.Now())
		assert.Nil(t, err)
		assert.Equal(t, len(updated), 0)
	})
}

func Test_GetUpdatedProjects_Success(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec(`
			insert into authen_projects (id, issuer, totp_max, totp_setup_ttl, updated) values
			('p1', '', 1, 11, unixepoch() - 500),
			('p2', '', 2, 12, unixepoch() - 200),
			('p3', '', 3, 13, unixepoch() - 100),
			('p4', '', 4, 14, unixepoch() - 10)
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

func Test_CreateTOTP(t *testing.T) {
	withTestDB(func(conn Conn) {
		now := time.Now()
		projectId1, projectId2 := uuid.String(), uuid.String()

		conn.MustExec(`
			insert into authen_totps (project_id, user_id, type, pending, secret) values
			(?1, 'u1', 't1', 0, 'sec1'),
			(?1, 'u2', 't2', 1, 'sec2')
		`, projectId1)

		// Adds more when less than max
		for i, expires := range []*time.Time{nil, &now} {
			secret := []byte{byte(i), byte(i)}
			tpe := fmt.Sprintf("t-%d", i)
			res, err := conn.CreateTOTP(data.CreateTOTP{
				Max:       4,
				UserId:    "u1",
				Type:      tpe,
				Secret:    secret,
				Expires:   expires,
				ProjectId: projectId1,
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.CREATE_TOTP_OK)

			row, _ := conn.RowToMap("select * from authen_totps where project_id = ?1 and user_id = ?2 and type = ?3 and pending = ?4", projectId1, "u1", tpe, expires != nil)
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
			res, err := conn.CreateTOTP(data.CreateTOTP{
				Max:       4,
				UserId:    "u4",
				Type:      "",
				Expires:   expires,
				ProjectId: projectId1,
				Secret:    []byte{13, 14},
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.CREATE_TOTP_MAX)
		}

		// 0 == no limit
		{
			res, err := conn.CreateTOTP(data.CreateTOTP{
				Max:       0,
				UserId:    "u4",
				Type:      "t4",
				ProjectId: projectId1,
				Secret:    []byte{23, 24},
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
			row, _ := conn.RowToMap("select * from authen_totps where project_id = ?1 and user_id = ?2 and type = ?3", projectId1, "u4", "t4")
			assert.Nowish(t, row.Time("created"))
			assert.Nil(t, row["expires"])
			assert.Bytes(t, row.Bytes("secret"), []byte{23, 24})
		}

		// limits are per project (there's no other totp for project2)
		{
			res, err := conn.CreateTOTP(data.CreateTOTP{
				Max:       1,
				UserId:    "u4",
				Type:      "",
				ProjectId: projectId2,
				Secret:    []byte{23, 24},
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
			row, _ := conn.RowToMap("select * from authen_totps where project_id = ?1 and user_id = ?2", projectId2, "u4")
			assert.Nowish(t, row.Time("created"))
			assert.Bytes(t, row.Bytes("secret"), []byte{23, 24})
		}

		// existing users+type don't increment count
		for _, expires := range []*time.Time{nil, &now} {
			res, err := conn.CreateTOTP(data.CreateTOTP{
				Max:       1,
				UserId:    "u1",
				Type:      "t1",
				Expires:   expires,
				ProjectId: projectId1,
				Secret:    []byte{33, 34},
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
			row, _ := conn.RowToMap("select * from authen_totps where project_id = ?1 and user_id = ?2 and type = ?3 and pending = ?4", projectId1, "u1", "t1", expires != nil)
			assert.Nowish(t, row.Time("created"))
			assert.Bytes(t, row.Bytes("secret"), []byte{33, 34})
		}

		// existing users DO increment count for a different type
		for _, expires := range []*time.Time{nil, &now} {
			res, err := conn.CreateTOTP(data.CreateTOTP{
				Max:       1,
				UserId:    "u1",
				Type:      "t-new",
				Expires:   expires,
				ProjectId: projectId1,
				Secret:    []byte{33, 34},
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.CREATE_TOTP_MAX)
		}
	})
}

func Test_CreateTOTP_NonPending_DeletesPending(t *testing.T) {
	withTestDB(func(conn Conn) {
		projectId1 := uuid.String()

		conn.MustExec(`
			insert into authen_totps (project_id, user_id, type, pending, secret) values
			(?1, 'u1', 't1', 1, 'sec1')
		`, projectId1)

		res, err := conn.CreateTOTP(data.CreateTOTP{
			Type:      "t1",
			UserId:    "u1",
			ProjectId: projectId1,
			Secret:    []byte{99, 98},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.CREATE_TOTP_OK)

		rows, _ := conn.RowsToMap("select * from authen_totps where project_id = ?1", projectId1)
		assert.Equal(t, len(rows), 1)
		row := rows[0]
		assert.Nil(t, row["expires"])
		assert.False(t, row.Bool("pending"))
	})
}

func Test_GetTOTP(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec(`
			insert into authen_totps (project_id, user_id, type, pending, expires, secret) values
			('p1', 'u1', 't1', 1, unixepoch() - 1, 'sec1'),
			('p1', 'u2', 't2', 1, unixepoch() + 5, 'sec2'),
			('p1', 'u2', 't4', 1, unixepoch() + 5, 'sec3'),
			('p1', 'u2', 't2', 0, null, 'sec4'),
			('p2', 'u2', 't3', 0, null, 'sec5')
		`)

		assertNotFound := func(opts data.GetTOTP) {
			result, err := conn.GetTOTP(opts)
			assert.Nil(t, err)
			assert.Equal(t, result.Status, data.GET_TOTP_NOT_FOUND)
		}

		assertSecret := func(opts data.GetTOTP, secret string) {
			result, err := conn.GetTOTP(opts)
			assert.Nil(t, err)
			assert.Equal(t, result.Status, data.GET_TOTP_OK)
			assert.Bytes(t, result.Secret, []byte(secret))
		}

		// expired
		assertNotFound(data.GetTOTP{
			Type:      "t1",
			UserId:    "u1",
			ProjectId: "p1",
			Pending:   true,
		})

		// user doesn't have this type
		assertNotFound(data.GetTOTP{
			Type:      "t9",
			UserId:    "u1",
			ProjectId: "p1",
			Pending:   false,
		})

		// user doesn't have this type in non-setup
		assertNotFound(data.GetTOTP{
			Type:      "t4",
			UserId:    "u1",
			ProjectId: "p1",
			Pending:   false,
		})

		// wrong project
		assertNotFound(data.GetTOTP{
			Type:      "t3",
			UserId:    "u2",
			ProjectId: "p1",
			Pending:   false,
		})

		// not expired
		assertSecret(data.GetTOTP{
			Type:      "t2",
			UserId:    "u2",
			ProjectId: "p1",
			Pending:   true,
		}, "sec2")

		// non-setup
		assertSecret(data.GetTOTP{
			Type:      "t2",
			UserId:    "u2",
			ProjectId: "p1",
			Pending:   false,
		}, "sec4")
	})
}

func Test_DeleteTOTP(t *testing.T) {
	withTestDB(func(conn Conn) {
		assertCount := func(expected int, args ...string) {
			actual := 0
			var err error

			switch len(args) {
			case 0:
				// count of all, to make sure we didn't over-delete
				actual, err = sqlite.Scalar[int](conn.Conn, "select count(*) from authen_totps")
			case 2:
				// count of all fo user
				actual, err = sqlite.Scalar[int](conn.Conn, "select count(*) from authen_totps where project_id = ?1 and user_id = ?2", args[0], args[1])
			case 3:
				// count for user+type
				actual, err = sqlite.Scalar[int](conn.Conn, "select count(*) from authen_totps where project_id = ?1 and user_id = ?2 and type = ?3", args[0], args[1], args[2])
			}
			if err != nil {
				panic(err)
			}
			assert.Equal(t, actual, expected)
		}

		conn.MustExec(`
			insert into authen_totps (project_id, user_id, type, pending, expires, secret) values
			('p1', 'u1', 't1', 1, unixepoch() - 1, 'sec1'),
			('p1', 'u2', 't2', 1, unixepoch() + 5, 'sec2'),
			('p1', 'u2', 't4', 0, unixepoch() + 5, 'sec3'),
			('p1', 'u2', 't2', 0, null, 'sec4'),
			('p2', 'u2', 't3', 1, null, 'sec5'),
			('p1', 'u3', 't1', 1, null, 'sec5')
		`)

		// specific type
		err := conn.DeleteTOTP(data.GetTOTP{
			Type:      "t1",
			UserId:    "u1",
			ProjectId: "p1",
		})
		assert.Nil(t, err)
		assertCount(5)
		assertCount(0, "p1", "u1", "t1")

		// all types for the user
		err = conn.DeleteTOTP(data.GetTOTP{
			UserId:    "u2",
			ProjectId: "p1",
		})
		assert.Nil(t, err)
		assertCount(2)
		assertCount(0, "p1", "u2")
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
