package sqlite

import (
	"fmt"
	"testing"
	"time"

	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/utils/log"
	"src.goblgobl.com/utils/sqlite"
	"src.goblgobl.com/utils/uuid"
)

func init() {
	// silence migration info
	err := log.Configure(log.Config{
		Level: "WARN",
	})
	if err != nil {
		panic(err)
	}
}

func Test_Ping(t *testing.T) {
	withTestDB(func(conn Conn) {
		assert.Nil(t, conn.Ping())
	})
}

func Test_Clean_Totps(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec(`
			insert into authen_totps (expires, project_id, user_id, type, pending, secret) values
			(unixepoch() - 1, ?1, 'uid1', '', false, ''),
			(unixepoch() - 999, ?1, 'uid2', '', false, ''),
			(unixepoch() + 5, ?1, 'uid3', '', false, ''),
			(null, ?1, 'uid4', '', false, '')
		`, uuid.String())

		assert.Nil(t, conn.Clean())
		rows, _ := conn.RowsToMap("select user_id from authen_totps order by user_id")
		assert.Equal(t, len(rows), 2)
		assert.Equal(t, rows[0].String("user_id"), "uid3")
		assert.Equal(t, rows[1].String("user_id"), "uid4")
	})
}

func Test_Clean_Tickets(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec(`
			insert into authen_tickets (expires, uses, project_id, ticket) values
			(unixepoch() - 1, null, ?1, 't1'),
			(unixepoch() - 999, null, ?1, 't2'),
			(null, 0, ?1, 't3'),
			(unixepoch() + 5, 1, ?1, 't4'),
			(null, null, ?1, 't5')
		`, uuid.String())

		assert.Nil(t, conn.Clean())
		rows, _ := conn.RowsToMap("select ticket from authen_tickets order by ticket")
		assert.Equal(t, len(rows), 2)
		assert.Bytes(t, rows[0].Bytes("ticket"), []byte("t4"))
		assert.Bytes(t, rows[1].Bytes("ticket"), []byte("t5"))
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
		conn.MustExec(`
			insert into authen_projects (id, totp_issuer, totp_max, totp_setup_ttl, totp_secret_length, ticket_max, ticket_max_payload_length, login_log_max, login_log_max_payload_length)
			values ('p1', 'is1', 93, 121, 39, 49, 1021, 59, 1029)
		`)
		p, err := conn.GetProject("p1")
		assert.Nil(t, err)
		assert.Equal(t, p.Id, "p1")
		assert.Equal(t, p.TOTPMax, 93)
		assert.Equal(t, p.TOTPIssuer, "is1")
		assert.Equal(t, p.TOTPSecretLength, 39)
		assert.Equal(t, p.TOTPSetupTTL, 121)
		assert.Equal(t, p.TicketMax, 49)
		assert.Equal(t, p.TicketMaxPayloadLength, 1021)
		assert.Equal(t, p.LoginLogMax, 59)
		assert.Equal(t, p.LoginLogMaxPayloadLength, 1029)
	})
}

func Test_GetUpdatedProjects_None(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec(`
			insert into authen_projects (id, updated, totp_issuer, totp_max, totp_setup_ttl, totp_secret_length, ticket_max, ticket_max_payload_length, login_log_max, login_log_max_payload_length)
			values ('p1', 0, '', 0, 0, 0, 0, 0, 0, 0)
		`)
		updated, err := conn.GetUpdatedProjects(time.Now())
		assert.Nil(t, err)
		assert.Equal(t, len(updated), 0)
	})
}

