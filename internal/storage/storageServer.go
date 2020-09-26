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
		log *TweetLogWriter

		stream *tweets.Stream

		done chan struct{}
		stop chan struct{}
	}
)

// NewServer saving files to basedir and reading content from stream
func NewServer(basedir string, stream *tweets.Stream) (*Server, error) {
	s := &Server{}
	var err error
	s.log, err = NewLog(basedir)
	s.stream = stream
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) Serve() {
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	defer close(s.done)

	// avoid blocking at the cost of more memory
	// and maybe some data loss if the stream is closed
	sub := s.stream.NewSink(1000)
	defer s.stream.RemoveSink(sub)

	logctx := log.Logger.With().Str("service", "storage-server").Logger()

	buf := make([]*schema.Tweet, 0, 100)

	syncInterval := time.NewTicker(time.Second)
	defer syncInterval.Stop()

	defer s.closeTweetLog(logctx)
	for {
		select {
		case <-syncInterval.C:
			buf = s.flush(logctx, buf)
			continue
		case <-s.stop:
			buf = s.flush(logctx, buf)
			logctx.Info().Str("action", "stop").Msg("Got signal to stop storage server")
			return
		case t, open := <-sub:
			if !open {
				s.flush(logctx, buf)
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

func (s *Server) flush(ctx zerolog.Logger, buf []*schema.Tweet) []*schema.Tweet {
	err := s.log.Append(buf...)
	if err != nil {
		ctx.Error().Err(err).Str("action", "flush").Send()
	}
	// clear it
	return buf[:]
}

func (s *Server) closeTweetLog(ctx zerolog.Logger) {
	err := s.log.Close()
	if err != nil {
		ctx.Error().Err(err).Str("action", "close-tweet-log").Msg("Unable to close tweet log")
		return
	}
	ctx.Info().Str("action", "close-tweet-log").Msg("TweetLog closed!")
}

func (s *Server) Stop() {
	close(s.stop)
	<-s.done
}

func (s *Server) String() string { return "storage-service" }
