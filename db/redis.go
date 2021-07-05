package db

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
)

var Redis *redis.Client

// ConfigRedis Redis配置参数
type ConfigRedis struct {
	Addr     string `ini:"host"`
	Port     int    `ini:"port"`
	Auth     string `ini:"auth"`
	SelectDB int    `ini:"select_db"`
}

// ConnectRedis 连接Redis 成功返回nil
func ConnectRedis(c *ConfigRedis) error {
	var (
		opt *redis.Options
		err error
	)
	if c.Port == 0 {
		c.Port = 6379
	}
	opt = &redis.Options{
		Network:  "tcp",
		Addr:     fmt.Sprintf("%s:%d", c.Addr, c.Port),
		Password: c.Auth,
		DB:       c.SelectDB,
	}
	Redis = redis.NewClient(opt)
	_, err = Redis.Ping(context.Background()).Result()
	return err
}
