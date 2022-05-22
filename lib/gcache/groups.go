package gcache

import (
	"context"
	"fmt"
	"github.com/mailgun/groupcache/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	doCheckInits()
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
		return groupcache.NewGroup(
			groupName,
			viper.GetInt64(cacheSizeKey),
			groupcache.GetterFunc(func(ctx context.Context, key string, dest groupcache.Sink) error {
				log := log.WithField("groupcache.key", key)
				log.Trace("Running group getter")
				err := groupGetterF(ctx, log, sgc, key, dest)
				if err != nil {
					log.WithError(err).Error("Problem running getter")
				} else {
					log.Trace("Ran getter successfully")
				}
				return err
			}),
		)
	})
	if err != nil {
		panic(fmt.Sprintf("Problem setting up group: %e", err))
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
