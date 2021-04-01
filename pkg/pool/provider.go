package pool

import "github.com/google/wire"

var Set = wire.NewSet(NewFetcherJob, NewCheckerService, NewRepository, NewPoolService)
