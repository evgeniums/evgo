package inmem_cache

import (
	"time"

	"github.com/evgeniums/evgo/pkg/cache"
	"github.com/jellydator/ttlcache/v3"
)

type InmemLocker struct {
	cache *ttlcache.Cache[string, struct{}]
}

type InmemLock struct {
	locker      *InmemLocker
	key         string
	notObtained bool
}

func (l *InmemLock) Release() error {
	l.locker.cache.Delete(l.key)
	return nil
}

func (l *InmemLock) NotObtained() bool {
	return l.notObtained
}

func NewLocker() *InmemLocker {
	l := &InmemLocker{cache: ttlcache.New(ttlcache.WithDisableTouchOnHit[string, struct{}]())}
	return l
}

func (l *InmemLocker) Lock(key string, ttlSeconds time.Duration) (cache.Lock, error) {

	lock := &InmemLock{locker: l, key: key}

	ttl := ttlcache.NoTTL
	if ttlSeconds != 0 {
		ttl = time.Second * time.Duration(ttlSeconds)
	}

	_, lock.notObtained = l.cache.GetOrSet(key, struct{}{}, ttlcache.WithTTL[string, struct{}](ttl))

	return lock, nil
}
