package mondb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/gcache"
	"github.com/go-redis/redis/v8"
	"github.com/mailgun/groupcache/v2"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"text/template"
)

func init() {
	// Keep to things that are immutable for the team
	viper.SetDefault("team.monitor.template", `{{ .TeamID }}-{{ .EventID }}`)
}

func (t *TeamMonitor) GetKey() string {
	return fmt.Sprintf("%d", t.TeamID)
}

func (t *TeamMonitor) MonitorKey() string {
	return t.MakeKey(t.GetKey())
}

//TeamKafkaKeyEvents is used in kafka for identity - For events topic (compacted) - tl;dr All data, not fetch time though
func (t *TeamMonitor) TeamKafkaKeyEvents(team *df.CachedTeam) ([]byte, error) {
	return team.GetRawTeamData()
}

//TeamKafkaKeyTeams is used in kafka for identity - For teams topic (compacted)
func (t *TeamMonitor) TeamKafkaKeyTeams(team *df.CachedTeam) ([]byte, error) {
	log := df.Log.WithFields(logrus.Fields{
		"team.id":      team.TeamID,
		"team.name":    team.Name,
		"event.id":     team.EventID,
		"event.name":   team.EventName,
		"last-refresh": team.GetFetchedAt(),
	})

	tplate, err := template.New(df.TextTemplateNameTeamMonitor).Parse(viper.GetString("team.monitor.template"))
	if err != nil {
		log.WithError(err).Error("Problem parsing team monitor template")
		return nil, err
	}

	bf := new(bytes.Buffer)
	if err := tplate.Execute(bf, team); err != nil {
		log.WithError(err).Error("Problem executing team monitor template")
		return nil, err
	}
	return bf.Bytes(), nil
}

//TeamKafkaHeaders are used in kafka for info, routing, and debugging
func (t *TeamMonitor) TeamKafkaHeaders(team *df.CachedTeam) []kafka.Header {
	ret := make([]kafka.Header, 0)
	if team.TeamID != nil {
		ret = append(ret, kafka.Header{
			Key:   df.KHeaderKeyTeamID,
			Value: []byte(fmt.Sprintf("%d", *team.TeamID)),
		})
	}
	if team.Name != nil {
		ret = append(ret, kafka.Header{
			Key:   df.KHeaderKeyTeamName,
			Value: []byte(*team.Name),
		})
	}
	if team.EventID != nil {
		ret = append(ret, kafka.Header{
			Key:   df.KHeaderKeyEventID,
			Value: []byte(fmt.Sprintf("%d", *team.EventID)),
		})
	}
	if team.EventName != nil {
		ret = append(ret, kafka.Header{
			Key:   df.KHeaderKeyEventName,
			Value: []byte(*team.EventName),
		})
	}
	ret = append(ret, kafka.Header{
		Key:   df.KHeaderKeyFetchedAt,
		Value: []byte(team.GetFetchedAt()),
	})
	return ret
}

//MakeTeamMessages creates the kafka message(s) for the given team - teams topic
func (t *TeamMonitor) MakeTeamMessages(team *df.CachedTeam) ([]kafka.Message, error) {
	key, err := t.TeamKafkaKeyTeams(team)
	if err != nil {
		return nil, err
	}
	return []kafka.Message{
		{
			Key:     key,
			Value:   team.RawData,
			Headers: t.TeamKafkaHeaders(team),
		},
	}, nil
}

//MakeEventsMessages creates the kafka message(s) for the given team - events topic
func (t *TeamMonitor) MakeEventsMessages(team *df.CachedTeam) ([]kafka.Message, error) {
	key, err := t.TeamKafkaKeyEvents(team)
	if err != nil {
		return nil, err
	}
	return []kafka.Message{
		{
			Key:     key,
			Value:   team.RawData,
			Headers: t.TeamKafkaHeaders(team),
		},
	}, nil
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
	log := df.Log

	rClient, err := GetRedisClient()
	if err != nil {
		log.WithError(err).Error("Problem getting redis client")
		return nil, err
	}

	sKey := GetLookupKey(df.MonitorNameTeam, TeamMonitorIDSet)
	log = log.WithField("teams.key", sKey)

	keys, err := rClient.SMembers(ctx, sKey).Result()
	if err != nil {
		log.WithError(err).Error("Problem getting monitor id set")
		return nil, err
	}
	log = log.WithField("set.len", len(keys))

	ret := make([]*TeamMonitor, 0) // Can't assume len - might have to remove some
	for _, key := range keys {
		log := log.WithField("key", key)
		data, err := rClient.Get(ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			log.Trace("Doesn't exist")
			continue
		}
		if err != nil {
			// Something else happened
			log.WithError(err).Error("Problem with member key lookup")
			return nil, err
		}

		tm, err := NewTeamMonitorFromJSON(data)
		if err != nil {
			log.WithError(err).Error("Problem with turning json data into team")
			return nil, err
		}

		// While it existed earlier, maybe in the future other checks will be here - so do it again
		if amMon, err := tm.AmMonitoring(ctx); err != nil {
			log.WithError(err).Info("Problem getting monitoring info")
			return nil, err
		} else if !amMon {
			log.WithError(err).Info("Not monitoring anymore")
			continue // Not monitoring
		}

		ret = append(ret, tm)
	}

	return ret, nil
}

func (t *TeamMonitor) GetTeam(ctx context.Context) (*df.CachedTeam, error) {
	log := df.Log
	gca := gcache.GlobalCache()
	teamGC, err := gca.GetGroupByName(gcache.GroupELTeam)
	if err != nil {
		log.WithError(err).Error("Problem getting gca group by name")
		return nil, err
	}

	log.Trace("Kicking off cache get/fill")
	var data []byte
	if err := teamGC.Get(ctx, t.GetKey(), groupcache.AllocatingByteSliceSink(&data)); err != nil {
		log.WithError(err).Error("Couldn't get entry from team's group cache")
		return nil, err
	}

	log.Trace("Unmarshalling")
	// While we could get away without this, let's be sure the schema is right - security :)
	team := df.CachedTeam{}
	if err := json.Unmarshal(data, &team); err != nil {
		log.WithError(err).Error("Couldn't unmarshal team")
		return nil, err
	}
	log = log.WithField("team.name", team.Name)
	team.RawData = data // Set late

	return &team, nil
}
