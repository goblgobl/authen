package authen

import (
	"time"

	"src.goblgobl.com/authen/config"
	"src.goblgobl.com/authen/storage"
	"src.goblgobl.com/utils/log"
)

var Config config.Config

func Init(config config.Config) error {
	Config = config
	if seconds := config.ProjectUpdateFrequency; seconds != 0 {
		go reloadUpdatedProjects(time.Duration(seconds) * time.Second)
	}
	return nil
}

// Constantly pulling the db to figure out what project, if any,
// have changed, isn't ideal. But it's a really simple solution
// that works without any extra pieces (e.g. a queue) and works
// in an HA environment. Can always set the configuration value
// 0 to disable this behavior and rely on some other mechanism
func reloadUpdatedProjects(seconds time.Duration) {
	lastChecked := time.Now()
	for {
		time.Sleep(seconds)
		now := time.Now()
		updateProjectsUpdatedSince(lastChecked)
		lastChecked = now
	}
}

// extracted from reloadUpdatedProjects so we can test it...*eyeroll*
func updateProjectsUpdatedSince(t time.Time) {
	updatedProjects, err := storage.DB.GetUpdatedProjects(t)
	if err != nil {
		log.Error("reload_projects").Err(err).Log()
		return
	}

	for _, data := range updatedProjects {
		project := NewProject(data, true)
		Projects.Put(project.Id, project)
	}
}
