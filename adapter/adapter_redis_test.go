// Copyright 2020 gf Author(https://github.com/gogf/gf). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package adapter_test

import (
	"github.com/gogf/gcache-adapter/adapter"
	"github.com/gogf/gf/database/gredis"
	"github.com/gogf/gf/os/gcache"
	"github.com/gogf/gf/test/gtest"
	"testing"
	"time"
)

var (
	cacheRedis  = gcache.New()
	redisConfig = gredis.Config{
		Host: "127.0.0.1",
		Port: 6379,
		Db:   1,
	}
)

func init() {
	cacheRedis.SetAdapter(adapter.NewRedis(gredis.New(redisConfig)))
}

func Test_Basic1(t *testing.T) {
	size := 10
	gtest.C(t, func(t *gtest.T) {
		for i := 0; i < size; i++ {
			cacheRedis.Set(i, i*10, 0)
		}
		for i := 0; i < size; i++ {
			v, _ := cacheRedis.Get(i)
			t.Assert(v, i*10)
		}
		n, _ := cacheRedis.Size()
		t.Assert(n, size)
	})
	gtest.C(t, func(t *gtest.T) {
		data, _ := cacheRedis.Data()
		t.Assert(len(data), size)
		t.Assert(data["0"], "0")
		t.Assert(data["1"], "10")
		t.Assert(data["9"], "90")
	})
	gtest.C(t, func(t *gtest.T) {
		cacheRedis.Clear()
		n, _ := cacheRedis.Size()
		t.Assert(n, 0)
	})
}

func Test_Basic2(t *testing.T) {
	defer cacheRedis.Clear()
	size := 10
	gtest.C(t, func(t *gtest.T) {
		for i := 0; i < size; i++ {
			cacheRedis.Set(i, i*10, -1)
		}
		for i := 0; i < size; i++ {
			v, _ := cacheRedis.Get(i)
			t.Assert(v, nil)
		}
		n, _ := cacheRedis.Size()
		t.Assert(n, 0)
	})
}

func Test_Basic3(t *testing.T) {
	defer cacheRedis.Clear()
	size := 10
	gtest.C(t, func(t *gtest.T) {
		for i := 0; i < size; i++ {
			cacheRedis.Set(i, i*10, time.Second)
		}
		for i := 0; i < size; i++ {
			v, _ := cacheRedis.Get(i)
			t.Assert(v, i*10)
		}
		n, _ := cacheRedis.Size()
		t.Assert(n, size)
	})
	time.Sleep(time.Second * 2)
	gtest.C(t, func(t *gtest.T) {
		for i := 0; i < size; i++ {
			v, _ := cacheRedis.Get(i)
			t.Assert(v, nil)
		}
		n, _ := cacheRedis.Size()
		t.Assert(n, 0)
	})
}

func TestRedis_Update(t *testing.T) {
	defer cacheRedis.Clear()
	gtest.C(t, func(t *gtest.T) {
		var (
			key    = "key"
			value1 = "value1"
			value2 = "value2"
		)
		cacheRedis.Set(key, value1, time.Second)
		v, _ := cacheRedis.Get(key)
		t.Assert(v, value1)

		d, _ := cacheRedis.GetExpire(key)
		t.Assert(d > time.Millisecond*500, true)
		t.Assert(d <= time.Second, true)

		cacheRedis.Update(key, value2)

		v, _ = cacheRedis.Get(key)
		t.Assert(v, value2)
		d, _ = cacheRedis.GetExpire(key)
		t.Assert(d > time.Millisecond*500, true)
		t.Assert(d <= time.Second, true)
	})
}

func TestRedis_UpdateExpire(t *testing.T) {
	defer cacheRedis.Clear()
	gtest.C(t, func(t *gtest.T) {
		var (
			key   = "key"
			value = "value"
		)
		cacheRedis.Set(key, value, time.Second)
		v, _ := cacheRedis.Get(key)
		t.Assert(v, value)

		d, _ := cacheRedis.GetExpire(key)
		t.Assert(d > time.Millisecond*500, true)
		t.Assert(d <= time.Second, true)

		cacheRedis.UpdateExpire(key, time.Second*2)

		d, _ = cacheRedis.GetExpire(key)
		t.Assert(d > time.Second, true)
		t.Assert(d <= 2*time.Second, true)
	})
}

func TestRedis_SetIfNotExist(t *testing.T) {
	defer cacheRedis.Clear()
	gtest.C(t, func(t *gtest.T) {
		var (
			key    = "key"
			value1 = "value1"
			value2 = "value2"
		)
		cacheRedis.Set(key, value1, time.Second)
		v, _ := cacheRedis.Get(key)
		t.Assert(v, value1)

		r, _ := cacheRedis.SetIfNotExist(key, value2, time.Second*2)

		t.Assert(r, false)

		v, _ = cacheRedis.Get(key)
		t.Assert(v, value1)

		d, _ := cacheRedis.GetExpire(key)
		t.Assert(d > time.Millisecond*500, true)
		t.Assert(d <= time.Second, true)
	})
}
