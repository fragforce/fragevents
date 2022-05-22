package handlers

import (
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/gcache"
	"github.com/gin-gonic/gin"
	"github.com/mailgun/groupcache/v2"
	"net/http"
)

const (
	BaseRespMessageOK = "ok"
)

type BaseResponse struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
	Err     error  `json:"error,omitempty"`
}

type DetailedStatusResponse struct {
	*BaseResponse
	Caches          map[string]groupcache.Stats `json:"cache-stats"`
	CachePeersCount int                         `json:"cache-peers-count"`
}

//NewErrorResp creates a new base response - should only be used for bad calls
func NewErrorResp(err error, msg string) *BaseResponse {
	return &BaseResponse{
		Ok:      false,
		Message: msg,
		Err:     err,
	}
}

//NewBaseResp creates a new base response - should only be used for good calls
func NewBaseResp() *BaseResponse {
	return &BaseResponse{
		Ok:      true,
		Message: BaseRespMessageOK,
	}
}

func GetDetailedStatus(c *gin.Context) {
	log := df.Log.WithContext(c)
	gca := gcache.GlobalCache()

	peers, err := gca.FetchPeers()
	if err != nil {
		log.WithError(err).Error("Couldn't get cache peers")
		c.JSON(http.StatusInternalServerError, NewErrorResp(err, "Couldn't get cache peers"))
		return
	}

	// Cache Status
	groups, err := gca.GetAllGroups()
	if err != nil {
		log.WithError(err).Error("Couldn't get all groups")
		c.JSON(http.StatusInternalServerError, NewErrorResp(err, "Couldn't get all groups"))
		return
	}
	cStatus := make(map[string]groupcache.Stats)
	for _, group := range groups {
		cStatus[group.Name()] = group.Stats
	}

	c.JSON(http.StatusOK, DetailedStatusResponse{
		BaseResponse:    NewBaseResp(),
		Caches:          cStatus,
		CachePeersCount: len(peers),
	})
}
