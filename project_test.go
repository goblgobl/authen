package authen

import (
	"testing"
	"time"

	"src.goblgobl.com/authen/tests"
	"src.goblgobl.com/tests/assert"
)

func Test_Project_NextRequestId(t *testing.T) {
	seen := make(map[string]struct{}, 60)

	p := Project{requestId: 1}
	for i := 0; i < 20; i++ {
		seen[p.NextRequestId()] = struct{}{}
	}

	p = Project{requestId: 100}
	for i := 0; i < 20; i++ {
		seen[p.NextRequestId()] = struct{}{}
	}

	InstanceId += 1
	p = Project{requestId: 1}
	for i := 0; i < 20; i++ {
		seen[p.NextRequestId()] = struct{}{}
	}

	assert.Equal(t, len(seen), 60)
}

func Test_Projects_Get_Unknown(t *testing.T) {
	p, err := Projects.Get("unknown")
	assert.Nil(t, p)
	assert.Nil(t, err)
}

func Test_Projects_Get_Known(t *testing.T) {
	row := tests.Factory.Project.Insert("max_users", 76)
	id := row.String("id")

	p, err := Projects.Get(id)
	assert.Nil(t, err)
	assert.Equal(t, p.Id, id)
	assert.Equal(t, p.Capabilities.MaxUsers, 76)
	assert.Nowish(t, time.Unix(int64(p.requestId), 0))
	assert.Equal(t, string(p.logField.KV()), "pid="+id)
}
