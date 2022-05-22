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
	TaskExtraLifeTeamUpdate  = "extralife:team_update"
	TaskExtraLifeTeamsUpdate = "extralife:teams_update"
)

type ExtraLifeTeamUpdate struct {
	TeamID int
}

//NewExtraLifeTeamUpdateTask runs an update check for the given monitored team
func NewExtraLifeTeamUpdateTask(teamId int) (*asynq.Task, error) {
	payload, err := json.Marshal(ExtraLifeTeamUpdate{TeamID: teamId})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskExtraLifeTeamUpdate, payload, asynq.MaxRetry(0)), nil
}

func HandleExtraLifeTeamUpdateTask(ctx context.Context, t *asynq.Task) error {
	log := df.Log.WithField("task.type", t.Type()).WithContext(ctx)
	log.Trace("Doing team update")

	var p ExtraLifeTeamUpdate
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
			log.WithError(err).Error("Problem enqueuing task")
			return err
		}
		log.WithField("task.id", tInfo.ID).Trace("Task queued")
	}
	log.Trace("Done with triggering el team updates")

	return nil
}
