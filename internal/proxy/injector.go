//+build wireinject

package proxy

import (
	"github.com/google/wire"
	"proxy-pool/internal/core"
	"proxy-pool/pkg/pool"
)

func NewProxy() *Proxy {
	panic(wire.Build(core.Set, pool.Set, newProxy))
}
