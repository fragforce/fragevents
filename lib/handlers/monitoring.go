package handlers

import (
	"errors"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/mondb"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"sync"
)

type RegisterTypeResponse struct {
	*BaseResponse
}

type RTypeTeamRequest struct {
	TeamID int `json:"team-id"`
}

type RTTypeParticipantRequest struct {
	ParticipantID int `json:"participant-id"`
}

type TypeHandlerF func(rType string, c *gin.Context, log *logrus.Entry) (int, error)

var (
	ErrNoSuchType    = errors.New("no such register type")
	typeHandlers     map[string]TypeHandlerF
	typeHandlersLock *sync.Mutex
)

func init() {
	initTHand()
	RegisterTypeHandler(df.RTypeParticipant, RTTypeParticipantHandler)
	RegisterTypeHandler(df.RTypeTeam, RTypeTeamHandler)
}

func initTHand() { // Can be called many times
	if typeHandlersLock == nil {
		typeHandlersLock = &sync.Mutex{}
	}

	typeHandlersLock.Lock()
	if typeHandlers == nil {
		typeHandlers = make(map[string]TypeHandlerF)
	}
	typeHandlersLock.Unlock()
}

func RTypeTeamHandler(rType string, c *gin.Context, log *logrus.Entry) (statusCode int, err error) {
	tr := RTypeTeamRequest{}
	if err := c.BindJSON(&tr); err != nil {
		log.WithError(err).Info("Problem binding JSON in request")
		return http.StatusBadRequest, err
	}

	// FIXME: Add in TeamID checks

	tm := mondb.NewTeamMonitor(tr.TeamID)
	if err := tm.SetUpdateMonitoring(c); err != nil {
		log.WithError(err).Info("Problem enabling monitoring")
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func RTTypeParticipantHandler(rType string, c *gin.Context, log *logrus.Entry) (statusCode int, err error) {
	tr := RTTypeParticipantRequest{}
	if err := c.BindJSON(&tr); err != nil {
		log.WithError(err).Info("Problem binding JSON in request")
		return http.StatusBadRequest, err
	}

	// FIXME: Add in ParticipantID checks

	tm := mondb.NewParticipantMonitor(tr.ParticipantID)
	if err := tm.SetUpdateMonitoring(c, viper.GetDuration("participant.active")); err != nil {
		log.WithError(err).Info("Problem enabling monitoring")
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func RegisterTypeHandler(name string, f TypeHandlerF) {
	initTHand() // Just to be safe

	typeHandlersLock.Lock()
	defer typeHandlersLock.Unlock()

	typeHandlers[name] = f
}

func RegisterType(c *gin.Context) {
	rType := c.Param("rtype")
	log := df.Log.WithFields(logrus.Fields{
		"register.type": rType,
	}).WithContext(c)

	handlerF, ok := typeHandlers[rType]
	if !ok {
		log.WithError(ErrNoSuchType).Info("Invalid register type requested")
		c.JSON(http.StatusNotFound, NewErrorResp(ErrNoSuchType, "Invalid register type requested"))
		return
	}

	scode, err := handlerF(rType, c, log)
	log = log.WithField("ret.status.code", scode)
	if err != nil {
		log.WithError(ErrNoSuchType).Info("Invalid register type requested")
		c.JSON(scode, NewErrorResp(ErrNoSuchType, "Invalid register type requested"))
		return
	}

	// If not set then just assume 200
	if scode == 0 {
		scode = http.StatusOK
		log = log.WithField("ret.status.code", scode)
	}

	log.Trace("All done")
	c.JSON(scode, RegisterTypeResponse{
		BaseResponse: NewBaseResp(),
	})
}
