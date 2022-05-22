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

func (t *ParticipantMonitor) SetUpdateMonitoring() error {
	rClient, err := t.GetRedisClient()
	if err != nil {
		return err
	}

	key := t.MakeKey(t.GetKey())

	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	if err := rClient.Set(context.Background(), key, data, viper.GetDuration("participant.active")).Err(); err != nil {
		return err
	}

	// Make lookups for what we have quick - will have to verify where they point exists
	if err := rClient.SAdd(context.Background(), t.GetLookupKey(ParticipantMonitorIDSet), key).Err(); err != nil {
		return err
	}

	return nil
}
