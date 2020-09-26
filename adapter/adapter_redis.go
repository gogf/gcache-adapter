// Copyright 2020 gf Author(https://github.com/gogf/gf). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package adapter

import (
	"github.com/gogf/gf/container/gvar"
	"github.com/gogf/gf/database/gredis"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/os/gcache"
	"sync"
	"time"
)

// Redis is the gcache adapter implements using Redis server.
type Redis struct {
	mu    sync.RWMutex
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
func (c *Redis) Set(key interface{}, value interface{}, duration time.Duration) {
	var err error
	if value == nil || duration < 0 {
		_, err = c.redis.Do("DEL", key)
	} else {
		if duration == 0 {
			_, err = c.redis.Do("SET", key, value)
		} else {
			_, err = c.redis.Do("SETEX", key, duration.Seconds(), value)
		}
	}
	if err != nil {
		g.Log().Error(err)
	}
}

// Update updates the value of <key> without changing its expiration and returns the old value.
// The returned value <exist> is false if the <key> does not exist in the cache.
//
// It deletes the <key> if given <value> is nil.
// It does nothing if <key> does not exist in the cache.
func (c *Redis) Update(key interface{}, value interface{}) (oldValue interface{}, exist bool) {
	var (
		err         error
		v           *gvar.Var
		oldDuration time.Duration
	)
	// TTL.
	v, err = c.redis.DoVar("TTL", key)
	if err != nil {
		g.Log().Error(err)
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
		g.Log().Error(err)
		return
	}
	oldValue = v.Val()
	// DEL.
	if value == nil {
		_, err = c.redis.Do("DEL", key)
		if err != nil {
			g.Log().Error(err)
			return
		}
		return
	}
	// Update the value.
	if oldDuration == -1 {
		_, err = c.redis.Do("SET", key, value)
		if err != nil {
			g.Log().Error(err)
		}
	} else {
		oldDuration *= time.Second
		_, err = c.redis.Do("SETEX", key, oldDuration.Seconds(), value)
		if err != nil {
			g.Log().Error(err)
		}
	}
	return oldValue, true
}

// UpdateExpire updates the expiration of <key> and returns the old expiration duration value.
//
// It returns -1 if the <key> does not exist in the cache.
// It deletes the <key> if <duration> < 0.
func (c *Redis) UpdateExpire(key interface{}, duration time.Duration) (oldDuration time.Duration) {
	var (
		err error
		v   *gvar.Var
	)
	// TTL.
	v, err = c.redis.DoVar("TTL", key)
	if err != nil {
		g.Log().Error(err)
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
		if err != nil {
			g.Log().Error(err)
		}
	}
	// No expire.
	if duration == 0 {
		v, err = c.redis.DoVar("GET", key)
		if err != nil {
			g.Log().Error(err)
			return
		}
		_, err = c.redis.Do("SET", key, v.Val())
		if err != nil {
			g.Log().Error(err)
		}
	}
	return
}

// GetExpire retrieves and returns the expiration of <key> in the cache.
//
// It returns 0 if the <key> does not expire.
// It returns -1 if the <key> does not exist in the cache.
func (c *Redis) GetExpire(key interface{}) time.Duration {
	v, err := c.redis.DoVar("TTL", key)
	if err != nil {
		g.Log().Error(err)
	}
	switch v.Int() {
	case -1:
		return 0
	case -2:
		return -1
	default:
		return v.Duration() * time.Second
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
func (c *Redis) SetIfNotExist(key interface{}, value interface{}, duration time.Duration) bool {
	// Execute the function and retrieve the result.
	if f, ok := value.(func() interface{}); ok {
		value = f()
		if value == nil {
			return false
		}
	}
	// DEL.
	if duration < 0 || value == nil {
		v, err := c.redis.DoVar("DEL", key, value)
		if err != nil {
			g.Log().Error(err)
			return false
		}
		if v.Int() == 1 {
			return true
		} else {
			return false
		}
	}
	v, err := c.redis.DoVar("SETNX", key, value)
	if err != nil {
		g.Log().Error(err)
		return false
	}
	if v.Int() > 0 {
		// Set the expire.
		_, err := c.redis.Do("TTL", key, duration.Seconds())
		if err != nil {
			g.Log().Error(err)
			return false
		}
		return true
	}
	return false
}

// Sets batch sets cache with key-value pairs by <data>, which is expired after <duration>.
//
// It does not expire if <duration> == 0.
// It deletes the keys of <data> if <duration> < 0 or given <value> is nil.
func (c *Redis) Sets(data map[interface{}]interface{}, duration time.Duration) {
	if len(data) == 0 {
		return
	}
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
			g.Log().Error(err)
			return
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
			g.Log().Error(err)
			return
		}
	}
	if duration > 0 {
		for k, v := range data {
			c.Set(k, v, duration)
		}
	}
}

// Get retrieves and returns the associated value of given <key>.
// It returns nil if it does not exist or its value is nil.
func (c *Redis) Get(key interface{}) interface{} {
	v, err := c.redis.DoVar("GET", key)
	if err != nil {
		g.Log().Error(err)
		return nil
	}
	return v.Val()
}

// GetOrSet retrieves and returns the value of <key>, or sets <key>-<value> pair and
// returns <value> if <key> does not exist in the cache. The key-value pair expires
// after <duration>.
//
// It does not expire if <duration> == 0.
// It deletes the <key> if <duration> < 0 or given <value> is nil, but it does nothing
// if <value> is a function and the function result is nil.
func (c *Redis) GetOrSet(key interface{}, value interface{}, duration time.Duration) interface{} {
	if v := c.Get(key); v == nil {
		c.Set(key, value, duration)
		return value
	} else {
		return v
	}
}

// GetOrSetFunc retrieves and returns the value of <key>, or sets <key> with result of
// function <f> and returns its result if <key> does not exist in the cache. The key-value
// pair expires after <duration>.
//
// It does not expire if <duration> == 0.
// It deletes the <key> if <duration> < 0 or given <value> is nil, but it does nothing
// if <value> is a function and the function result is nil.
func (c *Redis) GetOrSetFunc(key interface{}, f func() interface{}, duration time.Duration) interface{} {
	if v := c.Get(key); v == nil {
		value := f()
		if value == nil {
			return nil
		}
		c.Set(key, value, duration)
		return value
	} else {
		return v
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
func (c *Redis) GetOrSetFuncLock(key interface{}, f func() interface{}, duration time.Duration) interface{} {
	return c.GetOrSetFunc(key, f, duration)
}

// Remove deletes the one or more keys from cache, and returns its value.
// If multiple keys are given, it returns the value of the deleted last item.
func (c *Redis) Remove(keys ...interface{}) (value interface{}) {
	if len(keys) == 0 {
		return nil
	}
	// Retrieves the last key value.
	v, err := c.redis.DoVar("GET", keys[len(keys)-1])
	if err != nil {
		g.Log().Error(err)
		return
	}
	value = v.Val()
	// Deletes all given keys.
	_, err = c.redis.Do("DEL", keys...)
	if err != nil {
		g.Log().Error(err)
		return
	}
	return
}

// Data returns a copy of all key-value pairs in the cache as map type.
func (c *Redis) Data() map[interface{}]interface{} {
	// Keys.
	v, err := c.redis.DoVar("KEYS", "*")
	if err != nil {
		g.Log().Error(err)
		return nil
	}
	keys := v.Slice()
	// Values.
	v, err = c.redis.DoVar("MGET", keys...)
	if err != nil {
		g.Log().Error(err)
		return nil
	}
	values := v.Slice()
	// Compose keys and values.
	data := make(map[interface{}]interface{})
	for i := 0; i < len(keys); i++ {
		data[keys[i]] = values[i]
	}
	return data
}

// Keys returns all keys in the cache as slice.
func (c *Redis) Keys() []interface{} {
	v, err := c.redis.DoVar("KEYS", "*")
	if err != nil {
		g.Log().Error(err)
		return nil
	}
	return v.Slice()
}

// Values returns all values in the cache as slice.
func (c *Redis) Values() []interface{} {
	// Keys.
	v, err := c.redis.DoVar("KEYS", "*")
	if err != nil {
		g.Log().Error(err)
		return nil
	}
	keys := v.Slice()
	// Values.
	v, err = c.redis.DoVar("MGET", keys...)
	if err != nil {
		g.Log().Error(err)
		return nil
	}
	return v.Slice()
}

// Size returns the size of the cache.
func (c *Redis) Size() (size int) {
	return len(c.Keys())
}

// Clear clears all data of the cache.
// Note that this function is sensitive and should be carefully used.
func (c *Redis) Clear() error {
	// The "FLUSHDB" may not be available.
	if _, err := c.redis.Do("FLUSHDB"); err != nil {
		c.Remove(c.Keys()...)
	}
	return nil
}

// Close closes the cache.
func (c *Redis) Close() error {
	// It does nothing.
	return nil
}
