package handlers

import (
	"context"
	"encoding/json"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/gcache"
	"github.com/gin-gonic/gin"
	"github.com/mailgun/groupcache/v2"
	"github.com/ptdave20/donordrive"
	"github.com/sirupsen/logrus"
	"time"
)

func GetTeam(c *gin.Context) {
	teamID := c.Param("teamid")
	log := df.Log.WithFields(logrus.Fields{
		"team.id": teamID,
	})

	fErr := func(err error, msg string) {
		log.WithError(err).Error("Problem handling request: ", msg)
		c.JSON(500, gin.H{
			"ok":      false,
			"message": msg,
			"error":   err.Error(),
		})
	}

	gca := gcache.GlobalCache()
	teamCache, err := gca.GetGroupByName(gcache.GroupELTeam)
	if err != nil {
		fErr(err, "Couldn't get team group cache")
		return
	}

	var data []byte
	ctx, _ := context.WithTimeout(c, time.Second*7)
	if err := teamCache.Get(ctx, teamID, groupcache.AllocatingByteSliceSink(&data)); err != nil {
		fErr(err, "Couldn't get entry from team's group cache")
		return
	}

	// While we could get away without this, let's be sure the schema is right - security :)
	team := donordrive.Team{}
	if err := json.Unmarshal(data, &team); err != nil {
		fErr(err, "Couldn't unmarshal team")
		return
	}

	c.JSON(200, gin.H{
		"ok":      true,
		"error":   nil,
		"message": "ok",
		"team":    team,
	})
}
