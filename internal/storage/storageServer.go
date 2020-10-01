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

		maxSize   int64
		packlimit int64

		stream *tweets.Stream

		done chan struct{}
		stop chan struct{}
	}
)

const (
	kilobytes = 1000
	megabytes = 1000 * kilobytes
)

// NewServer saving files to basedir and reading content from stream
func NewServer(basedir string, stream *tweets.Stream) (*Server, error) {
	s := &Server{
		trunc:     time.Minute * 1,
		dir:       basedir,
		maxSize:   300 * megabytes,
		packlimit: 50 * megabytes,
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
		s.log, err = NewLog(s.dir)
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

	packInterval := time.NewTimer(s.trunc)
	defer packInterval.Stop()

	truncate := time.NewTicker(time.Minute * 1)
	defer truncate.Stop()

	checkSizeInterval := time.NewTicker(s.trunc / 4)
	defer checkSizeInterval.Stop()

	defer s.closeLog(logctx)
	for {
		select {
		case <-packInterval.C:
			s.packLog(logctx)
		case <-checkSizeInterval.C:
			size, err := s.log.UnpackedSize()
			if err != nil {
				logctx.Error().Err(err).Str("action", "checkSize").Msg("Unable to compute size")
				continue
			}
			if size > s.packlimit {
				if s.packLog(logctx) {
					packInterval.Reset(s.trunc)
				}
			}
		case <-truncate.C:
			s.truncate(logctx)
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
				s.closeLog(logctx)
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

func (s *Server) Stop() {
	close(s.stop)
	<-s.done
}

func (s *Server) closeLog(logctx zerolog.Logger) {
	err := s.log.Close()
	if err != nil {
		logctx.Error().Err(err).Str("action", "close").Msg("Unable to close log")
	}
}

func (s *Server) packLog(logctx zerolog.Logger) bool {
	err := s.log.Pack()
	if err != nil {
		logctx.Error().Err(err).Str("action", "pack").Msg("Unable to pack log")
		return false
	}
	return true
}

func (s *Server) truncate(logctx zerolog.Logger) {
	segments, err := s.log.ComputeTrim(s.maxSize)
	if err != nil {
		logctx.Error().Err(err).Str("action", "truncate").Msg("Unable to compute trim")
		panic(err)
	}
	if len(segments) == 0 {
		return
	}
	err = s.log.Trim(segments...)
	if err != nil {
		logctx.Error().Err(err).Str("action", "truncate").Strs("segmentsToTrim", segments).Msg("Unable to perform trim")
		panic(err)
	}
	logctx.Info().Str("action", "truncate").Strs("segments", segments).Msg("Trim performed")
}

func (s *Server) String() string { return "storage-service" }
