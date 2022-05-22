package mondb

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	TeamMonitorIDSet        = "id-set"
	ParticipantMonitorIDSet = "id-set"
)

func init() {
	viper.SetDefault("team.active", time.Hour*24)
	viper.SetDefault("participant.active", time.Hour*24)
}

//GetRedisClient get our redis client
func (m *BaseMonitor) GetRedisClient() (*redis.Client, error) {
	return df.QuickClient(df.RPoolMonitoring, true)
}

func (m *BaseMonitor) MakeKey(key string) string {
	return fmt.Sprintf("monitor-%s-%s", m.MonitorName, key)
}

func (t *TeamMonitor) GetKey() string {
	return fmt.Sprintf("%d", t.TeamID)
}

func (m *BaseMonitor) GetLookupKey(parts ...string) string {
	partsStr := strings.Join(parts, "-")
	return fmt.Sprintf("monitor-%s-%s", m.MonitorName, partsStr)
}

func (t *TeamMonitor) SetUpdateMonitoring() error {
	rClient, err := t.GetRedisClient()
	if err != nil {
		return err
	}

	key := t.MakeKey(t.GetKey())

	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	if err := rClient.Set(context.Background(), key, data, viper.GetDuration("team.active")).Err(); err != nil {
		return err
	}

	// Make lookups for what we have quick - will have to verify where they point exists
	if err := rClient.SAdd(context.Background(), t.GetLookupKey(TeamMonitorIDSet), key).Err(); err != nil {
		return err
	}

	return nil
}

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
