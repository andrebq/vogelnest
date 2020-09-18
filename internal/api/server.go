package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

type (
	Server struct {
		actual            *http.Server
		shutdownCompleted chan struct{}
		addr              string
		port              int
	}
)

func init() {
	http.Handle("/metrics", promhttp.Handler())
}

func NewServer(addr string, port int) *Server {
	return &Server{
		addr: addr,
		port: port,
	}
}

// Serve incoming requests using the default mux
func (s *Server) Serve() {
	logCtx := log.With().Str("service", "api-server").Logger()
	s.actual = &http.Server{
		Addr: fmt.Sprintf("%v:%v", s.addr, s.port),
	}
	logCtx.Info().Str("addr", s.addr).Int("port", s.port).Msg("Starting HTTP API Server")
	err := s.actual.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return
	} else if err != nil {
		panic(err)
	}
}

// Stop the server and wait for all connections to be closed
func (s *Server) Stop() {
	s.actual.Shutdown(context.Background())
}
