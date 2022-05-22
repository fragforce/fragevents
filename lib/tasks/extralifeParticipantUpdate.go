package tasks

import (
	"context"
	"encoding/json"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/mondb"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"time"
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
		if pMonitor.ParticipantID == 0 {
			log.Info("Skipping nil ParticipantID")
			continue
		}

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

//NewExtraLifeParticipantUpdateTask runs an update check for the given monitored participant
func NewExtraLifeParticipantUpdateTask(participantID int) (*asynq.Task, error) {
	if participantID == 0 {
		return nil, ErrInvalidID
	}
	payload, err := json.Marshal(ELParticipantID{ParticipantID: participantID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskExtraLifeTeamParticipantUpdate, payload, asynq.Timeout(time.Minute*20), asynq.MaxRetry(0)), nil
}

func HandleExtraLifeParticipantUpdateTask(ctx context.Context, t *asynq.Task) error {
	log := df.Log.WithField("task.type", t.Type()).WithContext(ctx)
	log.Trace("Doing participants participant update")

	p := ELParticipantID{}
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		log.WithError(err).Error("Problem unmarshalling payload")
		return err
	}
	log = log.WithFields(logrus.Fields{
		"participants.id": p.ParticipantID,
	})

	if p.ParticipantID == 0 {
		log.WithError(ErrInvalidID).Info("Invalid participant id")
		return ErrInvalidID
	}

	tm := mondb.NewParticipantMonitor(p.ParticipantID)

	log.Trace("Checking monitoring")
	amMon, err := tm.AmMonitoring(ctx)
	if err != nil {
		log.WithError(err).Error("Problem checking if monitored")
		return err
	}
	log = log.WithField("participants.monitoring", amMon)
	if !amMon {
		log.Debug("Not monitored anymore - skipping update")
		return nil
	}

	if err := tm.WriteParticipantToKafka(ctx); err != nil {
		log.WithError(err).Error("Problem writing to kafka")
		return err
	}

	return nil
}
