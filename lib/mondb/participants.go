package mondb

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
)

func (t *ParticipantMonitor) GetKey() string {
	return fmt.Sprintf("%d", t.ParticipantID)
}

//SetUpdateMonitoring turns on monitoring for team.active period
func (t *ParticipantMonitor) SetUpdateMonitoring(ctx context.Context) error {
	rClient, err := t.GetRedisClient()
	if err != nil {
		return err
	}

	key := t.MakeKey(t.GetKey())

	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	if err := rClient.Set(ctx, key, data, viper.GetDuration("participant.active")).Err(); err != nil {
		return err
	}

	// Make lookups for what we have quick - will have to verify where they point exists
	if err := rClient.SAdd(ctx, t.GetLookupKey(ParticipantMonitorIDSet), key).Err(); err != nil {
		return err
	}

	return nil
}

//AmMonitoring are we monitoring this id
func (t *ParticipantMonitor) AmMonitoring(ctx context.Context) (bool, error) {
	rClient, err := t.GetRedisClient()
	if err != nil {
		return false, err
	}

	key := t.MakeKey(t.GetKey())
	cnt, err := rClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return cnt == 1, nil
}
