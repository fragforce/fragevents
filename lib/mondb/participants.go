package mondb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/gcache"
	"github.com/fragforce/fragevents/lib/kdb"
	"github.com/go-redis/redis/v8"
	"github.com/mailgun/groupcache/v2"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"text/template"
	"time"
)

func init() {
	// Keep to things that are immutable for the team
	viper.SetDefault("participant.monitor.template", `{{ .ParticipantId }}-{{ .EventId }}`)
}

func (t *ParticipantMonitor) GetKey() string {
	return fmt.Sprintf("%d", t.ParticipantID)
}

func (t *ParticipantMonitor) MonitorKey() string {
	return t.MakeKey(t.GetKey())
}

//KafkaKeyForEvents is used in kafka for identity - For events topic (non-compacted) - tl;dr All data, not fetch time though
func (t *ParticipantMonitor) KafkaKeyForEvents(p *df.CachedParticipant) ([]byte, error) {
	return p.GetRawParticipantData()
}

//KafkaKeyForParticipants is used in kafka for identity - For teams topic (compacted)
func (t *ParticipantMonitor) KafkaKeyForParticipants(p *df.CachedParticipant) ([]byte, error) {
	log := df.Log.WithFields(logrus.Fields{
		"participant.id":   p.ParticipantId,
		"participant.name": p.DisplayName,
		"event.id":         p.EventId,
		"event.name":       p.EventName,
		"last-refresh":     p.GetFetchedAt(),
	})

	tplate, err := template.New(df.TextTemplateParticipantMonitor).Parse(viper.GetString("participant.monitor.template"))
	if err != nil {
		log.WithError(err).Error("Problem parsing participant monitor template")
		return nil, err
	}

	bf := new(bytes.Buffer)
	if err := tplate.Execute(bf, p); err != nil {
		log.WithError(err).Error("Problem executing participant monitor template")
		return nil, err
	}
	return bf.Bytes(), nil
}

//KafkaHeaders are used in kafka for info, routing, and debugging
func (t *ParticipantMonitor) KafkaHeaders(p *df.CachedParticipant) []kafka.Header {
	ret := make([]kafka.Header, 0)
	if p.ParticipantId != 0 {
		ret = append(ret, kafka.Header{
			Key:   df.KHeaderKeyParticipantID,
			Value: []byte(fmt.Sprintf("%d", p.ParticipantId)),
		})
	}
	if p.CampaignName != "" {
		ret = append(ret, kafka.Header{
			Key:   df.KHeaderKeyCampaignName,
			Value: []byte(p.CampaignName),
		})
	}
	if p.DisplayName != "" {
		ret = append(ret, kafka.Header{
			Key:   df.KHeaderKeyDisplayName,
			Value: []byte(p.DisplayName),
		})
	}
	if p.EventId != 0 {
		ret = append(ret, kafka.Header{
			Key:   df.KHeaderKeyEventID,
			Value: []byte(fmt.Sprintf("%d", p.EventId)),
		})
	}
	if p.EventName != "" {
		ret = append(ret, kafka.Header{
			Key:   df.KHeaderKeyEventName,
			Value: []byte(p.EventName),
		})
	}
	if p.TeamId != 0 {
		ret = append(ret, kafka.Header{
			Key:   df.KHeaderKeyTeamID,
			Value: []byte(fmt.Sprintf("%d", p.TeamId)),
		})
	}
	if p.TeamName != "" {
		ret = append(ret, kafka.Header{
			Key:   df.KHeaderKeyTeamName,
			Value: []byte(p.TeamName),
		})
	}
	ret = append(ret, kafka.Header{
		Key:   df.KHeaderKeyFetchedAt,
		Value: []byte(p.GetFetchedAt()),
	})
	return ret
}

//MakeParticipantMessages creates the kafka message(s) for the given participant - participants topic
func (t *ParticipantMonitor) MakeParticipantMessages(p *df.CachedParticipant) ([]kafka.Message, error) {
	key, err := t.KafkaKeyForParticipants(p)
	if err != nil {
		return nil, err
	}
	return []kafka.Message{
		{
			Key:     key,
			Value:   p.RawData,
			Headers: t.KafkaHeaders(p),
		},
	}, nil
}

//MakeEventsMessages creates the kafka message(s) for the given Participant - events topic
func (t *ParticipantMonitor) MakeEventsMessages(p *df.CachedParticipant) ([]kafka.Message, error) {
	key, err := t.KafkaKeyForEvents(p)
	if err != nil {
		return nil, err
	}
	return []kafka.Message{
		{
			Key:     key,
			Value:   p.RawData,
			Headers: t.KafkaHeaders(p),
		},
	}, nil
}

