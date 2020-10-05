package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/andrebq/vogelnest/internal/schema"
	"github.com/rs/cors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type (
	Server struct {
		upgrader          *websocket.Upgrader
		actual            *http.Server
		shutdownCompleted chan struct{}
		addr              string
		port              int
		serveStatic       string
		setTerms          func([]string) bool
		addsink           func(int) <-chan *twitter.Tweet
		removesink        func(<-chan *twitter.Tweet)
		logCtx            zerolog.Logger
		sampledCtx        zerolog.Logger
		corsOrigins       []string
	}
)

// NewServer returns a suture compatible HTTP server using
func NewServer(addr string, port int, serveStatic string,
	corsOrigins []string,
	setTerms func([]string) bool,
	addsink func(int) <-chan *twitter.Tweet,
	removesink func(<-chan *twitter.Tweet)) *Server {
	s := &Server{
		addr:        addr,
		port:        port,
		serveStatic: serveStatic,
		setTerms:    setTerms,
		addsink:     addsink,
		removesink:  removesink,

		upgrader: &websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},

		logCtx:      log.With().Str("service", "api-server").Logger(),
		corsOrigins: corsOrigins,
	}
	s.sampledCtx = s.logCtx.Sample(zerolog.Sometimes)
	return s
}

// Serve incoming requests using the default mux
func (s *Server) Serve() {
	logCtx := s.logCtx
	s.actual = &http.Server{
		Addr:    fmt.Sprintf("%v:%v", s.addr, s.port),
		Handler: s.rootHandler(),
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

func (s *Server) rootHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/stream/terms", s.handleSetTerms)
	mux.HandleFunc("/stream/ws", s.handleWebsocket)
	if len(s.serveStatic) > 0 {
		fs := http.FileServer(http.Dir(s.serveStatic))
		mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("Cache-Control", "no-cache")
			switch {
			case strings.HasSuffix(req.URL.Path, ".wasm"):
				w.Header().Set("content-type", "application/wasm")
			case strings.HasSuffix(req.URL.Path, ".json"):
				w.Header().Set("content-type", "application/json")
			case strings.HasSuffix(req.URL.Path, ".js"):
				w.Header().Set("content-type", "application/js")
			case strings.HasSuffix(req.URL.Path, ".css"):
				w.Header().Set("content-type", "text/stylesheet")
			}
			fs.ServeHTTP(w, req)
		})
	}

	corsOpts := cors.Options{
		AllowedOrigins: s.corsOrigins,
		AllowedMethods: []string{
			"POST", "GET", "PUT", "DELETE", "UPGRADE", "CONNECT",
		},
		AllowCredentials: true,
	}
	s.logCtx.Info().Strs("cors-origins", corsOpts.AllowedOrigins).
		Strs("cors-methods", corsOpts.AllowedMethods).
		Strs("cors-headers", corsOpts.AllowedHeaders).
		Msg("CORS Options")

	c := cors.New(corsOpts)
	return c.Handler(mux)
}

func (s *Server) handleSetTerms(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "PUT":
		terms := struct {
			Terms string `json:"terms"`
		}{}
		err := json.NewDecoder(req.Body).Decode(&terms)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ok := s.setTerms(strings.Split(terms.Terms, ","))
		if !ok {
			http.Error(w, "unable to change terms", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (s *Server) handleWebsocket(w http.ResponseWriter, req *http.Request) {
	c, err := s.upgrader.Upgrade(w, req, nil)
	if err != nil {
		s.logCtx.Error().Err(err).Str("action", "handle-websocket").Msg("Unable to setup ws connection")
		return
	}
	s.sampledCtx.Info().Str("conn", c.RemoteAddr().String()).Msg("New WebSocket connection")
	defer c.Close()
	output := s.addsink(100)
	for v := range output {
		t := &schema.Tweet{}
		err = t.Populate(v)
		if err != nil {
			s.sampledCtx.Error().Err(err).Str("action", "convertTweet").Send()
			continue
		}
		buf, err := protojson.Marshal(t)
		if err != nil {
			s.sampledCtx.Error().Err(err).Str("action", "protjson").Send()
			continue
		}
		err = c.WriteMessage(websocket.TextMessage, buf)
		if err != nil {
			s.removesink(output)
		}
	}
}

func (s *Server) String() string { return "api-server" }
