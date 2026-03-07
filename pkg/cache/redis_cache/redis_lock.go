package redis_cache

import (
	"time"

	"github.com/bsm/redislock"
	"github.com/evgeniums/evgo/pkg/cache"
)

type RedisLocker struct {
	*RedisCache
	locker *redislock.Client
}

type RedisLock struct {
	locker      *RedisLocker
	lock        *redislock.Lock
	notObtained bool
}

func (r *RedisLock) Release() error {
	err := r.lock.Release(r.locker.Context())
	if err != nil {
		if err == redislock.ErrLockNotHeld {
			return nil
		}
		return err
	}
	return nil
}

func (r *RedisLock) NotObtained() bool {
	return r.notObtained
}

func NewLocker(redisCache *RedisCache) *RedisLocker {
	l := &RedisLocker{RedisCache: redisCache}
	l.locker = redislock.New(redisCache.NativeHandler())
	return l
}

func (r *RedisLocker) Lock(key string, ttl time.Duration) (cache.Lock, error) {

	var err error

	lock := &RedisLock{locker: r}
	lock.lock, err = r.locker.Obtain(r.Context(), key, ttl, nil)
	if err != nil {

		if err == redislock.ErrNotObtained {
			lock.lock = nil
			lock.notObtained = true
			return lock, nil
		}

		return nil, err
	}

	return lock, nil
}

/**

HOW TO USE.

1. Add variable to struct.

type SomeType struct {

	...

	locker cache.Locker

	...
}

2. Init locker in Init.const

func (s *SomeType) Init(...) error {

...

	// init locker
	redisCache := redis_cache.NewCache()
	err = redisCache.Init(app.Cfg(), app.Logger(), app.Validator(), redis_cache.RedisCacheConfigPath)
	if err != nil {
		...
		return ...
	}
	s.locker = redis_cache.NewLocker(redisCache)

...

}

3. Use locker to lock access to some object when required.

func FuncWithLock(...) (...) {

...

lock, err := cache.LockObject(s.locker, keyPrefix, objectId, s.LOCK_TTL)
if err!= nil {
	c.SetLoggerField("locked_object_id",objectId)
	c.SetMessage("failed to lock object")
	return ...
}
if lock==nil {

	// object already locked - skip operation or retry later
	c.SetLoggerField("locked_object_id",objectId)
	c.Logger().Warn("object already locked")
	...

	return ...
}
defer func() {
	e:=lock.Release()
	if e!=nil {
		c.SetLoggerField("locked_object_id",objectId)
		c.Logger().Error("failed to release lock",e)
	}
} ()


...

}

**/
