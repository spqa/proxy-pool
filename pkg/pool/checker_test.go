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
		Ip:       "zproxy.lum-superproxy.io",
		Port:     22225,
		Type:     Https,
		Country:  "",
		Latency:  0,
		Username: "lum-customer-hl_487d21a4-zone-static-ip-2.56.19.73",
		Password: "zdpuov7gze7u",
	})
	if !result {
		t.Fail()
	}
}
