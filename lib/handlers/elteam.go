package handlers

import (
	"context"
	"encoding/json"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/gcache"
	"github.com/gin-gonic/gin"
	"github.com/mailgun/groupcache/v2"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type TeamResponse struct {
	*BaseResponse
	Team *df.CachedTeam `json:"team"`
}

type ParticipantsResponse struct {
	*BaseResponse
	Participants *df.CachedParticipants `json:"participants"`
}

func GetTeam(c *gin.Context) {
	teamID := c.Param("teamid")
	log := df.Log.WithFields(logrus.Fields{
		"team.id.str": teamID,
	})

	log.Trace("Setting up gca")
	gca := gcache.GlobalCache()
	teamCache, err := gca.GetGroupByName(gcache.GroupELTeam)
	if err != nil {
		log.WithError(err).Error("Couldn't get team group cache")
		c.JSON(http.StatusInternalServerError, NewErrorResp(err, "Couldn't get team group cache"))
		return
	}

	log.Trace("Kicking off cache get/fill")
	var data []byte
	ctx, _ := context.WithTimeout(c, time.Second*20)
	if err := teamCache.Get(ctx, teamID, groupcache.AllocatingByteSliceSink(&data)); err != nil {
		log.WithError(err).Error("Couldn't get entry from team's group cache")
		c.JSON(http.StatusInternalServerError, NewErrorResp(err, "Couldn't get entry from team's group cache"))
		return
	}

	log.Trace("Unmarshalling")
	// While we could get away without this, let's be sure the schema is right - security :)
	team := df.CachedTeam{}
	if err := json.Unmarshal(data, &team); err != nil {
		log.WithError(err).Error("Couldn't unmarshal team")
		c.JSON(http.StatusInternalServerError, NewErrorResp(err, "Couldn't unmarshal team"))
		return
	}
	log = log.WithField("team.name", team.Name)

	log.Trace("All done")
	c.JSON(http.StatusOK, TeamResponse{
		BaseResponse: NewBaseResp(),
		Team:         &team,
	})
}
func GetTeamParticipants(c *gin.Context) {
	teamID := c.Param("teamid")
	log := df.Log.WithFields(logrus.Fields{
		"participants.id.str": teamID,
	})

	log.Trace("Setting up gca")
	gca := gcache.GlobalCache()
	teamParticipantsCache, err := gca.GetGroupByName(gcache.GroupELParticipantForTeam)
	if err != nil {
		log.WithError(err).Error("Couldn't get participants participants group cache")
		c.JSON(http.StatusInternalServerError, NewErrorResp(err, "Couldn't get participants participants group cache"))
		return
	}

	log.Trace("Kicking off cache get/fill")
	var data []byte
	ctx, canc := context.WithTimeout(c, time.Second*20)
	defer canc()
	if err := teamParticipantsCache.Get(ctx, teamID, groupcache.AllocatingByteSliceSink(&data)); err != nil {
		log.WithError(err).Error("Couldn't get entry from participants's participants group cache")
		c.JSON(http.StatusInternalServerError, NewErrorResp(err, "Couldn't get entry from participants's participants group cache"))
		return
	}

	log.Trace("Unmarshalling")
	// While we could get away without this, let's be sure the schema is right - security :)
	participants := df.CachedParticipants{}
	if err := json.Unmarshal(data, &participants); err != nil {
		log.WithError(err).Error("Couldn't unmarshal participants")
		c.JSON(http.StatusInternalServerError, NewErrorResp(err, "Couldn't unmarshal participants"))
		return
	}
	log = log.WithField("participants.count", participants.Count)

	log.Trace("All done")
	c.JSON(http.StatusOK, ParticipantsResponse{
		BaseResponse: NewBaseResp(),
		Participants: &participants,
	})
}
