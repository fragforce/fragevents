package mondb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/gcache"
	"github.com/go-redis/redis/v8"
	"github.com/mailgun/groupcache/v2"
	"github.com/ptdave20/donordrive"
	"github.com/spf13/viper"
)

func (t *TeamMonitor) GetKey() string {
	return fmt.Sprintf("%d", t.TeamID)
}

func (t *TeamMonitor) MonitorKey() string {
	return t.MakeKey(t.GetKey())
}

func (t *TeamMonitor) TeamKafkaKey() []byte {
	return []byte(t.MonitorKey())
}

//SetUpdateMonitoring turns on monitoring for team.active period
func (t *TeamMonitor) SetUpdateMonitoring(ctx context.Context) error {
	rClient, err := GetRedisClient()
	if err != nil {
		return err
	}

	key := t.MonitorKey()

	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	if err := rClient.Set(ctx, key, data, viper.GetDuration("team.active")).Err(); err != nil {
		return err
	}

	// Make lookups for what we have quick - will have to verify where they point exists
	if err := rClient.SAdd(ctx, t.GetLookupKey(TeamMonitorIDSet), key).Err(); err != nil {
		return err
	}

	return nil
}

//AmMonitoring are we monitoring this id
func (t *TeamMonitor) AmMonitoring(ctx context.Context) (bool, error) {
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

//GetAllTeams returns a list of all monitored teams
func GetAllTeams(ctx context.Context) ([]*TeamMonitor, error) {
	rClient, err := GetRedisClient()
	if err != nil {
		return nil, err
	}

	keys, err := rClient.SMembers(ctx, GetLookupKey(TeamMonitorIDSet)).Result()
	if err != nil {
		return nil, err
	}

	ret := make([]*TeamMonitor, 0) // Can't assume len - might have to remove some
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

		tm, err := NewTeamMonitorFromJSON(data)
		if err != nil {
			return nil, err
		}

		// While it existed earlier, maybe in the future other checks will be here - so do it again
		if amMon, err := tm.AmMonitoring(ctx); err != nil {
			return nil, err
		} else if !amMon {
			continue // Not monitoring
		}

		ret = append(ret, tm)
	}

	return ret, nil
}

func (t *TeamMonitor) GetTeam(ctx context.Context) (*donordrive.Team, []byte, error) {
	log := df.Log
	gca := gcache.GlobalCache()
	teamGC, err := gca.GetGroupByName(gcache.GroupELTeam)
	if err != nil {
		log.WithError(err).Error("Problem getting gca group by name")
		return nil, nil, err
	}

	log.Trace("Kicking off cache get/fill")
	var data []byte
	if err := teamGC.Get(ctx, t.GetKey(), groupcache.AllocatingByteSliceSink(&data)); err != nil {
		log.WithError(err).Error("Couldn't get entry from team's group cache")
		return nil, nil, err
	}

	log.Trace("Unmarshalling")
	// While we could get away without this, let's be sure the schema is right - security :)
	team := donordrive.Team{}
	if err := json.Unmarshal(data, &team); err != nil {
		log.WithError(err).Error("Couldn't unmarshal team")
		return nil, nil, err
	}
	log = log.WithField("team.name", team.Name)

	return &team, data, nil
}
