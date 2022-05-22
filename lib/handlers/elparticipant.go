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

type ParticipantResponse struct {
	*BaseResponse
	Participant *df.CachedParticipant `json:"participant"`
}

func GetParticipant(c *gin.Context) {
	participantID := c.Param("participantid")
	log := df.Log.WithFields(logrus.Fields{
		"participant.id.str": participantID,
	})

	log.Trace("Setting up gca")
	gca := gcache.GlobalCache()
	participantCache, err := gca.GetGroupByName(gcache.GroupELParticipants)
	if err != nil {
		log.WithError(err).Error("Couldn't get participant group cache")
		c.JSON(http.StatusInternalServerError, NewErrorResp(err, "Couldn't get participant group cache"))
		return
	}

	log.Trace("Kicking off cache get/fill")
	var data []byte
	ctx, canc := context.WithTimeout(c, time.Second*20)
	defer canc()
	if err := participantCache.Get(ctx, participantID, groupcache.AllocatingByteSliceSink(&data)); err != nil {
		log.WithError(err).Error("Couldn't get entry from participant's group cache")
		c.JSON(http.StatusInternalServerError, NewErrorResp(err, "Couldn't get entry from participant's group cache"))
		return
	}

	log.Trace("Unmarshalling")
	// While we could get away without this, let's be sure the schema is right - security :)
	participant := df.CachedParticipant{}
	if err := json.Unmarshal(data, &participant); err != nil {
		log.WithError(err).Error("Couldn't unmarshal participant")
		c.JSON(http.StatusInternalServerError, NewErrorResp(err, "Couldn't unmarshal participant"))
		return
	}
	log = log.WithField("participant.name", participant.DisplayName)

	log.Trace("All done")
	c.JSON(http.StatusOK, ParticipantResponse{
		BaseResponse: NewBaseResp(),
		Participant:  &participant,
	})
}
