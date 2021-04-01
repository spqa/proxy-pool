package pool

import (
	"proxy-pool/internal/core"
	"testing"
)

func TestCheckerService_Check(t *testing.T) {
	checkerService := &CheckerService{
		redis: core.ProvideRedis(core.ProvideConfig()),
	}
	result := checkerService.Check(&entity{
		Ip:      "102.65.3.140",
		Port:    4153,
		Type:    Socks4,
		Country: "",
		Latency: 0,
	})
	if !result {
		t.Fail()
	}
}
