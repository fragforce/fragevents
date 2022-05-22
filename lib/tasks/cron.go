package tasks

import (
	"github.com/fragforce/fragevents/lib/df"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"time"
)

func RegisterSched(scheduler *asynq.Scheduler) {
	log := df.Log
	//https://github.com/hibiken/asynq/wiki/Periodic-Tasks#entries

	// Quick, very frequently run update jobs
	// FIXME: Move durations to viper
	registerUpdateJob(log, scheduler, NewExtraLifeTeamsUpdateTask(), time.Second*60)
	registerUpdateJob(log, scheduler, NewExtraLifeParticipantsUpdateTask(), time.Second*120)
}

//registerUpdateJob helper to register quick update tasks
func registerUpdateJob(log *logrus.Entry, scheduler *asynq.Scheduler, task *asynq.Task, cadence time.Duration) {
	// FIXME: Move period to viper
	entryID, err := scheduler.Register("@every "+cadence.String(), task) // Never retry it!
	if err != nil {
		log.WithError(err).Fatal("Couldn't register cron job")
	}
	log.WithField("entry.id", entryID).Trace("Registered job")
}
