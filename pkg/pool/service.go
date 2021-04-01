package pool

import "context"

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
	return s.repository.delete(ctx, entity)
}

func (s Service) GetByRandom(ctx context.Context) (*entity, error) {
	return s.repository.getByRandom(ctx)
}

func (s Service) Start(ctx context.Context) {
	s.fetcherJob.Setup()
	s.checkerService.ProcessQueue(ctx, func(e *entity) error {
		return s.SaveMany(ctx, []*entity{e})
	})
}
