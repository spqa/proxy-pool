package pool

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/mitchellh/mapstructure"
	"strings"
)

type Type string

const (
	Socks4 Type = "socks4"
	Socks5 Type = "socks5"
	Http   Type = "http"
	Https  Type = "https"
)

func parseType(t string) (Type, error) {
	t = strings.TrimSpace(t)
	t = strings.ToLower(t)
	switch t {
	case "socks4":
		return Socks4, nil
	case "socks5":
		return Socks5, nil
	case "http":
		return Http, nil
	case "https":
		return Https, nil
	default:
		return "", errors.New("unknown protocol " + t)
	}
}

const indexKey = "index:proxy:set"

type entity struct {
	Ip      string `mapstructure:"ip"`
	Port    int    `mapstructure:"port"`
	Type    Type   `mapstructure:"type"`
	Country string `mapstructure:"country"`
	Latency int64  `mapstructure:"latency"`
}

func (e entity) getProxyUri() string {
	return fmt.Sprintf("%v://%v:%v", e.Type, e.Ip, e.Port)
}

type repository struct {
	redis *redis.Client
}

func NewRepository(client *redis.Client) *repository {
	return &repository{redis: client}
}

func (r repository) saveMany(ctx context.Context, entities []*entity) error {
	pipeline := r.redis.Pipeline()
	for _, e := range entities {
		m := map[string]interface{}{}
		err := mapstructure.Decode(e, &m)
		if err != nil {
			return err
		}
		// save to hash
		pipeline.HMSet(ctx, buildKeyName(e), m)
		// add index
		pipeline.SAdd(ctx, indexKey, buildKeyName(e))
	}
	_, err := pipeline.Exec(ctx)
	return err
}

func (r repository) delete(ctx context.Context, entity *entity) error {
	pipeline := r.redis.Pipeline()
	// remove from index
	pipeline.SRem(ctx, indexKey, buildKeyName(entity))
	// remove hash
	pipeline.Del(ctx, buildKeyName(entity))
	_, err := pipeline.Exec(ctx)
	return err
}

func (r repository) getByRandom(ctx context.Context) (*entity, error) {
	// get random key name from index
	key, err := r.redis.SPop(ctx, indexKey).Result()
	if err != nil {
		return nil, err
	}

	// load data of that key
	result, err := r.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	e := entity{}
	err = mapstructure.Decode(result, &e)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func buildKeyName(e *entity) string {
	return fmt.Sprintf("proxy:%v:%v", e.Ip, e.Port)
}
