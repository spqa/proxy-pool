package pool

import (
	"context"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"proxy-pool/pkg/log"
	"sync"
)

type FetcherJob struct {
	fetchers       []Fetcher
	checkerService *CheckerService
}

func NewFetcherJob(service *CheckerService) *FetcherJob {
	return &FetcherJob{
		checkerService: service,
	}
}

func (f *FetcherJob) RegisterFetcher(fetcher Fetcher) {
	log.Logger.Info("register fetcher: " + fetcher.Name())
	f.fetchers = append(f.fetchers, fetcher)
}

func (f *FetcherJob) ProcessFetcher(ctx context.Context, fetcher Fetcher, group *sync.WaitGroup) {
	entities, err := fetcher.Get()
	if err != nil {
		log.Logger.Error("failed to process fetcher", zap.String("name", fetcher.Name()), zap.Error(err))
	}
	for _, v := range entities {
		err = f.checkerService.AddToQueue(ctx, v)
		if err != nil {
			log.Logger.Error("failed to enqueue checker", zap.Error(err))
		}
	}
	if err != nil {
		log.Logger.Error("failed to process fetcher", zap.String("name", fetcher.Name()), zap.Error(err))
	}
	log.Logger.Info("finish fetcher job", zap.String("name", fetcher.Name()))
	group.Done()
}

func (f *FetcherJob) Start() {
	log.Logger.Info("starting process fetcher job")
	ctx := context.Background()
	var group sync.WaitGroup
	group.Add(len(f.fetchers))
	for _, fetcher := range f.fetchers {
		go f.ProcessFetcher(ctx, fetcher, &group)
	}
	group.Wait()
	log.Logger.Info("finished all fetcher jobs")
}

func (f *FetcherJob) Setup() {
	log.Logger.Info("setting up fetcher cron job")
	f.RegisterFetcher(&ProxyHubFetcher{})
	log.Logger.Info("fetcher count", zap.Int("count", len(f.fetchers)))
	c := cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	_, err := c.AddFunc("50 * * * * *", func() {
		f.Start()
	})
	if err != nil {
		log.Logger.Error("failed to setup cron job", zap.Error(err))
	}
	c.Start()
}
