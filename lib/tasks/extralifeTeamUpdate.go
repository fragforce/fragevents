package tasks

import (
	"context"
	"encoding/json"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/kdb"
	"github.com/fragforce/fragevents/lib/mondb"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	TaskExtraLifeTeamUpdate            = "extralife:team_update"
	TaskExtraLifeTeamParticipantUpdate = "extralife:team_participant_update"
	TaskExtraLifeParticipantUpdate     = "extralife:participant_update"
	TaskExtraLifeTeamsUpdate           = "extralife:teams_update"
)

type ELTeamID struct {
	TeamID int
}

type ELParticipantID struct {
	ParticipantID int
}

//NewExtraLifeTeamUpdateTask runs an update check for the given monitored team
func NewExtraLifeTeamUpdateTask(teamId int) (*asynq.Task, error) {
	payload, err := json.Marshal(ELTeamID{TeamID: teamId})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskExtraLifeTeamUpdate, payload, asynq.Timeout(time.Minute*10), asynq.MaxRetry(0)), nil
}

func HandleExtraLifeTeamUpdateTask(ctx context.Context, t *asynq.Task) error {
	log := df.Log.WithField("task.type", t.Type()).WithContext(ctx)
	log.Trace("Doing team update")

	var p ELTeamID
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		log.WithError(err).Error("Problem unmarshalling payload")
		return err
	}
	log = log.WithFields(logrus.Fields{
		"team.id": p.TeamID,
	})

	// TODO: Maybe move this into TeamMonitor...?
	tm := mondb.NewTeamMonitor(p.TeamID)

	log.Trace("Checking monitoring")
	amMon, err := tm.AmMonitoring(ctx)
	if err != nil {
		log.WithError(err).Error("Problem checking if monitored")
		return err
	}
	log = log.WithField("team.monitoring", amMon)
	if !amMon {
		log.Debug("Not monitored anymore - skipping update")
		return nil
	}

	log.Trace("Getting team")
	team, err := tm.GetTeam(ctx)
	if err != nil {
		log.WithError(err).Error("Problem getting team from gca")
		return err
	}
	log = log.WithFields(logrus.Fields{
		"team.id":      team.TeamID,
		"team.name":    team.Name,
		"event.id":     team.EventID,
		"event.name":   team.EventName,
		"last-refresh": team.GetFetchedAt(),
		"topic.teams":  kdb.MakeTopicName(df.KTopicTeams),
		"topic.events": kdb.MakeTopicName(df.KTopicEvents),
	})
	log.Trace("Got team")

	log.Trace("Recording to teams topic")
	// TODO: Maybe move this into TeamMonitor...?
	kWriteTeams, err := kdb.W.Get(ctx, kdb.MakeTopicName(df.KTopicTeams))
	if err != nil {
		log.WithError(err).Error("Problem getting kafka writer for teams")
		return err
	}

	msgs, err := tm.MakeTeamMessages(team)
	if err != nil {
		log.WithError(err).Error("Problem making kafka message(s)")
		return err
	}
	c1, can1 := context.WithTimeout(ctx, time.Second*120)
	defer can1()
	if err := kWriteTeams.WriteMessages(
		c1,
		msgs...,
	); err != nil {
		log.WithError(err).Error("Problem writing messages to kafka team topic")
		return err
	}

	log.Trace("Recording to events topic")
	// TODO: Maybe move this into TeamMonitor...?
	kWriteEvents, err := kdb.W.Get(ctx, kdb.MakeTopicName(df.KTopicEvents))
	if err != nil {
		log.WithError(err).Error("Problem getting kafka writer for events")
		return err
	}

	msgs, err = tm.MakeEventsMessages(team)
	if err != nil {
		log.WithError(err).Error("Problem making kafka message(s)")
		return err
	}
	c2, can2 := context.WithTimeout(ctx, time.Second*120)
	defer can2()
	if err := kWriteEvents.WriteMessages(
		c2,
		msgs...,
	); err != nil {
		log.WithError(err).Error("Problem writing messages to kafka events topic")
		return err
	}

	log.Trace("Done with team update")
	return nil
}

//NewExtraLifeTeamsUpdateTask runs an update check of all monitored teams
func NewExtraLifeTeamsUpdateTask() *asynq.Task {
	return asynq.NewTask(TaskExtraLifeTeamsUpdate, nil, asynq.MaxRetry(0))
}

