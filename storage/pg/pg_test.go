package pg

import (
	"fmt"
	"os"
	"testing"
	"time"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/pg"
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

	err := log.Configure(log.Config{
		Level: "WARN",
	})
	if err != nil {
		panic(err)
	}

	url := tests.PG()
	tpe := tests.StorageType()
	if tpe == "cockroach" {
		url = tests.CR()
	}

	db, err = New(Config{URL: url}, tpe)
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

func Test_Clean_Totps(t *testing.T) {
	db.MustExec("truncate table authen_totps")
	db.MustExec(`
		insert into authen_totps (expires, project_id, user_id, type, pending, secret) values
		(now() - interval '1 second', $1, 'uid1', '', false, ''),
		(now() - interval '999 second', $1, 'uid2', '', false, ''),
		(now() + interval '5 second', $1, 'uid3', '', false, ''),
		(null, $1, 'uid4', '', false, '')
	`, uuid.String())

	assert.Nil(t, db.Clean())
	rows, _ := db.RowsToMap("select user_id from authen_totps order by user_id")
	assert.Equal(t, len(rows), 2)
	assert.Equal(t, rows[0].String("user_id"), "uid3")
	assert.Equal(t, rows[1].String("user_id"), "uid4")
}

func Test_Clean_Tickets(t *testing.T) {
	db.MustExec("truncate table authen_tickets")
	db.MustExec(`
		insert into authen_tickets (expires, uses, project_id, ticket) values
		(now() - interval '1 second', null, $1, 't1'),
		(now() - interval '999 second', null,  $1, 't2'),
		(null, 0, $1, 't3'),
		(now() + interval '5 second', 1, $1, 't4'),
		(null, null, $1, 't5')
	`, uuid.String())

	assert.Nil(t, db.Clean())
	rows, _ := db.RowsToMap("select ticket from authen_tickets order by ticket")
	assert.Equal(t, len(rows), 2)
	assert.Bytes(t, rows[0].Bytes("ticket"), []byte("t4"))
	assert.Bytes(t, rows[1].Bytes("ticket"), []byte("t5"))
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
		insert into authen_projects (id, totp_issuer, totp_max, totp_setup_ttl, totp_secret_length, ticket_max, ticket_max_payload_length, login_log_max, login_log_max_payload_length)
		values ($1, 'goblgobl.com', 84, 124, 38, 49, 1022, 59, 1029)
	`, id)

	p, err := db.GetProject(id)
	assert.Nil(t, err)
	assert.Equal(t, p.Id, id)
	assert.Equal(t, p.TOTPMax, 84)
	assert.Equal(t, p.TOTPSetupTTL, 124)
	assert.Equal(t, p.TOTPSecretLength, 38)
	assert.Equal(t, p.TOTPIssuer, "goblgobl.com")
	assert.Equal(t, p.TicketMax, 49)
	assert.Equal(t, p.TicketMaxPayloadLength, 1022)
	assert.Equal(t, p.LoginLogMax, 59)
	assert.Equal(t, p.LoginLogMaxPayloadLength, 1029)
}

func Test_GetUpdatedProjects_None(t *testing.T) {
	id := uuid.String()
	db.MustExec("truncate table authen_projects")
	db.MustExec(`
		insert into authen_projects (id, updated, totp_issuer, totp_max, totp_setup_ttl, totp_secret_length, ticket_max, ticket_max_payload_length, login_log_max, login_log_max_payload_length)
		values ($1, now() - interval '1 second', '', 0, 0, 0, 0, 0, 0, 0)
	`, id)
	updated, err := db.GetUpdatedProjects(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, len(updated), 0)
}

func Test_GetUpdatedProjects_Success(t *testing.T) {
	id1, id2, id3, id4 := uuid.String(), uuid.String(), uuid.String(), uuid.String()
	db.MustExec("truncate table authen_projects")
	db.MustExec(`
		insert into authen_projects (id, updated, totp_issuer, totp_max, totp_setup_ttl, totp_secret_length, ticket_max, ticket_max_payload_length, login_log_max, login_log_max_payload_length) values
		($1, now() - interval '500 second', '', 0, 0, 0, 0, 0, 0, 0),
		($2, now() - interval '200 second', '', 0, 0, 0, 0, 0, 0, 0),
		($3, now() - interval '100 second', '', 0, 0, 0, 0, 0, 0, 0),
		($4, now() - interval '10 second', '', 0, 0, 0, 0, 0, 0, 0)
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
	deleted, err := db.TOTPDelete(data.TOTPGet{
		Type:      "t1",
		UserId:    "u1",
		ProjectId: projectId1,
	})
	assert.Nil(t, err)
	assert.Equal(t, deleted, 1)
	assertCount(5)
	assertCount(0, projectId1, "u1", "t1")

	// all types for the user
	deleted, err = db.TOTPDelete(data.TOTPGet{
		UserId:    "u2",
		AllTypes:  true,
		ProjectId: projectId1,
	})
	assert.Nil(t, err)
	assert.Equal(t, deleted, 3)

	assertCount(2)
	assertCount(0, projectId1, "u2")
}

func Test_TicketCreate(t *testing.T) {
	projectId1 := uuid.String()

	assertTicket := func(ticket []byte, payload []byte, expires *time.Time, uses *int) {
		t.Helper()
		row, _ := db.RowToMap("select * from authen_tickets where project_id = $1 and ticket = $2", projectId1, ticket)
		assert.Nowish(t, row.Time("created"))
		assert.Bytes(t, row.Bytes("payload"), payload)
		if expires == nil {
			assert.Nil(t, row["expires"])
		} else {
			assert.Timeish(t, row.Time("expires"), *expires)
		}
		if uses == nil {
			assert.Nil(t, row["uses"])
		} else {
			assert.Equal(t, row.Int("uses"), *uses)
		}
	}

	// without expiry or usage or payload
	{
		res, err := db.TicketCreate(data.TicketCreate{
			Max:       2,
			ProjectId: projectId1,
			Ticket:    []byte{1, 2, 3},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TICKET_CREATE_OK)
		assertTicket([]byte{1, 2, 3}, nil, nil, nil)
	}

	// with expiry and usage
	{
		uses := 9
		expires := time.Now().Add(time.Hour)

		res, err := db.TicketCreate(data.TicketCreate{
			Max:       0,
			ProjectId: projectId1,
			Ticket:    []byte{4, 5, 6},
			Payload:   []byte{0, 0, 1},
			Expires:   &expires,
			Uses:      &uses,
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TICKET_CREATE_OK)
		assertTicket([]byte{4, 5, 6}, []byte{0, 0, 1}, &expires, &uses)
	}

	// max reached (previous 2 tests inserted a row each)
	{
		res, err := db.TicketCreate(data.TicketCreate{
			Max:       2,
			ProjectId: projectId1,
			Ticket:    []byte{9, 9, 9},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TICKET_CREATE_MAX)

		// no new insert
		count, _ := pg.Scalar[int](db.DB, "select count(*) from authen_tickets where project_id = $1", projectId1)
		assert.Equal(t, count, 2)
	}
}

func Test_TicketUse_Found(t *testing.T) {
	projectId := uuid.String()

	assertTicket := func(opts data.TicketUse, payload string, uses int) {
		res, err := db.TicketUse(opts)
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TICKET_USE_OK)

		if uses == -1 {
			assert.Nil(t, res.Uses)
		} else {
			assert.Equal(t, *res.Uses, uses)
		}

		if payload == "" {
			assert.Nil(t, res.Payload)
		} else {
			assert.Bytes(t, *res.Payload, []byte(payload))
		}
	}

	// setup our data
	db.MustExec(`
		insert into authen_tickets (project_id, ticket, payload, uses, expires) values
		($1, $2, 'd1', 1, null),
		($1, $3, null, null, null),
		($1, $4, null, 10, now() + interval '100 seconds')
	`, projectId, []byte("t1"), []byte("t2"), []byte("t3"))

	assertTicket(data.TicketUse{
		ProjectId: projectId,
		Ticket:    []byte("t1"),
	}, "d1", 0)

	assertTicket(data.TicketUse{
		ProjectId: projectId,
		Ticket:    []byte("t2"),
	}, "", -1)

	assertTicket(data.TicketUse{
		ProjectId: projectId,
		Ticket:    []byte("t3"),
	}, "", 9)
}

// wrong ticket, no more use or expired
func Test_TicketUse_NotFound(t *testing.T) {
	projectId := uuid.String()

	assertNotFound := func(opts data.TicketUse) {
		res, err := db.TicketUse(opts)
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TICKET_USE_NOT_FOUND)
	}

	// setup our data
	db.MustExec(`
		insert into authen_tickets (project_id, ticket, payload, uses, expires) values
		($1, $2, null, 2, null),
		($1, $3, null, null, now() - interval '1 second')
	`, projectId, []byte("t1"), []byte("t2"))

	// wrong project
	assertNotFound(data.TicketUse{
		ProjectId: "p2",
		Ticket:    []byte("t1"),
	})

	// wrong ticket
	assertNotFound(data.TicketUse{
		ProjectId: projectId,
		Ticket:    []byte("t9"),
	})

	// expired
	assertNotFound(data.TicketUse{
		ProjectId: projectId,
		Ticket:    []byte("t2"),
	})

	{
		// important test, checks both our use limit, and that using
		// a ticket decreases the limit

		// this ticket has 2 uses
		opts := data.TicketUse{
			ProjectId: projectId,
			Ticket:    []byte("t1"),
		}
		// 1st use
		res, _ := db.TicketUse(opts)
		assert.Equal(t, res.Status, data.TICKET_USE_OK)

		// 2nd use
		res, _ = db.TicketUse(opts)
		assert.Equal(t, res.Status, data.TICKET_USE_OK)

		// no more uses
		assertNotFound(opts)
	}
}

func Test_TicketDelete(t *testing.T) {
	assertDelete := func(opts data.TicketUse, uses int) {
		t.Helper()
		res, err := db.TicketDelete(opts)
		assert.Nil(t, err)
		if uses == -2 {
			assert.Equal(t, res.Status, data.TICKET_USE_NOT_FOUND)
		} else if uses == -1 {
			assert.Equal(t, res.Status, data.TICKET_USE_OK)
			assert.Nil(t, res.Uses)
		} else {
			assert.Equal(t, res.Status, data.TICKET_USE_OK)
			assert.Equal(t, *res.Uses, uses)
		}
	}

	// setup our data
	db.MustExec(`
			insert into authen_tickets (project_id, ticket, payload, uses, expires) values
			('p1', $1, null, null, null),
			('p1', $2, null, 3, null),
			('p1', $3, null, 0, null),
			('p1', $4, null, 0, now() - interval '1 second')
		`, []byte("t1"), []byte("t2"), []byte("t3"), []byte("t4"))

	// not found
	{
		// wrong project
		assertDelete(data.TicketUse{
			ProjectId: "p2",
			Ticket:    []byte("t1"),
		}, -2)

		// wrong ticket
		assertDelete(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t9"),
		}, -2)

		// no more use
		assertDelete(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t3"),
		}, -2)

		// expired
		assertDelete(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t4"),
		}, -2)
	}

	// found with unlimited use
	{
		// delete with unlimited use
		assertDelete(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t1"),
		}, -1)

		// it's really deleted not, so not found
		assertDelete(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t1"),
		}, -2)
	}

	// found with use
	{
		// delete with unlimited use
		assertDelete(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t2"),
		}, 3)

		// it's really deleted not, so not found
		assertDelete(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t2"),
		}, -2)
	}
}

