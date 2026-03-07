package cache

import (
	"time"

	"github.com/evgeniums/evgo/pkg/utils"
)

type Lock interface {
	NotObtained() bool
	Release() error
}

type Locker interface {
	Lock(key string, ttl time.Duration) (Lock, error)
}

func LockObject(locker Locker, keyPrefix string, objectId string, ttl int) (Lock, error) {

	key := utils.ConcatStrings(keyPrefix, "_", objectId)
	lock, err := locker.Lock(key, time.Second*time.Duration(ttl))
	if err != nil {
		return nil, err
	}
	if lock.NotObtained() {
		return nil, nil
	}

	return lock, nil
}
