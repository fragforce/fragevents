package tasks

import (
	"github.com/fragforce/fragevents/lib/df"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

func RegisterSched(scheduler *asynq.Scheduler) {
	log := df.Log
	//https://github.com/hibiken/asynq/wiki/Periodic-Tasks#entries

	// Quickies
	registerUpdateJob(log, scheduler, TaskExtraLifeTeamsUpdate)

}

//registerUpdateJob helper to register quick update tasks
func registerUpdateJob(log *logrus.Entry, scheduler *asynq.Scheduler, taskName string) {
	// FIXME: Move period to viper
	entryID, err := scheduler.Register("@every 10s", NewExtraLifeTeamsUpdateTask()) // Never retry it!
	if err != nil {
		log.WithError(err).Fatal("Couldn't register cron job")
	}
	log.WithField("entry.id", entryID).Trace("Registered job")
}
