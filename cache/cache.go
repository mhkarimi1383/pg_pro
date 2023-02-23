package cache

import (
	"context"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/coocood/freecache"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	bigcache_store "github.com/eko/gocache/store/bigcache/v4"
	freecache_store "github.com/eko/gocache/store/freecache/v4"
	memcache_store "github.com/eko/gocache/store/memcache/v4"
	"github.com/pkg/errors"

	"github.com/mhkarimi1383/pg_pro/config"
)

var (
	ctx          context.Context
	cacheManager *cache.Cache[[]byte]
)

func init() {
	ctx = context.Background()
	switch config.GetString("cache.backend") {
	case "memcached":
		memcacheStore := memcache_store.NewMemcache(
			memcache.New(config.GetStringSlice("cache.connection_info")...),
			store.WithExpiration(config.GetDuration("cache.ttl")*time.Second),
		)
		cacheManager = cache.New[[]byte](memcacheStore)
	case "bigcache":
		bigCacheClient, err := bigcache.New(ctx, bigcache.DefaultConfig(config.GetDuration("cache.ttl")+5*time.Second))
		if err != nil {
			panic(errors.Wrap(err, "inializing bigcache client"))
		}
		bigcacheStore := bigcache_store.NewBigcache(
			bigCacheClient,
			store.WithExpiration(config.GetDuration("cache.ttl")*time.Second),
		)
		cacheManager = cache.New[[]byte](bigcacheStore)
	case "freecache":
		freecacheStore := freecache_store.NewFreecache(
			freecache.NewCache(config.GetInt("cache.connection_info")),
			store.WithExpiration(config.GetDuration("cache.ttl")*time.Second),
		)
		cacheManager = cache.New[[]byte](freecacheStore)
	}

	// err := cacheManager.Set(ctx, "my-key", []byte("my-value"),
	// 	store.WithExpiration(15*time.Second), // Override default value of 10 seconds defined in the store
	// )
	// if err != nil {
	// 	panic(err)
	// }

	// value, _ := cacheManager.Get(ctx, "my-key")
	// log.Println(value)

	// cacheManager.Delete(ctx, "my-key")

	// cacheManager.Clear(ctx) // Clears the entire cache, in case you want to flush all cache
}
