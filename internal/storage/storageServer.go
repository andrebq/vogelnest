package storage

import (
	"time"

	"github.com/andrebq/vogelnest/internal/schema"
	"github.com/andrebq/vogelnest/internal/tweets"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type (
	Server struct {
		log   *TweetLogWriter
		trunc time.Duration
		dir   string

		stream *tweets.Stream

		done chan struct{}
		stop chan struct{}
	}
)

// NewServer saving files to basedir and reading content from stream
func NewServer(basedir string, stream *tweets.Stream) (*Server, error) {
	s := &Server{
		trunc: time.Minute * 1,
		dir:   basedir,
	}
	s.stream = stream
	return s, nil
}

func (s *Server) Serve() {
	logctx := log.Logger.With().Str("service", "storage-server").Logger()
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	defer close(s.done)

	if s.log == nil {
		var err error
		s.log, err = NewLog(s.dir, s.trunc)
		if err != nil {
			logctx.Error().Err(err).Str("action", "newlog").Msg("Unable to create log to persist incoming messages")
			panic(err)
		}
	}

	// avoid blocking at the cost of more memory
	// and maybe some data loss if the stream is closed
	sub := s.stream.NewSink(1000)
	defer s.stream.RemoveSink(sub)

	buf := make([]*schema.Tweet, 0, 100)

	syncInterval := time.NewTicker(time.Second)
	defer syncInterval.Stop()

	truncInterval := time.NewTicker(s.trunc)
	defer truncInterval.Stop()

	defer s.packLog(logctx, false)
	for {
		select {
		case <-truncInterval.C:
			s.packLog(logctx, true)
		case <-syncInterval.C:
			buf = s.flush(logctx, buf)
			continue
		case <-s.stop:
			buf = s.flush(logctx, buf)
			s.packLog(logctx, false)
			logctx.Info().Str("action", "stop").Msg("Got signal to stop storage server")
			return
		case t, open := <-sub:
			if !open {
				s.flush(logctx, buf)
				s.packLog(logctx, false)
				logctx.Warn().Str("action", "subscription-closed").Msg("Input stream closed. There won't be any new messages")
				return
			}
			st := schema.Tweet{}
			err := st.Populate(t)
			if err != nil {
				logctx.Error().Err(err).Str("action", "populate").Msg("Unable to process tweet")
				continue
			}
			buf = append(buf, &st)
			if len(buf) == 100 {
				buf = s.flush(logctx, buf)
			}
		}
	}
}

func (s *Server) packLog(ctx zerolog.Logger, opennew bool) {
	if s.log == nil {
		return
	}
	err := s.log.Pack()
	if err != nil {
		ctx.Error().Err(err).Str("action", "packLog").Msg("Unable to pack tweet log")
	}
	s.log = nil

	if !opennew {
		return
	}

	s.log, err = NewLog(s.dir, s.trunc)
	if err != nil {
		s.log = nil
		ctx.Error().Err(err).Str("action", "packLog").Str("subAction", "openNextLog").Msg("Unable to open next log")
		panic(err)
	}
}

func (s *Server) flush(ctx zerolog.Logger, buf []*schema.Tweet) []*schema.Tweet {
	err := s.log.Append(buf...)
	if err != nil {
		ctx.Error().Err(err).Str("action", "flush").Send()
	}
	// clear it
	return buf[:]
}

func (s *Server) Stop() {
	close(s.stop)
	<-s.done
}

func (s *Server) String() string { return "storage-service" }
