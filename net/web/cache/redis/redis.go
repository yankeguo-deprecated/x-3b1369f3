// Copyright 2013 Beego Authors
// Copyright 2014 The Macaron Authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package cache

import (
	"fmt"
	"time"

	"landzero.net/x/com"

	"landzero.net/x/database/redis"
	"landzero.net/x/net/web/cache"
)

// RedisCacher represents a redis cache adapter implementation.
type RedisCacher struct {
	c          *redis.Client
	prefix     string
	hsetName   string
	occupyMode bool
}

// Put puts value into cache with key and expire time.
// If expired is 0, it lives forever.
func (c *RedisCacher) Put(key string, val interface{}, expire int64) error {
	key = c.prefix + key
	if expire == 0 {
		if err := c.c.Set(key, com.ToStr(val), 0).Err(); err != nil {
			return err
		}
	} else {
		dur, err := time.ParseDuration(com.ToStr(expire) + "s")
		if err != nil {
			return err
		}
		if err = c.c.Set(key, com.ToStr(val), dur).Err(); err != nil {
			return err
		}
	}

	if c.occupyMode {
		return nil
	}
	return c.c.HSet(c.hsetName, key, "0").Err()
}

// Get gets cached value by given key.
func (c *RedisCacher) Get(key string) interface{} {
	val, err := c.c.Get(c.prefix + key).Result()
	if err != nil {
		return nil
	}
	return val
}

// Delete deletes cached value by given key.
func (c *RedisCacher) Delete(key string) error {
	key = c.prefix + key
	if err := c.c.Del(key).Err(); err != nil {
		return err
	}

	if c.occupyMode {
		return nil
	}
	return c.c.HDel(c.hsetName, key).Err()
}

// Incr increases cached int-type value by given key as a counter.
func (c *RedisCacher) Incr(key string) error {
	if !c.IsExist(key) {
		return fmt.Errorf("key '%s' not exist", key)
	}
	return c.c.Incr(c.prefix + key).Err()
}

// Decr decreases cached int-type value by given key as a counter.
func (c *RedisCacher) Decr(key string) error {
	if !c.IsExist(key) {
		return fmt.Errorf("key '%s' not exist", key)
	}
	return c.c.Decr(c.prefix + key).Err()
}

// IsExist returns true if cached value exists.
func (c *RedisCacher) IsExist(key string) bool {
	if c.c.Exists(c.prefix+key).Val() != 0 {
		return true
	}

	if !c.occupyMode {
		c.c.HDel(c.hsetName, c.prefix+key)
	}
	return false
}

// Flush deletes all cached data.
func (c *RedisCacher) Flush() error {
	if c.occupyMode {
		return c.c.FlushDb().Err()
	}

	keys, err := c.c.HKeys(c.hsetName).Result()
	if err != nil {
		return err
	}
	if err = c.c.Del(keys...).Err(); err != nil {
		return err
	}
	return c.c.Del(c.hsetName).Err()
}

// StartAndGC starts GC routine based on config string settings.
// AdapterConfig: redis://localhost:3333/1
func (c *RedisCacher) StartAndGC(opts cache.Options) error {
	c.hsetName = "webcache"
	c.occupyMode = opts.OccupyMode

	opt, err := redis.ParseURL(opts.AdapterConfig)
	if err != nil {
		return err
	}

	c.c = redis.NewClient(opt)
	return c.c.Ping().Err()
}

func init() {
	cache.Register("redis", &RedisCacher{})
}
