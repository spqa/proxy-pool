package proxy

import (
	"context"
	"h12.io/socks"
	"io"
	"log"
	"net"
	"net/http"
	"proxy-pool/config"
	"proxy-pool/pkg/pool"
)

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

func (p *Proxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	hij, ok := writer.(http.Hijacker)
	if !ok {
		panic("hijacking the connection is not supported")
	}
	sourceConnection, _, err := hij.Hijack()
	if err != nil {
		panic("failed to hijack connection, error: " + err.Error())
	}
	host := request.Host
	//targetConnection, err := net.Dial("tcp", host)
	//http.Transport{}
	dialFunc := socks.Dial("socks4://83.238.80.30:5678?timeout=5s")
	targetConnection, err := dialFunc("tcp", host)
	if err != nil {
		panic("failed to dial connection to target host: " + host + ", error: " + err.Error())
	}
	// success establish connection to target host
	_, err = sourceConnection.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	if err != nil {
		panic("failed to write success header")
	}

	sourceTcp, ok := sourceConnection.(*net.TCPConn)
	if !ok {
		panic("failed to cast source connection to TCP")
	}
	targetTcp, ok := targetConnection.(*net.TCPConn)
	if !ok {
		panic("failed to cast target connection to TCP")
	}
	go copyAndClose(sourceTcp, targetTcp)
	go copyAndClose(targetTcp, sourceTcp)
}

func copyAndClose(sourceConn *net.TCPConn, destConn *net.TCPConn) {
	_, err := io.Copy(sourceConn, destConn)
	if err != nil {
		log.Print("error copy to dest: " + err.Error())
	}
	_ = sourceConn.CloseRead()
	_ = destConn.CloseWrite()
}

func (p *Proxy) Start() error {
	p.poolService.Start(context.Background())
	return http.ListenAndServe(":3000", p)
}
