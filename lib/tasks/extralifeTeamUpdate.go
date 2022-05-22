package tasks

import (
	"context"
	"encoding/json"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/kdb"
	"github.com/fragforce/fragevents/lib/mondb"
	"github.com/hibiken/asynq"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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
	return asynq.NewTask(TaskExtraLifeTeamUpdate, payload), nil
}

func HandleExtraLifeTeamUpdateTask(ctx context.Context, t *asynq.Task) error {
	log := df.Log

	var p ExtraLifeTeamUpdate
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		log.WithError(err).Error("Problem unmarshalling payload")
		return err
	}

	tm := mondb.NewTeamMonitor(p.TeamID)

	amMon, err := tm.AmMonitoring(ctx)
	if err != nil {
		log.WithError(err).Error("Problem checking if monitored")
		return err
	}
	if !amMon {
		log.Debug("Not monitored anymore - skipping update")
		return nil
	}

	team, teamData, err := tm.GetTeam(ctx)
	if err != nil {
		log.WithError(err).Error("Problem getting team from gca")
		return err
	}
	log = log.WithField("team.name", team.Name)

	// Note the team
	kWriteTeams, err := kdb.W.Get(ctx, viper.GetString("kafka.topics.teams"))
	if err != nil {
		log.WithError(err).Error("Problem getting kafka writer for teams")
		return err
	}

	if err := kWriteTeams.WriteMessages(
		ctx,
		kafka.Message{
			Key:   tm.TeamKafkaKey(),
			Value: teamData,
			//Headers: nil,
		},
	); err != nil {
		log.WithError(err).Error("Problem writing messages to kafka team topic")
		return err
	}

	return nil
}

//NewExtraLifeTeamsUpdateTask runs an update check of all monitored teams
func NewExtraLifeTeamsUpdateTask() *asynq.Task {
	return asynq.NewTask(TaskExtraLifeTeamsUpdate, nil)
}

func HandleExtraLifeTeamsUpdateTask(ctx context.Context, t *asynq.Task) error {
	log := df.Log
	aClient := df.GetAsyncQClient()

	teamMonitors, err := mondb.GetAllTeams(ctx)
	if err != nil {
		log.WithError(err).Error("Problem getting all teams")
		return err
	}

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
		tInfo, err := aClient.Enqueue(task, asynq.MaxRetry(1))
		if err != nil {
			log.WithError(err).Error("Problem enqueuing task")
			return err
		}
		log.WithField("task.id", tInfo.ID).Trace("Task queued")
	}
	log.Trace("Done")

	return nil
}
