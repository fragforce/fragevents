package tasks

import (
	"context"
	"encoding/json"
	"github.com/hibiken/asynq"
)

const (
	TaskExtraLifeTeamUpdate = "extralife:team_update"
)

type ExtraLifeTeamUpdate struct {
	TeamID int
}

func NewExtraLifeTeamUpdateTask(teamId int) (*asynq.Task, error) {
	payload, err := json.Marshal(ExtraLifeTeamUpdate{TeamID: teamId})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskExtraLifeTeamUpdate, payload), nil
}

func HandleExtraLifeTeamUpdateTask(ctx context.Context, t *asynq.Task) error {
	//var p ExtraLifeTeamUpdate
	//if err := json.Unmarshal(t.Payload(), &p); err != nil {
	//	return err
	//}
	//gca := gcache.GlobalCache()
	//teamGC, err := gca.GetGroupByName(gcache.GroupELTeam)
	//if err != nil {
	//	return err
	//}
	//
	//t.ResultWriter().Write()

	return nil
}
