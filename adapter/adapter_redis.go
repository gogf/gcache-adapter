// Copyright 2020 gf Author(https://github.com/gogf/gf). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package adapter

import (
	"github.com/gogf/gf/container/gvar"
	"github.com/gogf/gf/database/gredis"
	"github.com/gogf/gf/os/gcache"
	"time"
)

// Redis is the gcache adapter implements using Redis server.
type Redis struct {
	redis *gredis.Redis
}

// newAdapterMemory creates and returns a new memory cache object.
func NewRedis(redis *gredis.Redis) gcache.Adapter {
	return &Redis{
		redis: redis,
	}
}

// Set sets cache with <key>-<value> pair, which is expired after <duration>.
// It does not expire if <duration> == 0.
// It deletes the <key> if <duration> < 0 or given <value> is nil.
func (c *Redis) Set(key interface{}, value interface{}, duration time.Duration) error {
	var err error
	if value == nil || duration < 0 {
		_, err = c.redis.DoVar("DEL", key)
	} else {
		if duration == 0 {
			_, err = c.redis.DoVar("SET", key, value)
		} else {
			_, err = c.redis.DoVar("SETEX", key, duration.Seconds(), value)
		}
	}
	return err
}

// Update updates the value of <key> without changing its expiration and returns the old value.
// The returned value <exist> is false if the <key> does not exist in the cache.
//
// It deletes the <key> if given <value> is nil.
// It does nothing if <key> does not exist in the cache.
func (c *Redis) Update(key interface{}, value interface{}) (oldValue interface{}, exist bool, err error) {
	var (
		v           *gvar.Var
		oldDuration time.Duration
	)
	// TTL.
	v, err = c.redis.DoVar("TTL", key)
	if err != nil {
		return
	}
	oldDuration = v.Duration()
	if oldDuration == -2 {
		// It does not exist.
		return
	}
	// Check existence.
	v, err = c.redis.DoVar("GET", key)
	if err != nil {
		return
	}
	oldValue = v.Val()
	// DEL.
	if value == nil {
		_, err = c.redis.DoVar("DEL", key)
		if err != nil {
			return
		}
		return
	}
	// Update the value.
	if oldDuration == -1 {
		_, err = c.redis.DoVar("SET", key, value)
	} else {
		oldDuration *= time.Second
		_, err = c.redis.DoVar("SETEX", key, oldDuration.Seconds(), value)
	}
	return oldValue, true, err
}

// UpdateExpire updates the expiration of <key> and returns the old expiration duration value.
//
// It returns -1 if the <key> does not exist in the cache.
// It deletes the <key> if <duration> < 0.
func (c *Redis) UpdateExpire(key interface{}, duration time.Duration) (oldDuration time.Duration, err error) {
	var (
		v *gvar.Var
	)
	// TTL.
	v, err = c.redis.DoVar("TTL", key)
	if err != nil {
		return
	}
	oldDuration = v.Duration()
	if oldDuration == -2 {
		// It does not exist.
		oldDuration = -1
		return
	}
	oldDuration *= time.Second
	// DEL.
	if duration < 0 {
		_, err = c.redis.Do("DEL", key)
		return
	}
	// Update the expire.
	if duration > 0 {
		_, err = c.redis.Do("EXPIRE", key, duration.Seconds())
	}
	// No expire.
	if duration == 0 {
		v, err = c.redis.DoVar("GET", key)
		if err != nil {
			return
		}
		_, err = c.redis.Do("SET", key, v.Val())
	}
	return
}

// GetExpire retrieves and returns the expiration of <key> in the cache.
//
// It returns 0 if the <key> does not expire.
// It returns -1 if the <key> does not exist in the cache.
func (c *Redis) GetExpire(key interface{}) (time.Duration, error) {
	v, err := c.redis.DoVar("TTL", key)
	if err != nil {
		return 0, err
	}
	switch v.Int() {
	case -1:
		return 0, nil
	case -2:
		return -1, nil
	default:
		return v.Duration() * time.Second, nil
	}
}

// SetIfNotExist sets cache with <key>-<value> pair which is expired after <duration>
// if <key> does not exist in the cache. It returns true the <key> dose not exist in the
// cache and it sets <value> successfully to the cache, or else it returns false.
//
// The parameter <value> can be type of <func() interface{}>, but it dose nothing if its
// result is nil.
//
// It does not expire if <duration> == 0.
// It deletes the <key> if <duration> < 0 or given <value> is nil.
func (c *Redis) SetIfNotExist(key interface{}, value interface{}, duration time.Duration) (bool, error) {
	var err error
	// Execute the function and retrieve the result.
	if f, ok := value.(func() (interface{}, error)); ok {
		value, err = f()
		if value == nil {
			return false, err
		}
	}
	// DEL.
	if duration < 0 || value == nil {
		v, err := c.redis.DoVar("DEL", key, value)
		if err != nil {
			return false, err
		}
		if v.Int() == 1 {
			return true, err
		} else {
			return false, err
		}
	}
	v, err := c.redis.DoVar("SETNX", key, value)
	if err != nil {
		return false, err
	}
	if v.Int() > 0 {
		// Set the expire.
		_, err := c.redis.Do("EXPIRE", key, duration.Seconds())
		if err != nil {
			return false, err
		}
		return true, err
	}
	return false, err
}

