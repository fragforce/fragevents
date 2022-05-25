package gcache

import (
	"context"
	"fmt"
	"github.com/mailgun/groupcache/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

func init() {
	doCheckInits()
	viper.SetDefault("cache.stat.initial", time.Minute*10) // How long to wait before posting the first time
	viper.SetDefault("cache.stat.sleep", time.Minute*5)    // How long in between posts after the first
}

func RegisterGroup(g *groupcache.Group) {
	gLock.Lock()
	defer gLock.Unlock()
	groups[g.Name()] = g
}

func RegisterPendingGroup(f GroupFunc) error {
	if pendingDone {
		return ErrPendingGroupsCreated
	}

	pLock.Lock()
	defer pLock.Unlock()

	pendingGroupsF = append(pendingGroupsF, f)
	return nil
}

//initPendingGroupF runs the pending group init if not already run
func (c *SharedGCache) initPendingGroupF() {
	log := c.log

	if pendingDone {
		return
	}
	pLock.Lock()
	defer pLock.Unlock()

	for _, gF := range pendingGroupsF {
		g := gF(log, c)
		log := log.WithField("group.name", g.Name())
		RegisterGroup(g)
		log.Trace("Registered group")
	}
	pendingDone = true
}

//registerGroupF called from init to create+register groupcache create functions
func registerGroupF(groupName string, defaultCacheSizeMB int64, groupGetterF GroupGetterFunc) {
	if defaultCacheSizeMB <= 0 {
		defaultCacheSizeMB = 16
	}
	cacheSizeKey := fmt.Sprintf("group.%s.bytes", groupName)
	viper.SetDefault(cacheSizeKey, 1024*1024*defaultCacheSizeMB)

	err := RegisterPendingGroup(func(log *logrus.Entry, sgc *SharedGCache) *groupcache.Group {
		log = log.WithField("cache.size.bytes", viper.GetInt64(cacheSizeKey))
		log.Debug("Creating new group")
		ret := groupcache.NewGroup(
			groupName,
			viper.GetInt64(cacheSizeKey),
			groupcache.GetterFunc(func(ctx context.Context, key string, dest groupcache.Sink) error {
				log := log.WithField("groupcache.key", key)
				log.Trace("Running group getter")
				res, err := groupGetterF(ctx, log, sgc, key)
				if err != nil {
					log.WithError(err).Error("Problem running getter")
					return err
				}
				grp := groupcache.GetGroup(groupName)
				t := time.Now().Add(time.Minute * 15)
				//if err := grp.Set(ctx, key, res, t, true); err != nil {
				//	log.WithError(err).Error("Problem updating cache")
				//	return err
				//}
				if err := grp.Set(ctx, key, res, t, false); err != nil {
					log.WithError(err).Error("Problem updating cache")
					return err
				}
				if err := dest.SetBytes(res, t); err != nil {
					log.WithError(err).Error("Problem returning data")
					return err
				}

				log.Trace("Ran getter successfully")
				return nil
			}),
		)
		// Log the group's stats every once in a while
		go sgc.logCacheStats(log, ret)
		return ret
	})
	if err != nil {
		panic(fmt.Sprintf("Problem setting up group: %e", err))
	}
}

//logCacheStats gets run via go routine to run forever and log the groupcache.Group stats every x period
func (c *SharedGCache) logCacheStats(log *logrus.Entry, group *groupcache.Group) {
	sleepPeriod := viper.GetDuration("cache.stat.sleep")
	log = log.WithField("sleep.period", sleepPeriod)
	time.Sleep(viper.GetDuration("cache.stat.initial"))
	for {
		log := log.WithFields(logrus.Fields{
			"group.stats.raw":  group.Stats,
			"group.name":       group.Name(),
			"group.stats.main": group.CacheStats(groupcache.MainCache),
			"group.stats.hot":  group.CacheStats(groupcache.HotCache),
		})

		if peers, err := c.FetchPeers(); err != nil {
			log = log.WithError(err) // Just add it in :shrug:
		} else {
			log = log.WithFields(logrus.Fields{
				"group.peers": peers,
			})
		}

		log.Info("Cache stats")
		time.Sleep(sleepPeriod)
	}
}

func (c *SharedGCache) GetGroupByName(groupName string) (*groupcache.Group, error) {
	if !pendingDone {
		return nil, ErrPendingGroupsNotCreated
	}

	gLock.Lock()
	defer gLock.Unlock()

	grp, ok := groups[groupName]
	if !ok {
		return nil, ErrNoSuchGroup
	}

	return grp, nil
}

//GetAllGroups returns a list of all groupcache groups
func (c *SharedGCache) GetAllGroups() ([]*groupcache.Group, error) {
	if !pendingDone {
		return nil, ErrPendingGroupsNotCreated
	}

	gLock.Lock()
	defer gLock.Unlock()

	ret := make([]*groupcache.Group, len(groups))
	idx := 0
	for _, v := range groups {
		ret[idx] = v
		idx++
	}
	return ret, nil
}
