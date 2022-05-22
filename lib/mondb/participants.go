package mondb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

func (t *ParticipantMonitor) GetKey() string {
	return fmt.Sprintf("%d", t.ParticipantID)
}

//SetUpdateMonitoring turns on monitoring for team.active period
func (t *ParticipantMonitor) SetUpdateMonitoring(ctx context.Context) error {
	rClient, err := GetRedisClient()
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
	rClient, err := GetRedisClient()
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

//GetAllParticipants returns a list of all monitored participants
func GetAllParticipants(ctx context.Context) ([]*ParticipantMonitor, error) {
	rClient, err := GetRedisClient()
	if err != nil {
		return nil, err
	}

	keys, err := rClient.SMembers(ctx, GetLookupKey(ParticipantMonitorIDSet)).Result()
	if err != nil {
		return nil, err
	}

	ret := make([]*ParticipantMonitor, 0) // Can't assume len - might have to remove some
	for _, key := range keys {
		data, err := rClient.Get(ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			// Doesn't exist
			continue
		}
		if err != nil {
			// Something else happened
			return nil, err
		}

		pm, err := NewParticipantMonitorFromJSON(data)
		if err != nil {
			return nil, err
		}

		// While it existed earlier, maybe in the future other checks will be here - so do it again
		if amMon, err := pm.AmMonitoring(ctx); err != nil {
			return nil, err
		} else if !amMon {
			continue // Not monitoring
		}

		ret = append(ret, pm)
	}

	return ret, nil
}
