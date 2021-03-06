package gcache

import (
	"context"
	"errors"
	"fmt"
	"github.com/fragforce/fragevents/lib/handler_global"
	"github.com/fragforce/fragevents/lib/utils"
	"github.com/gin-gonic/gin"
	"github.com/mailgun/groupcache/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	InsecureToken = "INSECURE"
	TokenKey      = "Authorization"
)

type SecuredHeaderTransport struct {
	http.RoundTripper
	Token string
	Ctx   context.Context
}

var (
	ErrBadStatusCode = errors.New("bad status code returned")
)

func init() {
	doCheckInits()
	viper.SetDefault("groupcache.token", InsecureToken)
	viper.SetDefault("groupcache.peers.key", "peers")
	viper.SetDefault("groupcache.peer.update", time.Second*10)
	viper.SetDefault("groupcache.wan.timeout", time.Second*5)
	viper.SetDefault("peer.failed.sleep", time.Second*5) // How long to sleep between peer checks
	viper.SetDefault("peer.failed.count", 3)             // How many checks must fail before a peer is removed
}

func (ct *SecuredHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add(TokenKey, fmt.Sprintf("Bearer %s", ct.Token))
	ctx, canc := context.WithTimeout(ct.Ctx, time.Second*3)
	defer canc()
	return ct.RoundTripper.RoundTrip(req.WithContext(ctx))
}

//GetPool returns pool to register to "/_groupcache/" web handler
func (c *SharedGCache) GetPool() *groupcache.HTTPPool {
	return c.pool
}

func (c *SharedGCache) createPool() error {
	log := c.log

	//ctx, canc := context.WithTimeout(context.Background(), viper.GetDuration("groupcache.wan.timeout"))
	//defer canc()
	//myIP, err := utils.GetExternalIP(ctx)
	myIP, err := utils.GetLocalNodeFQDN()
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

	peers, err := c.FetchPeers()
	if err != nil {
		log.WithError(err).Warn("Problem fetching peers")
	}
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
				Ctx:          ctx,
			}
		},
	})

	// Keep in same order
	sort.Strings(peers)

	pool.Set(peers...)

	// Critical portion
	c.lock.Lock()
	defer c.lock.Unlock()

	c.pool = pool
	c.myURI = myURI
	c.myAddr = myIP
	c.myPort = myPort

	// Keep peer list updated
	go c.doPeerUpdateLoop()

	return nil
}

func (c *SharedGCache) doPeerUpdateLoop() {
	log := c.log

	time.Sleep(viper.GetDuration("groupcache.peer.update"))

	for {
		if c.peerDebug {
			log.Trace("Checking peer list")
		}
		peers, err := c.FetchPeers()
		if err != nil {
			log.WithError(err).Warn("Problem fetching peers")
		}

		if c.peerDebug {
			log.Trace("Updating peer list")
		}
		c.lock.Lock()
		c.pool.Set(peers...)
		c.lock.Unlock()
		if c.peerDebug {
			log.Trace("Updated peer list")
		}

		time.Sleep(viper.GetDuration("groupcache.peer.update"))
	}
}

func (c *SharedGCache) FetchPeers() ([]string, error) {
	log := c.log.WithFields(logrus.Fields{
		"peers.key":    viper.GetString("groupcache.peers.key"),
		"peers.my.uri": c.myURI,
	})
	res, err := c.rClient.SMembers(context.Background(), viper.GetString("groupcache.peers.key")).Result()
	if err != nil {
		log.WithError(err).Error("Problem fetching the groupcache peer list")
		return res, err
	}
	log = log.WithField("peers.pre", res)
	if c.peerDebug {
		log.Trace("Fetched the groupcache peer list from redis")
	}

	// Won't help current but will help next time
	for _, peer := range res {
		go c.checkPeer(log, peer)
	}

	return res, nil
}

