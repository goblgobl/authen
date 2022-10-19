package authen

import (
	"testing"
	"time"

	"src.goblgobl.com/authen/tests"
	"src.goblgobl.com/tests/assert"
)

func Test_UpdateProjectsUpdatedSince(t *testing.T) {
	base := time.Now().Add(time.Minute * -60)
	row1 := tests.Factory.Project.Insert("max_users", 1, "updated", base.Add(time.Minute*-1))
	row2 := tests.Factory.Project.Insert("max_users", 2, "updated", base.Add(time.Minute))
	row3 := tests.Factory.Project.Insert("max_users", 3, "updated", base.Add(time.Minute+10))

	updateProjectsUpdatedSince(base)

	// clear the DB so we can be 100% sure these weren't lazy loaded
	tests.Factory.Project.Truncate()
	p, _ := Projects.Get(row1.String("id"))
	assert.Nil(t, p)

	p, _ = Projects.Get(row2.String("id"))
	assert.Equal(t, p.MaxUsers, 2)

	p, _ = Projects.Get(row3.String("id"))
	assert.Equal(t, p.MaxUsers, 3)
}