func Test_LoginLogCreate(t *testing.T) {
	assertLoginLog := func(opts data.LoginLogCreate) {
		t.Helper()
		res, err := db.LoginLogCreate(opts)
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.LOGIN_LOG_CREATE_OK)

		row, _ := db.RowToMap("select * from authen_login_logs where id = $1", opts.Id)
		assert.Nowish(t, row.Time("created"))
		assert.Equal(t, row.Int("status"), opts.Status)
		assert.Equal(t, row.String("user_id"), opts.UserId)
		assert.Equal(t, row.String("project_id"), opts.ProjectId)

		if opts.Payload == nil {
			assert.Nil(t, row["payload"])
		} else {
			assert.Bytes(t, row.Bytes("payload"), opts.Payload)
		}
	}

	//no payload
	{
		assertLoginLog(data.LoginLogCreate{
			Id:        uuid.String(),
			Status:    99,
			UserId:    "u1",
			ProjectId: uuid.String(),
		})
	}

	//payload
	{
		assertLoginLog(data.LoginLogCreate{
			Id:        uuid.String(),
			Status:    2,
			UserId:    "u2",
			ProjectId: uuid.String(),
			Payload:   []byte("over 9000!"),
		})
	}
}

func Test_LoginLogGet(t *testing.T) {
	assertRecord := func(actual data.LoginLogRecord, id string, status int, payloadName string) {
		t.Helper()
		assert.Equal(t, actual.Id, id)
		assert.Equal(t, actual.Status, status)
		if payloadName == "" {
			assert.Nil(t, actual.Payload)
		} else {
			payload := (actual.Payload).(map[string]any)
			assert.Equal(t, payload["name"].(string), payloadName)
		}
	}

	// empty result
	{
		res, err := db.LoginLogGet(data.LoginLogGet{
			ProjectId: uuid.String(),
		})

		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.LOGIN_LOG_GET_OK)
		assert.Equal(t, len(res.Records), 0)
	}

	projectId1, projectId2 := uuid.String(), uuid.String()
	id1, id2, id3 := uuid.String(), uuid.String(), uuid.String()
	id4, id5, id6 := uuid.String(), uuid.String(), uuid.String()
	db.MustExec(`
		insert into authen_login_logs (id, project_id, user_id, status, payload, created) values
		($1, $7, 'u1', 1, null, now() - interval '100 seconds'),
		($2, $7, 'u1', 2, '{"name": "idaho"}', now() - interval '110 seconds'),
		($3, $7, 'u1', 3, null, now() - interval '120 seconds'),
		($4, $7, 'u1', 4, '{"name": "ghanima"}', now() - interval '130 seconds'),
		($5, $7, 'u2', 1, null, now()),
		($6, $8, 'u1', 1, null, now());
	`, id1, id2, id3, id4, id5, id6, projectId1, projectId2)

	// first page
	{
		res, err := db.LoginLogGet(data.LoginLogGet{
			Limit:     2,
			Offset:    0,
			UserId:    "u1",
			ProjectId: projectId1,
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.LOGIN_LOG_GET_OK)
		assert.Equal(t, len(res.Records), 2)
		assertRecord(res.Records[0], id1, 1, "")
		assertRecord(res.Records[1], id2, 2, "idaho")
	}

	// 2nd page
	{
		res, err := db.LoginLogGet(data.LoginLogGet{
			Limit:     2,
			Offset:    2,
			UserId:    "u1",
			ProjectId: projectId1,
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.LOGIN_LOG_GET_OK)
		assert.Equal(t, len(res.Records), 2)
		assertRecord(res.Records[0], id3, 3, "")
		assertRecord(res.Records[1], id4, 4, "ghanima")
	}

	// Empty page
	{
		res, err := db.LoginLogGet(data.LoginLogGet{
			Limit:     4,
			Offset:    4,
			UserId:    "u1",
			ProjectId: projectId1,
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.LOGIN_LOG_GET_OK)
		assert.Equal(t, len(res.Records), 0)
	}
}
