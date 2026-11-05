package dbs

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wfu-work/nav-common-go-lib/configs"
	"github.com/wfu-work/nav-common-go-lib/global"
	"go.uber.org/zap"
)

func InitRedis() error {
	config := global.Config()
	if !config.System.UseRedis {
		global.NAV_REDIS = nil
		return nil
	}
	redisClient, err := initRedisClient(config.Redis)
	if err != nil {
		return err
	}
	global.NAV_REDIS = redisClient
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		_ = client.Close()
		global.NAV_LOG.Error("redis connect ping failed, err:", zap.String("name", redisCfg.Name), zap.Error(err))
		return nil, err
	}

	global.NAV_LOG.Info("redis connect ping response:", zap.String("name", redisCfg.Name), zap.String("pong", pong))
	return client, nil
}
