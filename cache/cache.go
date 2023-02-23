package cache

import (
	"context"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/coocood/freecache"
	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	bigcache_store "github.com/eko/gocache/store/bigcache/v4"
	freecache_store "github.com/eko/gocache/store/freecache/v4"
	go_cache_store "github.com/eko/gocache/store/go_cache/v4"
	memcache_store "github.com/eko/gocache/store/memcache/v4"
	pegasus_store "github.com/eko/gocache/store/pegasus/v4"
	redis_store "github.com/eko/gocache/store/redis/v4"
	rediscluster_store "github.com/eko/gocache/store/rediscluster/v4"
	ristretto_store "github.com/eko/gocache/store/ristretto/v4"
	rueidis_store "github.com/eko/gocache/store/rueidis/v4"
	"github.com/go-redis/redis/v8"
	go_cache "github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rueian/rueidis"

	"github.com/mhkarimi1383/pg_pro/config"
)

var (
	ctx          context.Context
	cacheManager *cache.Cache[[]byte]
)

func init() {
	ctx = context.Background()
	storeOpts := []store.Option{
		store.WithExpiration(config.GetDuration("cache.ttl") * time.Second),
	}
	switch config.GetString("cache.backend") {
	case "memcached":
		memcacheStore := memcache_store.NewMemcache(
			memcache.New(config.GetStringSlice("cache.connection_info")...),
			storeOpts...,
		)
		cacheManager = cache.New[[]byte](memcacheStore)
	case "bigcache":
		bigCacheClient, err := bigcache.New(ctx, bigcache.DefaultConfig(config.GetDuration("cache.ttl")+5*time.Second))
		if err != nil {
			panic(errors.Wrap(err, "inializing bigcache client"))
		}
		bigcacheStore := bigcache_store.NewBigcache(
			bigCacheClient,
			storeOpts...,
		)
		cacheManager = cache.New[[]byte](bigcacheStore)
	case "freecache":
		freecacheStore := freecache_store.NewFreecache(
			freecache.NewCache(config.GetInt("cache.connection_info")),
			storeOpts...,
		)
		cacheManager = cache.New[[]byte](freecacheStore)
	case "go-cache":
		gocacheStore := go_cache_store.NewGoCache(
			go_cache.New(config.GetDuration("cache.ttl")+5*time.Second, config.GetDuration("cache.connection_info")+5*time.Second),
			storeOpts...,
		)
		cacheManager = cache.New[[]byte](gocacheStore)
	case "pegasus":
		pegasusStore, err := pegasus_store.NewPegasus(
			ctx,
			&pegasus_store.OptionsPegasus{
				MetaServers: config.GetStringSlice("cache.connection_info"),
				Options: &store.Options{
					Expiration: config.GetDuration("cache.ttl") * time.Second,
				},
			},
		)
		if err != nil {
			panic(errors.Wrap(err, "inializing pegasus client"))
		}
		cacheManager = cache.New[[]byte](pegasusStore)
	case "redis":
		redisStore := redis_store.NewRedis(
			redis.NewClient(&redis.Options{
				Addr:     config.GetString("cache.connection_info.addr"),
				Password: config.GetString("cache.connection_info.password"),
				DB:       config.GetInt("cache.connection_info.db"),
			}),
			storeOpts...,
		)
		cacheManager = cache.New[[]byte](redisStore)
	case "rediscluster":
		redisclusterStore := rediscluster_store.NewRedisCluster(
			redis.NewClusterClient(
				&redis.ClusterOptions{
					Addrs:    config.GetStringSlice("cache.connection_info.addrs"),
					Password: config.GetString("cache.connection_info.password"),
				},
			),
			storeOpts...,
		)
		cacheManager = cache.New[[]byte](redisclusterStore)
	case "ristretto":
		ristrettoClient, err := ristretto.NewCache(&ristretto.Config{
			NumCounters: config.GetInt64("cache.connection_info.max_counter"),
			MaxCost:     config.GetInt64("cache.connection_info.max_cost"),
			BufferItems: config.GetInt64("cache.connection_info.buffer_items"),
		})
		if err != nil {
			panic(errors.Wrap(err, "inializing ristretto client"))
		}
		ristrettoStore := ristretto_store.NewRistretto(
			ristrettoClient,
			storeOpts...,
		)
		cacheManager = cache.New[[]byte](ristrettoStore)
	case "rueidis":
		rueidisclient, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress: config.GetStringSlice("cache.connection_info.addrs"),
			Password:    config.GetString("cache.connection_info.password"),
		})
		if err != nil {
			panic(err)
		}
		rueidisStore := rueidis_store.NewRueidis(
			rueidisclient,
			storeOpts...,
		)
		cacheManager = cache.New[[]byte](rueidisStore)
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
