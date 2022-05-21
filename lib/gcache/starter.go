package gcache

import (
	"context"
	"fmt"
	"github.com/fragforce/fragevents/lib/handlers"
	"github.com/fragforce/fragevents/lib/utils"
	"github.com/gin-gonic/gin"
	"github.com/mailgun/groupcache/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"strings"
)

const (
	InsecureToken = "INSECURE"
	TokenKey      = "Bearer"
)

type SecuredHeaderTransport struct {
	http.RoundTripper
	Token string
}

func (ct *SecuredHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add(TokenKey, ct.Token)
	return ct.RoundTripper.RoundTrip(req)
}

//GetPool returns pool to register to "/_groupcache/" web handler
func (c *SharedGCache) GetPool() *groupcache.HTTPPool {
	return c.pool
}

func (c *SharedGCache) createPool() error {
	log := c.log

	myIP, err := utils.GetLocalIP()
	if err != nil {
		log.WithError(err).Error("Problem getting interface ip")
		return err
	}
	log = log.WithField("my.ip", myIP)

	myPort := viper.GetInt("port") + 1
	log = log.WithField("my.port", myPort)

	myURI := fmt.Sprintf("http://%s:%d", myIP, myPort)
	log = log.WithField("my.uri", myURI)

	log.Trace("Have my uri built")

	peers, err := c.fetchPeers()
	// Make sure we're first - Not really sure if it matters
	peers = append([]string{myURI}, peers...)

	pool := groupcache.NewHTTPPoolOpts(myURI, &groupcache.HTTPPoolOptions{
		Transport: func(ctx context.Context) http.RoundTripper {
			// Warn if not in debug mode
			if !viper.GetBool("debug") && viper.GetString("groupcache.token") == InsecureToken {
				log.Warn("INSECURE GROUPCACHE TOKEN! Please ensure 'groupcache.token' is set to a random value. ")
			}

			return &SecuredHeaderTransport{
				RoundTripper: http.DefaultTransport,
				Token:        viper.GetString("groupcache.token"),
			}
		},
	})
	pool.Set(peers...)

	// Critical portion
	c.lock.Lock()
	defer c.lock.Unlock()

	c.pool = pool
	c.myURI = myURI
	c.myAddr = myIP
	c.myPort = myPort

	return nil
}

func (c *SharedGCache) fetchPeers() ([]string, error) {
	log := c.log.WithFields(logrus.Fields{
		"peers.key": viper.GetString("groupcache.peers.key"),
	})
	res, err := c.rClient.SMembers(context.Background(), viper.GetString("groupcache.peers.key")).Result()
	if err != nil {
		log.WithError(err).Error("Problem fetching the groupcache peer list")
		return res, err
	}
	log.Trace("Fetched the groupcache peer list")
	return res, nil
}

//removeMyPeer removes ourselves from the redis based peer list
func (c *SharedGCache) removeMyPeer() error {
	log := c.log.WithFields(logrus.Fields{
		"peers.key": viper.GetString("groupcache.peers.key"),
	})
	if _, err := c.rClient.SRem(context.Background(), viper.GetString("groupcache.peers.key"), c.myURI).Result(); err != nil {
		log.WithError(err).Error("Problem removing ourself from the groupcache peer list")
		return err
	}
	log.Trace("Removed ourselves to groupcache peer list")
	return nil
}

//addMyPeer adds ourselves to the redis based peer list
func (c *SharedGCache) addMyPeer() error {
	log := c.log.WithFields(logrus.Fields{
		"peers.key": viper.GetString("groupcache.peers.key"),
	})

	if _, err := c.rClient.SAdd(context.Background(), viper.GetString("groupcache.peers.key"), c.myURI).Result(); err != nil {
		log.WithError(err).Error("Problem adding ourself to groupcache peer list")
		return err
	}
	log.Trace("Added ourselves to groupcache peer list")
	return nil
}

//Shutdown our groupcache
func (c *SharedGCache) Shutdown() error {
	return c.removeMyPeer()
}

//StartRun handles the startup prep and background run of our groupcache node
func (c *SharedGCache) StartRun(r *gin.Engine) error {
	log := c.log

	// Let someone pass in a gin engine if they already have one
	if r == nil {
		r = gin.Default()
	}

	if viper.GetBool("debug") {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Add handlers
	r.Any("/_groupcache/", c.GroupCacheHandler)
	handlers.RegisterGlobalHandlers(r) // Globals only - not web ones too

	go func() {
		// TODO: Allow caller to decide if they want to start or not
		if err := r.Run(c.myURI); err != nil {
			log.WithError(err).Fatal("Problem running GIN")
		}
	}()

	// Add ourselves to the global list of groupcache URLs
	if err := c.addMyPeer(); err != nil {
		log.WithError(err).Error("Problem adding myself to peer list")
		return err
	}

	return nil
}

//GroupCacheHandler register via gin to "/_groupcache/"
func (c *SharedGCache) GroupCacheHandler(ctx *gin.Context) {
	if strings.TrimLeft(ctx.GetHeader(TokenKey), "Token ") != viper.GetString("groupcache.token") {
		// FIXME: Standardize error json
		// FIXME: Add logging

		ctx.AbortWithStatusJSON(http.StatusForbidden, map[string]string{
			"status": "error",
			"error":  "Forbidden",
		})
	}

	pool := c.GetPool()
	pool.ServeHTTP(ctx.Writer, ctx.Request)
}

func init() {
	viper.SetDefault("groupcache.token", InsecureToken)
	viper.SetDefault("groupcache.peers.key", "peers")
}
