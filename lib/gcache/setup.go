package gcache

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/mailgun/groupcache/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"sync"
)

type SharedGCache struct {
	lock      *sync.Mutex
	baseDir   string
	log       *logrus.Entry
	pool      *groupcache.HTTPPool
	myURI     string
	myAddr    string
	myPort    int
	rClient   *redis.Client
	peerDebug bool
}

type GroupFunc func(log *logrus.Entry, sgc *SharedGCache) *groupcache.Group
type GroupGetterFunc func(ctx context.Context, log *logrus.Entry, sgc *SharedGCache, key string, dest groupcache.Sink) error

var (
	cache                      *SharedGCache
	cLock                      *sync.Mutex
	groups                     map[string]*groupcache.Group
	gLock                      *sync.Mutex
	pendingGroupsF             []GroupFunc
	pLock                      *sync.Mutex
	pendingDone                bool
	ErrPendingGroupsCreated    = errors.New("pending groups already created")
	ErrPendingGroupsNotCreated = errors.New("pending groups not created yet")
	ErrNoSuchGroup             = errors.New("requested group doesn't exist")
)

func init() {
	doCheckInits()
	viper.SetDefault("groupcache.basedir", "/tmp/groupcache/")
}

//doCheckInits runs various local inits that need to run before others can do stuff - Safe to rerun many times
func doCheckInits() {
	if cLock == nil {
		cLock = &sync.Mutex{}
	}
	if gLock == nil {
		gLock = &sync.Mutex{}
	}
	if pLock == nil {
		pLock = &sync.Mutex{}
	}
	// Lock in case we're being run later somehow...
	// Might as well :shrug:
	if groups == nil {
		gLock.Lock()
		groups = make(map[string]*groupcache.Group)
		gLock.Unlock()
	}
	if pendingGroupsF == nil {
		pLock.Lock()
		pendingGroupsF = make([]GroupFunc, 0)
		pLock.Unlock()
	}
}

func NewSharedGCache(log *logrus.Entry, baseDir string, rClient *redis.Client) (*SharedGCache, error) {
	log = log.WithField("cache.basedir", baseDir)

	ret := SharedGCache{
		lock:      &sync.Mutex{},
		baseDir:   baseDir,
		log:       log,
		rClient:   rClient,
		peerDebug: viper.GetBool("debug.peers") && viper.GetBool("debug"),
	}

	// Init gcache pool
	if err := ret.createPool(); err != nil {
		return nil, err
	}

	// Do more stuff

	ret.log = log // In case of log updates before return
	return &ret, nil
}

func NewGlobalSharedGCache(log *logrus.Entry, baseDir string, rClient *redis.Client) (*SharedGCache, error) {
	c, err := NewSharedGCache(log, baseDir, rClient)
	if err != nil {
		return nil, err
	}

	c.lock = cLock

	cLock.Lock()
	cache = c
	cLock.Unlock()

	cache.initPendingGroupF()

	return cache, nil
}

//GlobalCache fetches the main, global, shared gcache
func GlobalCache() *SharedGCache {
	return cache
}
