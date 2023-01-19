package loginLogs

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"src.goblgobl.com/authen"
	"src.goblgobl.com/authen/tests"
	"src.goblgobl.com/tests/assert"
	"src.goblgobl.com/tests/request"
)

func Test_List_InvalidData(t *testing.T) {
	request.ReqT(t, authen.BuildEnv().Env()).
		QueryMap(map[string]string{
			"page":    "a",
			"perpage": "b",
		}).
		Get(List).
		ExpectValidation("user_id", 1001, "page", 1005, "perpage", 1005)

	request.ReqT(t, authen.BuildEnv().Env()).
		QueryMap(map[string]string{
			"user_id": strings.Repeat("a", 101),
			"page":    "0",
			"perpage": "-1",
		}).
		Get(List).
		ExpectValidation("user_id", 1003, "page", 1006, "perpage", 1006)
}

func Test_List_EmptyResult(t *testing.T) {
	json := request.ReqT(t, authen.BuildEnv().Env()).
		QueryMap(map[string]string{"user_id": "hi"}).
		Get(List).OK().Json
	assert.Equal(t, len(json.Objects("results")), 0)
}

func Test_List_Result(t *testing.T) {
	now := time.Now()
	projectId := tests.UUID()

	tests.Factory.LoginLog.Insert("project_id", projectId, "user_id", "u1", "status", 1, "created", now)
	tests.Factory.LoginLog.Insert("project_id", projectId, "user_id", "u1", "status", 2, "created", now.Add(time.Minute*-10))
	tests.Factory.LoginLog.Insert("project_id", projectId, "user_id", "u1", "status", 3, "created", now.Add(time.Minute*-20), "payload", "over 9000!")
	tests.Factory.LoginLog.Insert("project_id", projectId, "user_id", "u1", "status", 4, "created", now.Add(time.Minute*-30), "payload", map[string]int{"over": 9000})
	tests.Factory.LoginLog.Insert("project_id", projectId, "status", 6)
	tests.Factory.LoginLog.Insert("user_id", "u1", "status", 5)

	env := authen.BuildEnv().ProjectId(projectId).Env()
	json := request.ReqT(t, env).
		QueryMap(map[string]string{"user_id": "u1"}).
		Get(List).OK().Json

	rows := json.Objects("results")
	assert.Equal(t, len(rows), 4)
	assert.Nil(t, rows[0].Object("payload"))
	assert.Equal(t, rows[0].Int("status"), 1)
	assert.Timeish(t, rows[0].Time("created"), now)

	assert.Equal(t, rows[1].Int("status"), 2)
	assert.Nil(t, rows[1].Object("payload"))
	assert.Timeish(t, rows[1].Time("created"), now.Add(time.Minute*-10))

	assert.Equal(t, rows[2].Int("status"), 3)
	assert.Equal(t, rows[2].String("payload"), "over 9000!")
	assert.Timeish(t, rows[2].Time("created"), now.Add(time.Minute*-20))

	assert.Equal(t, rows[3].Int("status"), 4)
	assert.Equal(t, rows[3].Object("payload").Int("over"), 9000)
	assert.Timeish(t, rows[3].Time("created"), now.Add(time.Minute*-30))

	assertPage := func(page int, perpage int, statuses ...int) {
		json := request.ReqT(t, env).
			QueryMap(map[string]string{
				"user_id": "u1",
				"page":    strconv.Itoa(page),
				"perpage": strconv.Itoa(perpage),
			}).
			Get(List).OK().Json

		rows := json.Objects("results")
		assert.Equal(t, len(rows), len(statuses))
		for i, status := range statuses {
			assert.Equal(t, rows[i].Int("status"), status)
		}
	}
	assertPage(1, 2, 1, 2)
	assertPage(2, 2, 3, 4)
	assertPage(3, 2)

	assertPage(1, 3, 1, 2, 3)
	assertPage(2, 3, 4)
}
