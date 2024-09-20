package reportorV3

import "github.com/go-redis/redis/v8"

func (p ReportorConfig) NewRedisClusterManager() *redis.ClusterClient {
	// 解析redis-config

	var redisHost []string
	redisHost = append(redisHost, p.RedisHost)

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    redisHost,
		Password: p.RedisPass,
	})

	return rdb

}

func (p ReportorConfig) NewRedisManager() *redis.Client {
	// 解析redis-config

	if len(p.RedisHost) > 1 {
		var rdb = redis.NewClient(&redis.Options{
			Addr:     p.RedisHost,
			Password: p.RedisPass,
			PoolSize: 100,
		})
		return rdb
	}

	return nil

}