func Test_GetUpdatedProjects_Success(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec(`
			insert into authen_projects (id, updated, totp_issuer, totp_max, totp_setup_ttl, totp_secret_length, ticket_max, ticket_max_payload_length, login_log_max, login_log_max_payload_length) values
			('p1', unixepoch() - 500, '', 0, 0, 0, 0, 0, 0, 0),
			('p2', unixepoch() - 200, '', 0, 0, 0, 0, 0, 0, 0),
			('p3', unixepoch() - 100, '', 0, 0, 0, 0, 0, 0, 0),
			('p4', unixepoch() - 10, '', 0, 0, 0, 0, 0, 0, 0)
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

func Test_TOTPCreate(t *testing.T) {
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
			res, err := conn.TOTPCreate(data.TOTPCreate{
				Max:       4,
				UserId:    "u1",
				Type:      tpe,
				Secret:    secret,
				Expires:   expires,
				ProjectId: projectId1,
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.TOTP_CREATE_OK)

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
			res, err := conn.TOTPCreate(data.TOTPCreate{
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
			res, err := conn.TOTPCreate(data.TOTPCreate{
				Max:       0,
				UserId:    "u4",
				Type:      "t4",
				ProjectId: projectId1,
				Secret:    []byte{23, 24},
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.TOTP_CREATE_OK)
			row, _ := conn.RowToMap("select * from authen_totps where project_id = ?1 and user_id = ?2 and type = ?3", projectId1, "u4", "t4")
			assert.Nowish(t, row.Time("created"))
			assert.Nil(t, row["expires"])
			assert.Bytes(t, row.Bytes("secret"), []byte{23, 24})
		}

		// limits are per project (there's no other totp for project2)
		{
			res, err := conn.TOTPCreate(data.TOTPCreate{
				Max:       1,
				UserId:    "u4",
				Type:      "",
				ProjectId: projectId2,
				Secret:    []byte{23, 24},
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.TOTP_CREATE_OK)
			row, _ := conn.RowToMap("select * from authen_totps where project_id = ?1 and user_id = ?2", projectId2, "u4")
			assert.Nowish(t, row.Time("created"))
			assert.Bytes(t, row.Bytes("secret"), []byte{23, 24})
		}

		// existing users+type don't increment count
		for _, expires := range []*time.Time{nil, &now} {
			res, err := conn.TOTPCreate(data.TOTPCreate{
				Max:       1,
				UserId:    "u1",
				Type:      "t1",
				Expires:   expires,
				ProjectId: projectId1,
				Secret:    []byte{33, 34},
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.TOTP_CREATE_OK)
			row, _ := conn.RowToMap("select * from authen_totps where project_id = ?1 and user_id = ?2 and type = ?3 and pending = ?4", projectId1, "u1", "t1", expires != nil)
			assert.Nowish(t, row.Time("created"))
			assert.Bytes(t, row.Bytes("secret"), []byte{33, 34})
		}

		// existing users DO increment count for a different type
		for _, expires := range []*time.Time{nil, &now} {
			res, err := conn.TOTPCreate(data.TOTPCreate{
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
	})
}

func Test_TOTPCreate_NonPending_DeletesPending(t *testing.T) {
	withTestDB(func(conn Conn) {
		projectId1 := uuid.String()

		conn.MustExec(`
			insert into authen_totps (project_id, user_id, type, pending, secret) values
			(?1, 'u1', 't1', 1, 'sec1')
		`, projectId1)

		res, err := conn.TOTPCreate(data.TOTPCreate{
			Type:      "t1",
			UserId:    "u1",
			ProjectId: projectId1,
			Secret:    []byte{99, 98},
		})
		assert.Nil(t, err)
		assert.Equal(t, res.Status, data.TOTP_CREATE_OK)

		rows, _ := conn.RowsToMap("select * from authen_totps where project_id = ?1", projectId1)
		assert.Equal(t, len(rows), 1)
		row := rows[0]
		assert.Nil(t, row["expires"])
		assert.False(t, row.Bool("pending"))
	})
}

func Test_TOTPGet(t *testing.T) {
	withTestDB(func(conn Conn) {
		conn.MustExec(`
			insert into authen_totps (project_id, user_id, type, pending, expires, secret) values
			('p1', 'u1', 't1', 1, unixepoch() - 1, 'sec1'),
			('p1', 'u2', 't2', 1, unixepoch() + 5, 'sec2'),
			('p1', 'u2', 't4', 1, unixepoch() + 5, 'sec3'),
			('p1', 'u2', 't2', 0, null, 'sec4'),
			('p2', 'u2', 't3', 0, null, 'sec5')
		`)

		assertNotFound := func(opts data.TOTPGet) {
			result, err := conn.TOTPGet(opts)
			assert.Nil(t, err)
			assert.Equal(t, result.Status, data.TOTP_GET_NOT_FOUND)
		}

		assertSecret := func(opts data.TOTPGet, secret string) {
			result, err := conn.TOTPGet(opts)
			assert.Nil(t, err)
			assert.Equal(t, result.Status, data.TOTP_GET_OK)
			assert.Bytes(t, result.Secret, []byte(secret))
		}

		// expired
		assertNotFound(data.TOTPGet{
			Type:      "t1",
			UserId:    "u1",
			ProjectId: "p1",
			Pending:   true,
		})

		// user doesn't have this type
		assertNotFound(data.TOTPGet{
			Type:      "t9",
			UserId:    "u1",
			ProjectId: "p1",
			Pending:   false,
		})

		// user doesn't have this type in non-setup
		assertNotFound(data.TOTPGet{
			Type:      "t4",
			UserId:    "u1",
			ProjectId: "p1",
			Pending:   false,
		})

		// wrong project
		assertNotFound(data.TOTPGet{
			Type:      "t3",
			UserId:    "u2",
			ProjectId: "p1",
			Pending:   false,
		})

		// not expired
		assertSecret(data.TOTPGet{
			Type:      "t2",
			UserId:    "u2",
			ProjectId: "p1",
			Pending:   true,
		}, "sec2")

		// non-setup
		assertSecret(data.TOTPGet{
			Type:      "t2",
			UserId:    "u2",
			ProjectId: "p1",
			Pending:   false,
		}, "sec4")
	})
}

func Test_TOTPDelete(t *testing.T) {
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
		deleted, err := conn.TOTPDelete(data.TOTPGet{
			Type:      "t1",
			UserId:    "u1",
			ProjectId: "p1",
		})
		assert.Nil(t, err)
		assert.Equal(t, deleted, 1)
		assertCount(5)
		assertCount(0, "p1", "u1", "t1")

		// all types for the user
		deleted, err = conn.TOTPDelete(data.TOTPGet{
			UserId:    "u2",
			AllTypes:  true,
			ProjectId: "p1",
		})
		assert.Nil(t, err)
		assert.Equal(t, deleted, 3)
		assertCount(2)
		assertCount(0, "p1", "u2")
	})
}

func Test_TicketCreate(t *testing.T) {
	withTestDB(func(conn Conn) {
		projectId1 := uuid.String()

		assertTicket := func(ticket []byte, payload []byte, expires *time.Time, uses *int) {
			t.Helper()
			row, _ := conn.RowToMap("select * from authen_tickets where project_id = ?1 and ticket = ?2", projectId1, ticket)
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
			res, err := conn.TicketCreate(data.TicketCreate{
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

			res, err := conn.TicketCreate(data.TicketCreate{
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
			res, err := conn.TicketCreate(data.TicketCreate{
				Max:       2,
				ProjectId: projectId1,
				Ticket:    []byte{9, 9, 9},
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.TICKET_CREATE_MAX)

			// no new insert
			count, _ := sqlite.Scalar[int](conn.Conn, "select count(*) from authen_tickets where project_id = ?1", projectId1)
			assert.Equal(t, count, 2)
		}
	})
}

func Test_TicketUse_Found(t *testing.T) {
	withTestDB(func(conn Conn) {
		assertTicket := func(opts data.TicketUse, payload string, uses int) {
			res, err := conn.TicketUse(opts)
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
		conn.MustExec(`
			insert into authen_tickets (project_id, ticket, payload, uses, expires) values
			('p1', ?1, 'd1', 1, null),
			('p1', ?2, null, null, null),
			('p1', ?3, null, 10, unixepoch() + 100)
		`, []byte("t1"), []byte("t2"), []byte("t3"))

		assertTicket(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t1"),
		}, "d1", 0)

		assertTicket(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t2"),
		}, "", -1)

		assertTicket(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t3"),
		}, "", 9)

	})
}

// wrong ticket, no more use or expired
func Test_TicketUse_NotFound(t *testing.T) {
	withTestDB(func(conn Conn) {
		assertNotFound := func(opts data.TicketUse) {
			res, err := conn.TicketUse(opts)
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.TICKET_USE_NOT_FOUND)
		}

		// setup our data
		conn.MustExec(`
			insert into authen_tickets (project_id, ticket, payload, uses, expires) values
			('p1', ?1, null, 2, null),
			('p1', ?2, null, null, unixepoch() - 1)
		`, []byte("t1"), []byte("t2"))

		// wrong project
		assertNotFound(data.TicketUse{
			ProjectId: "p2",
			Ticket:    []byte("t1"),
		})

		// wrong ticket
		assertNotFound(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t9"),
		})

		// expired
		assertNotFound(data.TicketUse{
			ProjectId: "p1",
			Ticket:    []byte("t2"),
		})

		{
			// important test, checks both our use limit, and that using
			// a ticket decreases the limit

			// this ticket has 2 uses
			opts := data.TicketUse{
				ProjectId: "p1",
				Ticket:    []byte("t1"),
			}
			// 1st use
			res, _ := conn.TicketUse(opts)
			assert.Equal(t, res.Status, data.TICKET_USE_OK)

			// 2nd use
			res, _ = conn.TicketUse(opts)
			assert.Equal(t, res.Status, data.TICKET_USE_OK)

			// no more uses
			assertNotFound(opts)
		}
	})
}

func Test_TicketDelete(t *testing.T) {
	withTestDB(func(conn Conn) {
		assertDelete := func(opts data.TicketUse, uses int) {
			t.Helper()
			res, err := conn.TicketDelete(opts)
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
		conn.MustExec(`
			insert into authen_tickets (project_id, ticket, payload, uses, expires) values
			('p1', ?1, null, null, null),
			('p1', ?2, null, 3, null),
			('p1', ?3, null, 0, null),
			('p1', ?4, null, 0, unixepoch() - 1)
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
	})
}