// Sets batch sets cache with key-value pairs by <data>, which is expired after <duration>.
//
// It does not expire if <duration> == 0.
// It deletes the keys of <data> if <duration> < 0 or given <value> is nil.
func (c *Redis) Sets(data map[interface{}]interface{}, duration time.Duration) error {
	if len(data) == 0 {
		return nil
	}
	// DEL.
	if duration < 0 {
		var (
			index = 0
			keys  = make([]interface{}, len(data))
		)
		for k, _ := range data {
			keys[index] = k
			index += 1
		}
		_, err := c.redis.Do("DEL", keys...)
		if err != nil {
			return err
		}
	}
	if duration == 0 {
		var (
			index     = 0
			keyValues = make([]interface{}, len(data)*2)
		)
		for k, v := range data {
			keyValues[index] = k
			keyValues[index+1] = v
			index += 2
		}
		_, err := c.redis.Do("MSET", keyValues...)
		if err != nil {
			return err
		}
	}
	if duration > 0 {
		var err error
		for k, v := range data {
			if err = c.Set(k, v, duration); err != nil {
				return err
			}
		}
	}
	return nil
}

// Get retrieves and returns the associated value of given <key>.
// It returns nil if it does not exist or its value is nil.
func (c *Redis) Get(key interface{}) (interface{}, error) {
	v, err := c.redis.DoVar("GET", key)
	if err != nil {
		return nil, err
	}
	return v.Val(), nil
}

// GetOrSet retrieves and returns the value of <key>, or sets <key>-<value> pair and
// returns <value> if <key> does not exist in the cache. The key-value pair expires
// after <duration>.
//
// It does not expire if <duration> == 0.
// It deletes the <key> if <duration> < 0 or given <value> is nil, but it does nothing
// if <value> is a function and the function result is nil.
func (c *Redis) GetOrSet(key interface{}, value interface{}, duration time.Duration) (interface{}, error) {
	v, err := c.Get(key)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return value, c.Set(key, value, duration)
	} else {
		return v, nil
	}
}

// GetOrSetFunc retrieves and returns the value of <key>, or sets <key> with result of
// function <f> and returns its result if <key> does not exist in the cache. The key-value
// pair expires after <duration>.
//
// It does not expire if <duration> == 0.
// It deletes the <key> if <duration> < 0 or given <value> is nil, but it does nothing
// if <value> is a function and the function result is nil.
func (c *Redis) GetOrSetFunc(key interface{}, f func() (interface{}, error), duration time.Duration) (interface{}, error) {
	v, err := c.Get(key)
	if err != nil {
		return nil, err
	}
	if v == nil {
		value, err := f()
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, nil
		}
		return value, c.Set(key, value, duration)
	} else {
		return v, nil
	}
}

// GetOrSetFuncLock retrieves and returns the value of <key>, or sets <key> with result of
// function <f> and returns its result if <key> does not exist in the cache. The key-value
// pair expires after <duration>.
//
// It does not expire if <duration> == 0.
// It does nothing if function <f> returns nil.
//
// Note that the function <f> should be executed within writing mutex lock for concurrent
// safety purpose.
func (c *Redis) GetOrSetFuncLock(key interface{}, f func() (interface{}, error), duration time.Duration) (interface{}, error) {
	return c.GetOrSetFunc(key, f, duration)
}

// Contains returns true if <key> exists in the cache, or else returns false.
func (c *Redis) Contains(key interface{}) (bool, error) {
	v, err := c.redis.DoVar("EXISTS", key)
	if err != nil {
		return false, err
	}
	return v.Bool(), nil
}

// Remove deletes the one or more keys from cache, and returns its value.
// If multiple keys are given, it returns the value of the deleted last item.
func (c *Redis) Remove(keys ...interface{}) (value interface{}, err error) {
	if len(keys) == 0 {
		return nil, nil
	}
	// Retrieves the last key value.
	if v, err := c.redis.DoVar("GET", keys[len(keys)-1]); err != nil {
		return nil, err
	} else {
		value = v.Val()
	}
	// Deletes all given keys.
	_, err = c.redis.DoVar("DEL", keys...)
	return value, err
}

// Data returns a copy of all key-value pairs in the cache as map type.
func (c *Redis) Data() (map[interface{}]interface{}, error) {
	// Keys.
	v, err := c.redis.DoVar("KEYS", "*")
	if err != nil {
		return nil, err
	}
	keys := v.Slice()
	// Values.
	v, err = c.redis.DoVar("MGET", keys...)
	if err != nil {
		return nil, err
	}
	values := v.Slice()
	// Compose keys and values.
	data := make(map[interface{}]interface{})
	for i := 0; i < len(keys); i++ {
		data[keys[i]] = values[i]
	}
	return data, nil
}

// Keys returns all keys in the cache as slice.
func (c *Redis) Keys() ([]interface{}, error) {
	v, err := c.redis.DoVar("KEYS", "*")
	if err != nil {
		return nil, err
	}
	return v.Slice(), nil
}

// Values returns all values in the cache as slice.
func (c *Redis) Values() ([]interface{}, error) {
	// Keys.
	v, err := c.redis.DoVar("KEYS", "*")
	if err != nil {
		return nil, err
	}
	keys := v.Slice()
	// Values.
	v, err = c.redis.DoVar("MGET", keys...)
	if err != nil {
		return nil, err
	}
	return v.Slice(), nil
}

// Size returns the size of the cache.
func (c *Redis) Size() (size int, err error) {
	v, err := c.redis.DoVar("DBSIZE")
	if err != nil {
		return 0, err
	}
	return v.Int(), nil
}

// Clear clears all data of the cache.
// Note that this function is sensitive and should be carefully used.
func (c *Redis) Clear() error {
	// The "FLUSHDB" may not be available.
	if _, err := c.redis.DoVar("FLUSHDB"); err != nil {
		keys, err := c.Keys()
		if err != nil {
			return err
		}
		_, err = c.Remove(keys...)
		return err
	}
	return nil
}

// Close closes the cache.
func (c *Redis) Close() error {
	// It does nothing.
	return nil
}
