package tasks

import (
	"context"
	"encoding/json"
	"github.com/hibiken/asynq"
)

const (
	TaskKDBUpsert = "kdb:upsert"
)

type KDBUpsert struct {
	Topic   string `json:"Topic"`
	PK      string `json:"PK"`
	RawData []byte `json:"RawData"`
}

func NewKDBUpsertTask(rawData []byte) (*asynq.Task, error) {
	payload, err := json.Marshal(KDBUpsert{RawData: rawData})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskKDBUpsert, payload), nil
}

func HandleKDBUpsertTask(ctx context.Context, t *asynq.Task) error {
	var p KDBUpsert
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// FIXME: Do stuff with `p`
	//w, err := kdb.W.Get(context.Background(), "test")

	return nil
}