func Test_LoginLogCreate(t *testing.T) {
	withTestDB(func(conn Conn) {
		assertLoginLog := func(opts data.LoginLogCreate) {
			t.Helper()
			res, err := conn.LoginLogCreate(opts)
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.LOGIN_LOG_CREATE_OK)

			row, _ := conn.RowToMap("select * from authen_login_logs where id = $1", opts.Id)
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
				Id:        "l1",
				Status:    99,
				UserId:    "u1",
				ProjectId: uuid.String(),
			})
		}

		//payload
		{
			assertLoginLog(data.LoginLogCreate{
				Id:        "l2",
				Status:    2,
				UserId:    "u2",
				ProjectId: uuid.String(),
				Payload:   []byte("over 9000!"),
			})
		}
	})
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

	withTestDB(func(conn Conn) {
		// empty result
		{
			res, err := conn.LoginLogGet(data.LoginLogGet{})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.LOGIN_LOG_GET_OK)
			assert.Equal(t, len(res.Records), 0)
		}

		conn.MustExec(`
			insert into authen_login_logs (id, project_id, user_id, status, payload, created) values
			('id1', 'p1', 'u1', 1, null, unixepoch() - 100),
			('id2', 'p1', 'u1', 2, '{"name": "idaho"}', unixepoch() - 110),
			('id3', 'p1', 'u1', 3, null, unixepoch() - 120),
			('id4', 'p1', 'u1', 4, '{"name": "ghanima"}', unixepoch() - 130),
			('id5', 'p1', 'u2', 1, null, unixepoch()),
			('id6', 'p2', 'u1', 1, null, unixepoch());
		`)

		// first page
		{
			res, err := conn.LoginLogGet(data.LoginLogGet{
				Limit:     2,
				Offset:    0,
				UserId:    "u1",
				ProjectId: "p1",
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.LOGIN_LOG_GET_OK)
			assert.Equal(t, len(res.Records), 2)
			assertRecord(res.Records[0], "id1", 1, "")
			assertRecord(res.Records[1], "id2", 2, "idaho")
		}

		// 2nd page
		{
			res, err := conn.LoginLogGet(data.LoginLogGet{
				Limit:     2,
				Offset:    2,
				UserId:    "u1",
				ProjectId: "p1",
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.LOGIN_LOG_GET_OK)
			assert.Equal(t, len(res.Records), 2)
			assertRecord(res.Records[0], "id3", 3, "")
			assertRecord(res.Records[1], "id4", 4, "ghanima")
		}

		// Empty page
		{
			res, err := conn.LoginLogGet(data.LoginLogGet{
				Limit:     4,
				Offset:    4,
				UserId:    "u1",
				ProjectId: "p1",
			})
			assert.Nil(t, err)
			assert.Equal(t, res.Status, data.LOGIN_LOG_GET_OK)
			assert.Equal(t, len(res.Records), 0)
		}
	})
}

func withTestDB(fn func(conn Conn)) {
	conn, err := New(Config{Path: ":memory:"})
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	if err := conn.EnsureMigrations(); err != nil {
		panic(err)
	}
	fn(conn)
}
