package df

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"sync"
)

type RedisPools struct {
	pools     map[string]*RedisPool
	poolsLock *sync.Mutex
	log       *logrus.Entry
}

type RedisPool struct {
	Name       string `json:"name"`
	parent     *RedisPools
	client     *redis.Client
	clientLock *sync.Mutex
}

var (
	redisPools      *RedisPools
	ErrOutOfRetries = errors.New("out of retries")
)

func init() {
	viper.SetDefault("redis.retries", 6)
	viper.SetDefault(MakeCfgKey(RPoolGroupCache, "db"), RPoolGroupCacheDB)
	viper.SetDefault(MakeCfgKey(RPoolMonitoring, "db"), RPoolMonitoringDB)
}

func GlobalInit(log *logrus.Entry) error {
	return SetGlobalRedisPool(NewRedisPools(log))
}

func NewRedisPools(log *logrus.Entry) *RedisPools {
	return &RedisPools{
		pools:     make(map[string]*RedisPool),
		poolsLock: &sync.Mutex{},
		log:       log,
	}
}

func (p *RedisPools) newRedisPool(name string) *RedisPool {
	return &RedisPool{
		Name:       name,
		parent:     p,
		clientLock: &sync.Mutex{},
	}
}

//GetGlobal fetches the global redis client pooling
func GetGlobal() *RedisPools {
	if redisPools == nil {
		panic("GetGlobal called too early")
	}
	return redisPools
}

func SetGlobalRedisPool(rp *RedisPools) error {
	if redisPools != nil {

	}
	redisPools = rp
	return nil
}

//GetCreatePool fetch the pool, create if needed
func (p *RedisPools) GetCreatePool(name string) *RedisPool {
	p.poolsLock.Lock()
	defer p.poolsLock.Unlock()

	if _, ok := p.pools[name]; !ok {
		p.pools[name] = p.newRedisPool(name)
	}

	return p.pools[name]
}

func (p *RedisPool) TestClient(client *redis.Client) bool {
	// Make sure we're connected
	if err := client.Ping(context.Background()).Err(); err != nil {
		return false
	}
	return true
}

func (p *RedisPool) GetClient(retry bool) (*redis.Client, error) {
	p.clientLock.Lock()
	defer p.clientLock.Unlock()

	if p.client == nil {
		c, err := p.MakeClient(retry)
		if err != nil {
			return nil, err
		}
		p.client = c

		// Was just tested
		return p.client, nil
	}

	if p.TestClient(p.client) {
		// Tested good
		return p.client, nil
	}

	// Test failed
	if retry {
		c, err := p.MakeClient(retry)
		if err != nil {
			return nil, err
		}
		p.client = c

		// Was just tested
		return p.client, nil
	}

	return nil, ErrOutOfRetries
}

//MakeClient creates a new client - doesn't persist it
func (p *RedisPool) MakeClient(retry bool) (*redis.Client, error) {
	log := p.parent.log
	parsedRedisURL, err := ParseRedisURL()
	if err != nil {
		log.WithError(err).Error("Problem getting parsed URL")
		return nil, err
	}
	passwd, _ := parsedRedisURL.User.Password()

	opts := redis.Options{
		Addr:       parsedRedisURL.Host,
		Password:   passwd,
		DB:         viper.GetInt(p.CfgKey("db")),
		MaxRetries: viper.GetInt("redis.retries"),
	}
	// If rediss, enable tls
	if parsedRedisURL.Scheme == "rediss" {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	rdb := redis.NewClient(&opts)

	// Make sure we're connected
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		if retry {
			return p.MakeClient(false)
		}
		return nil, err
	}

	return rdb, nil
}

func (p *RedisPool) CfgKey(last string) string {
	return MakeCfgKey(p.Name, last)
}

func MakeCfgKey(name string, last string) string {
	return fmt.Sprintf("redis.%s.%s", name, last)
}

//QuickClient is a shortcut to get a client from the given name
func QuickClient(name string, retry bool) (*redis.Client, error) {
	return GetGlobal().GetCreatePool(name).GetClient(retry)
}

//ParseRedisURL returns a parsed copy of the redis url.
func ParseRedisURL() (*url.URL, error) {
	log := Log
	rawURL := os.Getenv("REDIS_URL")
	log = log.WithField("url", rawURL)

	parsedRedisURL, err := url.Parse(rawURL)
	if err != nil {
		log.WithError(err).WithField("url", rawURL).Error("Failed to parse Redis URL")
		return nil, err
	}

	_, ok := parsedRedisURL.User.Password()
	if !ok {
		log.Warn("No redis password")
	}

	return parsedRedisURL, nil
}
