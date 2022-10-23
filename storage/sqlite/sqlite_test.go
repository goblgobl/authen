package sqlite

import (
	"testing"
	"time"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/tests/assert"
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

func Test_CreateTOTPSetup(t *testing.T) {
	withTestDB(func(conn Conn) {
		projectId1, projectId2 := uuid.String(), uuid.String()

		conn.MustExec(`
			insert into authen_totps (project_id, user_id, secret) values
			($1, 'u1', ''),
			($2, 'u2', '')
		`, projectId1, projectId1)

		// can add 1 more
		res, err := conn.CreateTOTPSetup(data.CreateTOTP{
			MaxUsers:  3,
			UserId:    "u3",
			ProjectId: projectId1,
			Secret:    []byte{3, 4},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
		row, _ := conn.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId1, "u3")
		assert.Nowish(t, row.Time("created"))
		assert.Bytes(t, row.Bytes("secret"), []byte{3, 4})

		// can't add any more
		res, err = conn.CreateTOTPSetup(data.CreateTOTP{
			MaxUsers:  2,
			UserId:    "u4",
			ProjectId: projectId1,
			Secret:    []byte{13, 14},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.CREATE_TOTP_MAX_USERS)

		// 0 == no limit
		res, err = conn.CreateTOTPSetup(data.CreateTOTP{
			MaxUsers:  0,
			UserId:    "u4",
			ProjectId: projectId1,
			Secret:    []byte{13, 14},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
		row, _ = conn.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId1, "u4")
		assert.Nowish(t, row.Time("created"))
		assert.Bytes(t, row.Bytes("secret"), []byte{13, 14})

		// limits are per project
		res, err = conn.CreateTOTPSetup(data.CreateTOTP{
			MaxUsers:  1,
			UserId:    "u4",
			ProjectId: projectId2,
			Secret:    []byte{23, 24},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
		row, _ = conn.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId2, "u4")
		assert.Nowish(t, row.Time("created"))
		assert.Bytes(t, row.Bytes("secret"), []byte{23, 24})

		// existing users don't increment count
		res, err = conn.CreateTOTPSetup(data.CreateTOTP{
			MaxUsers:  1,
			UserId:    "u1",
			ProjectId: projectId1,
			Secret:    []byte{33, 34},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.CREATE_TOTP_OK)
		row, _ = conn.RowToMap("select * from authen_totp_setups where project_id = $1 and user_id = $2", projectId1, "u1")
		assert.Nowish(t, row.Time("created"))
		assert.Bytes(t, row.Bytes("secret"), []byte{33, 34})
	})
}

func Test_CreateTOTP(t *testing.T) {
	userId, projectId1 := uuid.String(), uuid.String()
	withTestDB(func(conn Conn) {
		// test upsert
		for i := byte(2); i < 4; i++ {
			conn.MustExec(`
				insert into authen_totp_setups (project_id, user_id, secret) values
				(?1, ?2, '2b')
			`, projectId1, userId)

			res, err := conn.CreateTOTP(data.CreateTOTP{
				UserId:    userId,
				ProjectId: projectId1,
				Secret:    []byte{i, i},
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.CREATE_TOTP_OK)

			row, _ := conn.RowToMap("select * from authen_totp_setups where user_id = $1", userId)
			assert.Nil(t, row)

			row, _ = conn.RowToMap("select * from authen_totps where user_id = $1", userId)
			assert.Nowish(t, row.Time("created"))
			assert.Equal(t, row.String("project_id"), projectId1)
			assert.Bytes(t, row.Bytes("secret"), []byte{i, i})
		}
	})
}

func Test_GetTOTPSetup_NotFound(t *testing.T) {
	withTestDB(func(conn Conn) {
		result, err := conn.GetTOTPSetup(data.GetTOTPSetup{
			UserId:    "u1",
			ProjectId: "p1",
		})
		assert.Nil(t, err)
		assert.Equal(t, result.Status, data.GET_TOTP_SETUP_NOT_FOUND)
	})
}

func Test_GetTOTPSetup_Found(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec(`
			insert into authen_totp_setups (project_id, user_id, secret) values
			('p1', 'u1', 'bbb')
		`)

		result, err := conn.GetTOTPSetup(data.GetTOTPSetup{
			UserId:    "u1",
			ProjectId: "p1",
		})
		assert.Nil(t, err)
		assert.Equal(t, result.Status, data.GET_TOTP_SETUP_OK)
		assert.Bytes(t, result.Secret, []byte("bbb"))
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
