package gcache

import (
	"context"
	"encoding/json"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/mailgun/groupcache/v2"
	"github.com/ptdave20/donordrive"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

const (
	GroupELTeam               = "EL-Team"
	GroupELParticipants       = "EL-Participants"
	GroupELParticipantForTeam = "EL-Participants-For-Team"
)

func init() {
	donordrive.SetBaseUrl(donordrive.ExtraLifeUrl)
	doCheckInits()
	registerGroupF(GroupELTeam, 64, teamGroup)
	registerGroupF(GroupELParticipants, 64, participantGroup)
	registerGroupF(GroupELParticipantForTeam, 64, participantsForTeamGroup)
}

func teamGroup(ctx context.Context, log *logrus.Entry, sgc *SharedGCache, key string, dest groupcache.Sink) error {
	teamID, err := strconv.ParseInt(key, 10, 32)
	if err != nil {
		log.WithError(err).Error("Problem converting team id from str to int")
		return err
	}
	log = log.WithField("team.id", teamID)

	log.Warn("Going to fetch team from extra-life")
	team, err := donordrive.GetTeam(int(teamID)) // Need int not int64
	if err != nil {
		log.WithError(err).Error("Problem fetching team")
		return err
	}
	log = log.WithField("team.name", team.Name)
	log.Warn("Got team from extra-life")

	cTeam := df.CachedTeam{
		Team:      *team,
		FetchedAt: time.Now().UTC(),
	}
	res, err := json.Marshal(&cTeam)
	if err != nil {
		log.WithError(err).Error("Problem marshaling team into json")
		return err
	}
	log.Warn("Done")
	// FIXME: Dynamic timeout and/or viper based
	return dest.SetBytes(res, time.Now().Add(time.Minute*5))
}

func participantsForTeamGroup(ctx context.Context, log *logrus.Entry, sgc *SharedGCache, key string, dest groupcache.Sink) error {
	teamID, err := strconv.ParseInt(key, 10, 32)
	if err != nil {
		log.WithError(err).Error("Problem converting team id from str to int")
		return err
	}
	log = log.WithField("team.id", teamID)

	log.Warn("Going to fetch team participants from extra-life")
	tps, err := donordrive.GetTeamParticipants(int(teamID)) // Need int not int64
	if err != nil {
		log.WithError(err).Error("Problem fetching team participants")
		return err
	}
	log = log.WithField("participants.count", len(tps))
	log.Warn("Got team participants from extra-life")

	cTeam := df.CachedParticipants{
		Participants: tps,
		Count:        len(tps),
		FetchedAt:    time.Now().UTC(),
	}
	res, err := json.Marshal(&cTeam)
	if err != nil {
		log.WithError(err).Error("Problem marshaling participants team into json")
		return err
	}
	log.Warn("Done")
	// FIXME: Dynamic timeout and/or viper based
	return dest.SetBytes(res, time.Now().Add(time.Minute*10))
}

func participantGroup(ctx context.Context, log *logrus.Entry, sgc *SharedGCache, key string, dest groupcache.Sink) error {
	participantID, err := strconv.ParseInt(key, 10, 32)
	if err != nil {
		log.WithError(err).Error("Problem converting participant id from str to int")
		return err
	}
	log = log.WithField("participant.id", participantID)

	log.Warn("Going to fetch participant from extra-life")
	participant, err := donordrive.GetParticipantDetails(int(participantID)) // Need int not int64
	if err != nil {
		log.WithError(err).Error("Problem fetching participant")
		return err
	}
	log = log.WithField("participants.name.display", participant.DisplayName)
	log.Warn("Got participant details from extra-life")

	cTeam := df.CachedParticipant{
		Participant: *participant,
		FetchedAt:   time.Now().UTC(),
	}
	res, err := json.Marshal(&cTeam)
	if err != nil {
		log.WithError(err).Error("Problem marshaling participants team into json")
		return err
	}
	log.Warn("Done")
	// FIXME: Dynamic timeout and/or viper based
	return dest.SetBytes(res, time.Now().Add(time.Minute*5))
}
