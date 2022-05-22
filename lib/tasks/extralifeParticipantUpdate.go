package tasks

import (
	"context"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/mondb"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

//NewExtraLifeParticipantsUpdateTask runs an update check of all monitored participants
func NewExtraLifeParticipantsUpdateTask() *asynq.Task {
	return asynq.NewTask(TaskExtraLifeParticipantsUpdate, nil, asynq.MaxRetry(0))
}

func HandleExtraLifeParticipantsUpdateTask(ctx context.Context, t *asynq.Task) error {
	log := df.Log.WithField("task.type", t.Type()).WithContext(ctx)
	aClient := df.GetAsyncQClient()

	pMonitors, err := mondb.GetAllParticipants(ctx)
	if err != nil {
		log.WithError(err).Error("Problem getting all participants")
		return err
	}
	log = log.WithField("participants.count", len(pMonitors))

	for idx, pMonitor := range pMonitors {
		log := log.WithFields(logrus.Fields{
			"participant.id": pMonitor.ParticipantID,
			"monitor.name":   pMonitor.MonitorName,
			"monitor.idx":    idx,
		})
		task, err := NewExtraLifeParticipantUpdateTask(pMonitor.ParticipantID)
		if err != nil {
			log.WithError(err).Error("Problem creating participant update task")
			return err
		}

		tInfo, err := aClient.Enqueue(task)
		if err != nil {
			log.WithError(err).Error("Problem enqueuing task")
			return err
		}
		log.WithField("task.id", tInfo.ID).Trace("Task queued")
	}
	log.Trace("Done with triggering el participant updates")

	return nil
}
