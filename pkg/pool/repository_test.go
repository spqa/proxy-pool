package pool

import (
	"context"
	"proxy-pool/config"
	"proxy-pool/internal/core"
	"testing"
)

func TestService_SaveMany(t *testing.T) {
	repository := NewRepository(core.ProvideRedis(&config.Config{
		RedisAddr:     "localhost:6378",
		RedisPassword: "",
		RedisDb:       0,
	}))
	err := repository.saveMany(context.Background(), []*entity{
		{
			Ip:       "zproxy.lum-superproxy.io",
			Port:     22225,
			Type:     Http,
			Country:  "",
			Latency:  0,
			Username: "lum-customer-hl_487d21a4-zone-static-ip-2.56.19.73",
			Password: "zdpuov7gze7u",
		},
	})
	if err != nil {
		t.Error(err)
	}
}
