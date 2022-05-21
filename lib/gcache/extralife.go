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
	registerGroupF(GroupELTeam, 64, teamGroup)
}

func teamGroup(ctx context.Context, log *logrus.Entry, sgc *SharedGCache, key string, dest groupcache.Sink) error {
	teamID, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return err
	}

	team, err := donordrive.GetTeam(int(teamID))
	if err != nil {
		return err
	}
	res, err := json.Marshal(team)
	if err != nil {
		return err
	}

	return dest.SetBytes(res, time.Now().Add(time.Minute*5))
}
