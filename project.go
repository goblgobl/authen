package authen

import (
	"sync/atomic"
	"time"

	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/authen/storage/data"
	"src.goblgobl.com/utils"
	"src.goblgobl.com/utils/concurrent"
	"src.goblgobl.com/utils/log"
)

var Projects concurrent.Map[*Project]

func init() {
	Projects = concurrent.NewMap[*Project](loadProject)
}

// A project instance isn't updated. If the project is changed,
// a new instance is created.
type Project struct {
	// Project-specific counter for generating the RequestId
	requestId uint32

	// Any log entry generate for this project should include
	// the pid=$id field
	logField log.Field

	Id       string
	Issuer   string `json:"issuer"`
	MaxUsers uint32 `json:"max_users"`
}

func (p *Project) NextRequestId() string {
	nextId := atomic.AddUint32(&p.requestId, 1)
	return utils.EncodeRequestId(nextId, Config.InstanceId)
}

func loadProject(id string) (*Project, error) {
	projectData, err := storage.DB.GetProject(id)
	if projectData == nil || err != nil {
		return nil, err
	}
	return createProjectFromProjectData(projectData), nil
}

func createProjectFromProjectData(projectData *data.Project) *Project {
	id := projectData.Id

	return &Project{
		Id:       id,
		Issuer:   projectData.Issuer,
		MaxUsers: projectData.MaxUsers,
		logField: log.NewField().String("pid", id).Finalize(),

		// If we let this start at 0, then restarts are likely to produce duplicates.
		// While we make no guarantees about the uniqueness of the requestId, there's
		// no reason we can't help things out a little.
		requestId: uint32(time.Now().Unix()),
	}
}
