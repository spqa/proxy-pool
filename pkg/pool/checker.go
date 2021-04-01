package pool

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"h12.io/socks"
	"net"
	"net/http"
	"proxy-pool/pkg/log"
	"time"
)

const queueName = "checker:proxy:queue"
const maxConcurrentCheck = 20

type CheckerService struct {
	redis *redis.Client
}

func NewCheckerService(redis *redis.Client) *CheckerService {
	return &CheckerService{redis: redis}
}

func (c *CheckerService) Check(entity *entity) bool {
	log.Logger.Info("start checking", zap.String("proxy", entity.getProxyUri()))
	var tr *http.Transport
	if entity.Type == Socks4 || entity.Type == Socks5 {
		dial := socks.Dial(entity.getProxyUri())
		tr = &http.Transport{DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dial(network, addr)
		}}
	}
	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: tr,
	}
	resp, err := client.Get("https://m.tiktok.com")
	if err != nil {
		log.Logger.Warn("check failed",
			zap.String("error", err.Error()),
			zap.String("proxy", entity.getProxyUri()),
		)
		return false
	}
	return resp.StatusCode == http.StatusOK
}

func (c CheckerService) AddToQueue(ctx context.Context, entity *entity) error {
	bytes, err := json.Marshal(entity)
	if err != nil {
		return err
	}
	_ = c.redis.RPush(ctx, queueName, string(bytes))
	return nil
}

type checkSuccessFunc func(entity *entity) error

func (c CheckerService) ProcessQueue(ctx context.Context, successFunc checkSuccessFunc) {
	log.Logger.Info("starting process checker queue")
	messageChannel := make(chan *entity, maxConcurrentCheck)
	slotChannel := make(chan int, maxConcurrentCheck)
	go func() {
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case e := <-messageChannel:
				go func() {
					if c.Check(e) {
						err := successFunc(e)
						log.Logger.Error("failed to process success func", zap.Error(err))
					}
					// release slot
					<-slotChannel
				}()
			}
		}
	}()
loop:
	for {
		select {
		case <-ctx.Done():
			log.Logger.Info("context ended, stop polling queue")
			break loop
		default:
			result, err := c.redis.BLPop(ctx, time.Second, queueName).Result()
			if errors.Is(err, redis.Nil) {
				continue
			}
			if err != nil {
				log.Logger.Error("polling queue failed", zap.Error(err))
				break
			}
			e := entity{}
			err = json.Unmarshal([]byte(result[1]), &e)
			if err != nil {
				log.Logger.Error("unmarshal queue message failed", zap.Error(err))
				break
			}
			messageChannel <- &e
			// take a slot to process
			slotChannel <- 1
		}
	}
	log.Logger.Info("stopped process checker queue")
}
