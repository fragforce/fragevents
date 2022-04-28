package tasks

import (
	"context"
	"encoding/json"
	"github.com/hibiken/asynq"
)

const (
	TaskExtraLifeUpdate = "extralife:update"
)

func NewExtraLifeUpdateTask(teamId int64) (*asynq.Task, error) {
	payload, err := json.Marshal(xxxxxxx)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskExtraLifeUpdate, payload), nil
}

func HandleExtraLifeUpdateTask(ctx context.Context, t *asynq.Task) error {
	var p xxxxxxx
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	// do stuff with xxxxxxx
	return nil
}
