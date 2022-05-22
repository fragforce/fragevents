package mondb

import (
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

func (m *BaseMonitor) GetLookupKey(parts ...string) string {
	partsStr := strings.Join(parts, "-")
	return fmt.Sprintf("monitor-%s-%s", m.MonitorName, partsStr)
}
