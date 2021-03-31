//+build wireinject

package api

import (
	"github.com/google/wire"
	"proxy-pool/internal/core"
)

func NewApiServer() *Server {
	panic(wire.Build(core.Set, newApiServer))
}