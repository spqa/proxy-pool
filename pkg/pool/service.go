package pool

import (
	"context"
	"go.uber.org/zap"
	"proxy-pool/pkg/log"
)

type Service struct {
	repository     *repository
	fetcherJob     *FetcherJob
	checkerService *CheckerService
}

func NewPoolService(repo *repository, job *FetcherJob, checker *CheckerService) *Service {
	return &Service{
		repository:     repo,
		fetcherJob:     job,
		checkerService: checker,
	}
}

func (s Service) SaveMany(ctx context.Context, entities []*entity) error {
	return s.repository.saveMany(ctx, entities)
}

func (s Service) Delete(ctx context.Context, entity *entity) error {
	log.Logger.Info("removing proxy from pool", zap.String("proxy", entity.GetProxyUri()))
	return s.repository.delete(ctx, entity)
}

func (s Service) GetByRandom(ctx context.Context, count int64) ([]*entity, error) {
	return s.repository.getByRandom(ctx, count)
}

func (s Service) Start(ctx context.Context) {
	s.fetcherJob.Setup()
	s.checkerService.ProcessQueue(ctx, func(e *entity) error {
		return s.SaveMany(ctx, []*entity{e})
	})
}
