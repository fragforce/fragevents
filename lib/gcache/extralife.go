package gcache

import (
	"context"
	"encoding/json"
	"github.com/mailgun/groupcache/v2"
	"github.com/ptdave20/donordrive"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

const (
	GroupELTeam = "EL-Team"
)

func init() {
	doCheckInits()
	registerGroupF(GroupELTeam, 64, teamGroup)
}

func teamGroup(ctx context.Context, log *logrus.Entry, sgc *SharedGCache, key string, dest groupcache.Sink) error {
	teamID, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		log.WithError(err).Error("Problem converting team id from str to int")
		return err
	}
	log = log.WithField("team.id", teamID)

	log.Trace("Going to fetch team from extra-life")
	team, err := donordrive.GetTeam(int(teamID))
	if err != nil {
		log.WithError(err).Error("Problem fetching team")
		return err
	}
	log = log.WithField("team.name", team.Name)
	log.Trace("Got team from extra-life")

	res, err := json.Marshal(team)
	if err != nil {
		log.WithError(err).Error("Problem marshaling team into json")
		return err
	}

	// FIXME: Dynamic timeout and/or viper based
	return dest.SetBytes(res, time.Now().Add(time.Minute*5))
}
