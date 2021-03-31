package config

type Config struct {
	RedisAddr         string `mapstructure:"redis_addr"`
	RedisPassword     string `mapstructure:"redis_password"`
	RedisDb           int    `mapstructure:"redis_db"`
}