package tasks

import (
	"context"
	"encoding/json"
	"github.com/hibiken/asynq"
	_ "github.com/ptdave20/donordrive"
)

const (
	TaskExtraLifeUpdate = "extralife:update"
)

type ExtraLifeUpdate struct {
	TeamID int64
}

func NewExtraLifeUpdateTask(teamId int64) (*asynq.Task, error) {
	payload, err := json.Marshal(ExtraLifeUpdate{TeamID: teamId})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskExtraLifeUpdate, payload), nil
}

func HandleExtraLifeUpdateTask(ctx context.Context, t *asynq.Task) error {
	var p ExtraLifeUpdate
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// FIXME: Do stuff with ExtraLifeUpdate.TeamID

	return nil
}
