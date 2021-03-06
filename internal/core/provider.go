package core

import (
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"github.com/spf13/viper"
	"proxy-pool/config"
)

func ProvideConfig() *config.Config {
	var cfg *config.Config
	_ = viper.Unmarshal(&cfg)
	return cfg
}

func ProvideRedis(config *config.Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword, // no password set
		DB:       config.RedisDb,       // use default DB
	})
	return rdb
}

var Set = wire.NewSet(ProvideConfig, ProvideRedis)