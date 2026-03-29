package dbs

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"navfirst.com/nav-common-go-lib/configs"
	"navfirst.com/nav-common-go-lib/global"
)

func InitRedis() error {
	if global.NAV_CONFIG.Redis.Addr != "" {
		redisClient, err := initRedisClient(global.NAV_CONFIG.Redis)
		if err != nil {
			return err
		}
		global.NAV_REDIS = redisClient
	}
	return nil
}

func initRedisClient(redisCfg configs.Redis) (redis.UniversalClient, error) {
	var client redis.UniversalClient
	if redisCfg.UseCluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    redisCfg.ClusterAddrs,
			Password: redisCfg.Password,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     redisCfg.Addr,
			Password: redisCfg.Password,
			DB:       redisCfg.DB,
		})
	}
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		global.NAV_LOG.Error("redis connect ping failed, err:", zap.String("name", redisCfg.Name), zap.Error(err))
		return nil, err
	}

	global.NAV_LOG.Info("redis connect ping response:", zap.String("name", redisCfg.Name), zap.String("pong", pong))
	return client, nil
}