//checkPeer checks if the given peer is up. If not, removes it from redis.
func (c *SharedGCache) checkPeer(log *logrus.Entry, uri string) {
	log = log.WithField("peer.uri", uri)
	if c.peerDebug {
		log.Trace("Going to check peer")
	}

	for i := 0; i < viper.GetInt("peer.failed.count"); i++ {
		log := log.WithField("check.number", i)
		ok, err := c.checkPeerStatus(log, uri)
		if err != nil {
			log = log.WithError(err)
		}
		log = log.WithField("status.ok", ok)

		if err == nil && ok {
			log.Trace("Peer is ok!")
			return
		}
		if c.peerDebug {
			log.Trace("Peer not responding this time")
		}
		time.Sleep(viper.GetDuration("peer.failed.sleep"))
	}

	if c.peerDebug {
		log.Debug("Removing peer from peer list in redis")
	}
	if err := c.removePeer(uri); err != nil {
		log.WithError(err).Info("Error removing peer from peer list in redis")
	}
	return
}

//checkPeer checks if the given peer is up. If not, removes it from redis.
func (c *SharedGCache) checkPeerStatus(log *logrus.Entry, uri string) (bool, error) {
	log = log.WithField("peer.uri", uri)
	if c.peerDebug {
		log.Trace("Going to check peer")
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/alive", uri), nil)
	if err != nil {
		log.WithError(err).Error("Problem creating request")
		return false, err
	}

	req.Header = http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", viper.GetString("groupcache.token"))},
	}

	res, err := client.Do(req)
	if err != nil {
		log.WithError(err).Info("Problem running request")
		return false, err
	}
	if res.StatusCode == 200 {
		if c.peerDebug {
			log.Trace("Done checking peer - it's ok")
		}
		return true, nil
	}
	log.WithField("status.code", res.StatusCode).Info("Problem with status code")
	return false, ErrBadStatusCode
}

//removeMyPeer removes ourselves from the redis based peer list
func (c *SharedGCache) removeMyPeer() error {
	return c.removePeer(c.myURI)
}

//removePeer removes a peer from the redis based peer list
func (c *SharedGCache) removePeer(peerURI string) error {
	log := c.log.WithFields(logrus.Fields{
		"peers.key": viper.GetString("groupcache.peers.key"),
		"peers.uri": peerURI,
	})
	if _, err := c.rClient.SRem(context.Background(), viper.GetString("groupcache.peers.key"), peerURI).Result(); err != nil {
		log.WithError(err).Error("Problem removing ourself from the groupcache peer list")
		return err
	}
	if c.peerDebug {
		log.Trace("Removed peer from groupcache peer list")
	}
	return nil
}

//addMyPeer adds ourselves to the redis based peer list
func (c *SharedGCache) addMyPeer() error {
	log := c.log.WithFields(logrus.Fields{
		"peers.key":    viper.GetString("groupcache.peers.key"),
		"peers.my.uri": c.myURI,
	})

	if _, err := c.rClient.SAdd(context.Background(), viper.GetString("groupcache.peers.key"), c.myURI).Result(); err != nil {
		log.WithError(err).Error("Problem adding ourself to groupcache peer list")
		return err
	}
	if c.peerDebug {
		log.Trace("Added ourselves to groupcache peer list")
	}
	return nil
}

//Shutdown our groupcache
func (c *SharedGCache) Shutdown() error {
	return c.removeMyPeer()
}

//StartRunPrep handles the startup prep
func (c *SharedGCache) StartRunPrep() error {
	log := c.log

	log.Info("Setting up GIN")
	ginEngine := gin.Default()

	if !viper.GetBool("debug") {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	handler_global.RegisterGlobalHandlers(ginEngine)
	// After main handlers since it
	ginEngine.Any("/_groupcache/*cache", c.GroupCacheHandler)

	// Add ourselves to the global list of groupcache URLs
	if err := c.addMyPeer(); err != nil {
		log.WithError(err).Error("Problem adding myself to peer list")
		return err
	}

	go func() {
		if err := ginEngine.Run(fmt.Sprintf("%s:%d", viper.GetString("listen"), viper.GetInt("port")+1)); err != nil {
			log.WithError(err).Fatal("Problem running GIN")
		}
	}()

	return nil
}

//GroupCacheHandler register via gin to "/_groupcache/"
func (c *SharedGCache) GroupCacheHandler(ctx *gin.Context) {
	if strings.TrimLeft(ctx.GetHeader(TokenKey), "Bearer ") != viper.GetString("groupcache.token") {
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
