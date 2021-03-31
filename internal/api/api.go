package api

import "proxy-pool/config"

type Server struct {
	cfg *config.Config
}

func newApiServer(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}