//SetUpdateMonitoring turns on monitoring for team.active period
func (t *ParticipantMonitor) SetUpdateMonitoring(ctx context.Context) error {
	rClient, err := GetRedisClient()
	if err != nil {
		return err
	}

	key := t.MonitorKey()

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
	log := df.Log.WithField("participant.id", t.ParticipantID)

	rClient, err := GetRedisClient()
	if err != nil {
		log.WithError(err).Error("Problem getting redis client")
		return false, err
	}

	key := t.MakeKey(t.GetKey())
	cnt, err := rClient.Exists(ctx, key).Result()
	if err != nil {
		log.WithError(err).Error("Problem checking if key exists for client monitoring")
		return false, err
	}
	if cnt == 1 {
		log.Trace("Am monitoring (direct)")
		return true, nil
	}

	// Check if we're monitored via team
	p, err := t.GetParticipant(ctx)
	if err != nil {
		log.WithError(err).Error("Problem getting participant")
		return false, err
	}

	if p.TeamId == 0 {
		log.Trace("No team set - not tracked")
		return false, nil
	}

	tm := NewTeamMonitor(p.TeamId)

	amMon, err := tm.AmMonitoring(ctx)
	if err != nil {
		log.WithError(err).Error("Problem checking monitoring")
		return false, err
	}
	log = log.WithField("team.monitoring", amMon)
	log.Trace("Using results from team monitoring")
	return amMon, nil
}

//GetAllParticipants returns a list of all monitored participants
func GetAllParticipants(ctx context.Context) ([]*ParticipantMonitor, error) {
	rClient, err := GetRedisClient()
	if err != nil {
		return nil, err
	}

	keys, err := rClient.SMembers(ctx, GetLookupKey(df.MonitorNameParticipant, ParticipantMonitorIDSet)).Result()
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

//GetParticipant gets the cached participant info
func (t *ParticipantMonitor) GetParticipant(ctx context.Context) (*df.CachedParticipant, error) {
	log := df.Log.WithField("participant.id", t.ParticipantID)
	gca := gcache.GlobalCache()
	teamPGC, err := gca.GetGroupByName(gcache.GroupELParticipants)
	if err != nil {
		log.WithError(err).Error("Problem getting gca group by name")
		return nil, err
	}

	log.Trace("Kicking off cache get/fill")
	var data []byte
	if err := teamPGC.Get(ctx, t.GetKey(), groupcache.AllocatingByteSliceSink(&data)); err != nil {
		log.WithError(err).Error("Couldn't get entry from participant's group cache")
		return nil, err
	}

	log.Trace("Unmarshalling")
	// While we could get away without this, let's be sure the schema is right - security :)
	participant := df.CachedParticipant{}
	if err := json.Unmarshal(data, &participant); err != nil {
		log.WithError(err).Error("Couldn't unmarshal participant")
		return nil, err
	}
	log = log.WithFields(logrus.Fields{
		"participant.name": participant.DisplayName,
		"participant.id":   participant.ParticipantId,
		"last-refresh":     participant.GetFetchedAt(),
		"topic.teams":      kdb.MakeTopicName(df.KTopicTeams),
		"topic.events":     kdb.MakeTopicName(df.KTopicEvents),
	})
	participant.RawData = data // Set late

	return &participant, nil
}

//WriteParticipantToKafka fetches and writes the updated info from gcache into kafka
func (t *ParticipantMonitor) WriteParticipantToKafka(ctx context.Context) error {
	log := df.Log.WithField("participants.id", t.ParticipantID)

	log.Trace("Getting participant")
	participant, err := t.GetParticipant(ctx)
	if err != nil {
		log.WithError(err).Error("Problem getting participants from gca")
		return err
	}
	log = log.WithFields(logrus.Fields{
		"participant.name": participant.DisplayName,
		"participant.id":   participant.ParticipantId,
		"last-refresh":     participant.GetFetchedAt(),
		"topic.teams":      kdb.MakeTopicName(df.KTopicTeams),
		"topic.events":     kdb.MakeTopicName(df.KTopicEvents),
	})
	log.Trace("Got participant")

	log.Trace("Recording to participants topic")
	// TODO: Maybe move this into TeamMonitor...?
	kWriteParticipants, err := kdb.W.Get(ctx, kdb.MakeTopicName(df.KTopicParticipants))
	if err != nil {
		log.WithError(err).Error("Problem getting kafka writer for participant")
		return err
	}

	msgs, err := t.MakeParticipantMessages(participant)
	if err != nil {
		log.WithError(err).Error("Problem making kafka message(s)")
		return err
	}
	c1, can1 := context.WithTimeout(ctx, time.Second*120)
	defer can1()
	if err := kWriteParticipants.WriteMessages(
		c1,
		msgs...,
	); err != nil {
		log.WithError(err).Error("Problem writing messages to kafka participants topic")
		return err
	}

	log.Trace("Recording to events topic")
	// TODO: Maybe move this into TeamMonitor...?
	kWriteEvents, err := kdb.W.Get(ctx, kdb.MakeTopicName(df.KTopicEvents))
	if err != nil {
		log.WithError(err).Error("Problem getting kafka writer for events")
		return err
	}

	msgs, err = t.MakeEventsMessages(participant)
	if err != nil {
		log.WithError(err).Error("Problem making kafka message(s)")
		return err
	}
	c2, can2 := context.WithTimeout(ctx, time.Second*120)
	defer can2()
	if err := kWriteEvents.WriteMessages(
		c2,
		msgs...,
	); err != nil {
		log.WithError(err).Error("Problem writing messages to kafka events topic")
		return err
	}

	log.Trace("Done with participant update")

	return nil
}
