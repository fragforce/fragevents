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
	GroupELTeam = "EL-Team"
)

func init() {
	donordrive.SetBaseUrl(donordrive.ExtraLifeUrl)
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

	log.Warn("Going to fetch team from extra-life")
	team, err := donordrive.GetTeam(int(teamID))
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
