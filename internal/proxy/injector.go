//+build wireinject

package proxy

import (
	"github.com/google/wire"
	"proxy-pool/internal/core"
)

func NewProxy() *Proxy {
	panic(wire.Build(core.Set, newProxy))
}
