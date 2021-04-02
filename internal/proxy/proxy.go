package proxy

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"io"
	"net"
	"net/http"
	"proxy-pool/config"
	"proxy-pool/pkg/log"
	"proxy-pool/pkg/pool"
	"time"
)

const maxTryCount = 5

type Proxy struct {
	cfg         *config.Config
	poolService *pool.Service
}

func newProxy(
	cfg *config.Config,
	poolService *pool.Service,
) *Proxy {
	return &Proxy{
		cfg:         cfg,
		poolService: poolService,
	}
}

type halfClosable interface {
	net.Conn
	CloseWrite() error
	CloseRead() error
}

func (p *Proxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			log.Logger.Error("panic", zap.Any("error", err))
			writer.WriteHeader(500)
			_, _ = writer.Write([]byte("internal server error"))
		}
	}()
	log.Logger.Debug("request", zap.String("host", request.Host))
	hij, ok := writer.(http.Hijacker)
	if !ok {
		panic("hijacking the connection is not supported")
	}
	sourceConnection, _, err := hij.Hijack()
	if err != nil {
		panic("failed to hijack connection, error: " + err.Error())
	}
	host := request.Host
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelFunc()
	targetConnection, err := p.tryDialConnectionToHost(ctx, host)
	if err != nil {
		panic("failed to dial connection to target host: " + host + ", error: " + err.Error())
	}
	// success establish connection to target host
	_, err = sourceConnection.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	if err != nil {
		panic("failed to write success header")
	}

	sourceClosableConn, ok := sourceConnection.(halfClosable)
	if !ok {
		panic("failed to cast source connection to closable connection")
	}
	targetClosableConn, ok := targetConnection.(halfClosable)
	if !ok {
		panic("failed to cast target connection to closable connection")
	}
	go copyAndClose(sourceClosableConn, targetClosableConn)
	go copyAndClose(targetClosableConn, sourceClosableConn)
}

func (p Proxy) tryDialConnectionToHost(ctx context.Context, host string) (net.Conn, error) {
	entities, err := p.poolService.GetByRandom(ctx, maxTryCount)
	if err != nil {
		return nil, err
	}
	tryCount := 1
	var targetConnection net.Conn
	for tryCount <= min(maxTryCount, len(entities)) {
		entity := entities[tryCount-1]
		dialFunc := entity.GetDialFunc()
		log.Logger.Debug("trying proxy", zap.String("proxy", entity.GetProxyUri()), zap.Int("tryCount", tryCount))
		targetConnection, err = dialFunc("tcp", host)
		if err != nil {
			log.Logger.Debug("trying proxy error", zap.Error(err))
			err := p.poolService.Delete(ctx, entity)
			if err != nil {
				log.Logger.Warn("remove from pool error", zap.Error(err))
			}
			tryCount++
			continue
		}
		break
	}
	if tryCount > min(maxTryCount, len(entities)) {
		return nil, errors.New("maximum retry reached")
	}
	return targetConnection, nil
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func copyAndClose(sourceConn halfClosable, destConn halfClosable) {
	_, err := io.Copy(sourceConn, destConn)
	if err != nil {
		log.Logger.Error("error copy to dest: " + err.Error())
	}
	_ = sourceConn.CloseRead()
	_ = destConn.CloseWrite()
}

func (p *Proxy) Start() error {
	go p.poolService.Start(context.Background())
	log.Logger.Info("starting proxy server")
	return http.ListenAndServe(":3001", p)
}
