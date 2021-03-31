package proxy

import (
	"io"
	"log"
	"net"
	"net/http"
	"proxy-pool/config"
)

type Proxy struct {
	cfg *config.Config
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
	targetConnection, err := net.Dial("tcp", host)
	socks.
	if err != nil {
		panic("failed to dial connection to target host: " + host)
	}

	// success establish connection to target host
	sourceConnection.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))

	go copyAndClose(sourceConnection, targetConnection)
	go copyAndClose(targetConnection, sourceConnection)
}

func copyAndClose(source io.ReadCloser, dest io.WriteCloser) {
	_, err := io.Copy(dest, source)
	if err != nil {
		log.Print("error copy to dest: " + err.Error())
	}
	_ = source.Close()
	_ = dest.Close()
}

func newProxy(config *config.Config) *Proxy {
	return &Proxy{
		cfg: config,
	}
}

func (p *Proxy) Start() error {
	return http.ListenAndServe(":3000", p)
}