func HandleExtraLifeTeamsUpdateTask(ctx context.Context, t *asynq.Task) error {
	log := df.Log.WithField("task.type", t.Type()).WithContext(ctx)
	aClient := df.GetAsyncQClient()

	teamMonitors, err := mondb.GetAllTeams(ctx)
	if err != nil {
		log.WithError(err).Error("Problem getting all teams")
		return err
	}
	log = log.WithField("teams.count", len(teamMonitors))

	for _, teamMonitor := range teamMonitors {
		log := log.WithFields(logrus.Fields{
			"team.id":      teamMonitor.TeamID,
			"monitor.name": teamMonitor.MonitorName,
		})
		task, err := NewExtraLifeTeamUpdateTask(teamMonitor.TeamID)
		if err != nil {
			log.WithError(err).Error("Problem creating team update task")
			return err
		}

		tInfo, err := aClient.Enqueue(task)
		if err != nil {
			log.WithError(err).Error("Problem enqueuing task 1")
			return err
		}
		log.WithField("task.id", tInfo.ID).Trace("Task 1 queued")

		task2, err := NewExtraLifeTeamUpdateParticipantTask(teamMonitor.TeamID)
		if err != nil {
			log.WithError(err).Error("Problem creating team participant update task")
			return err
		}

		tInfo2, err := aClient.Enqueue(task2)
		if err != nil {
			log.WithError(err).Error("Problem enqueuing task 2")
			return err
		}
		log.WithField("task.id", tInfo2.ID).Trace("Task 2 queued")
	}
	log.Trace("Done with triggering el team updates")

	return nil
}

//NewExtraLifeTeamUpdateParticipantTask runs an update check for the given monitored team - Runs over participants
func NewExtraLifeTeamUpdateParticipantTask(teamID int) (*asynq.Task, error) {
	payload, err := json.Marshal(ELTeamID{TeamID: teamID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskExtraLifeTeamParticipantUpdate, payload, asynq.Timeout(time.Minute*20), asynq.MaxRetry(0)), nil
}

func HandleExtraLifeTeamUpdateParticipantTask(ctx context.Context, t *asynq.Task) error {
	log := df.Log.WithField("task.type", t.Type()).WithContext(ctx)
	log.Trace("Doing participants participant update")
	aClient := df.GetAsyncQClient()

	var p ELTeamID
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		log.WithError(err).Error("Problem unmarshalling payload")
		return err
	}
	log = log.WithFields(logrus.Fields{
		"team.id": p.TeamID,
	})

	// TODO: Maybe move this into TeamMonitor...?
	tm := mondb.NewTeamMonitor(p.TeamID)

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

	participants, err := tm.GetTeamParticipants(ctx)
	if err != nil {
		log.WithError(err).Error("Problem getting team's participants")
		return err
	}

	for _, p := range participants.Participants {
		task, err := NewExtraLifeParticipantUpdateTask(p.ParticipantId)
		if err != nil {
			log.WithError(err).Error("Problem creating participant update task")
			return err
		}

		if tInfo, err := aClient.Enqueue(task); err != nil {
			log.WithError(err).Error("Problem enqueuing task 1")
			return err
		} else {
			log = log.WithField("task.id", tInfo.ID)
		}
		log.Trace("Queued up participate update task")
	}

	log.Trace("Done with participants update")
	return nil
}

//NewExtraLifeParticipantUpdateTask runs an update check for the given monitored participant
func NewExtraLifeParticipantUpdateTask(participantID int) (*asynq.Task, error) {
	payload, err := json.Marshal(ELParticipantID{ParticipantID: participantID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskExtraLifeTeamParticipantUpdate, payload, asynq.Timeout(time.Minute*20), asynq.MaxRetry(0)), nil
}

func HandleExtraLifeParticipantUpdateTask(ctx context.Context, t *asynq.Task) error {
	log := df.Log.WithField("task.type", t.Type()).WithContext(ctx)
	log.Trace("Doing participants participant update")

	var p ELParticipantID
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		log.WithError(err).Error("Problem unmarshalling payload")
		return err
	}
	log = log.WithFields(logrus.Fields{
		"participants.id": p.ParticipantID,
	})

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
