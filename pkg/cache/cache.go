package cache

import (
	"github.com/evgeniums/evgo/pkg/message"
	"github.com/evgeniums/evgo/pkg/message/message_json"
	"github.com/evgeniums/evgo/pkg/utils"
)

type Cache interface {
	Set(key string, value interface{}, ttlSeconds ...int) error
	Get(key string, value interface{}) (bool, error)
	Unset(key string) error
	GetUnset(key string, value interface{}) (bool, error)
	Touch(key string) error
	Keys() ([]string, error)
	Clear() error
}

type GenericCache[T any] interface {
	Set(key string, value T, ttlSeconds ...int) error
	Get(key string, value *T) (bool, error)
	Unset(key string) error
	GetUnset(key string, value *T) (bool, error)
	Touch(key string) error
	Keys() ([]string, error)
	Clear() error
}

type StringCache = GenericCache[string]

type SerializedObjectCache struct {
	impl         StringCache
	Serializer   message.Serializer
	StringCoding utils.StringCoding
}

func New(backend StringCache, serializer ...message.Serializer) *SerializedObjectCache {
	c := &SerializedObjectCache{}
	c.impl = backend
	c.Serializer = utils.OptionalArg[message.Serializer](&message_json.JsonSerializer{}, serializer...)
	c.StringCoding = &utils.Base64StringCoding{}
	return c
}

func (c *SerializedObjectCache) Set(key string, value interface{}, ttlSeconds ...int) error {

	b, err := c.Serializer.SerializeMessage(value)
	if err != nil {
		return err
	}
	str := c.StringCoding.Encode(b)

	return c.impl.Set(key, str, ttlSeconds...)
}

func (c *SerializedObjectCache) Get(key string, obj interface{}) (bool, error) {

	var val string

	found, err := c.impl.Get(key, &val)
	if !found || err != nil {
		return false, err
	}

	b, err := c.StringCoding.Decode(val)
	if err != nil {
		return true, err
	}

	err = c.Serializer.ParseMessage(b, obj)
	if err != nil {
		return true, err
	}

	return true, nil
}

func (c *SerializedObjectCache) GetUnset(key string, obj interface{}) (bool, error) {

	var val string

	found, err := c.impl.GetUnset(key, &val)
	if !found || err != nil {
		return false, err
	}

	b, err := c.StringCoding.Decode(val)
	if err != nil {
		return true, err
	}

	err = c.Serializer.ParseMessage(b, obj)
	if err != nil {
		return true, err
	}

	return true, nil
}

func (c *SerializedObjectCache) Unset(key string) error {
	return c.impl.Unset(key)
}

func (c *SerializedObjectCache) Clear() error {
	return c.impl.Clear()
}

func (c *SerializedObjectCache) Touch(key string) error {
	return c.impl.Touch(key)
}

func (c *SerializedObjectCache) Keys() ([]string, error) {
	return c.impl.Keys()
}
