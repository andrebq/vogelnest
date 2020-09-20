package tweets

import (
	"os"
	"sync"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
)

type (
	// Stream is a suture Service which streams tweets into channels
	Stream struct {
		initialized   chan struct{}
		stop          chan struct{}
		cleanShutdown chan struct{}

		logCtx     zerolog.Logger
		sampledLog zerolog.Logger

		terms chan []string

		client *twitter.Client

		outputList struct {
			sync.Mutex
			output []chan *twitter.Tweet
		}
	}
)

var (
	percentFull = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "percentFull",
		Namespace: "vogelnest",
		Subsystem: "tweets",
	})
	droppedTweets = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "droppedTweets",
		Namespace: "vogelnest",
		Subsystem: "tweets",
	})
	tweetsRecvd = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "tweetsRecvd",
		Namespace: "vogelnest",
		Subsystem: "tweets",
	})
	undelivered = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "undelivered",
		Namespace: "vogelnest",
		Subsystem: "tweets",
	})
)

func init() {
	prometheus.MustRegister(percentFull, droppedTweets, tweetsRecvd, undelivered)
}

// NewStream with tweets
func NewStream() *Stream {
	s := &Stream{
		terms: make(chan []string),
	}
	return s
}

// Serve requests and panics if any error happens
func (s *Stream) Serve() {
	s.validState()
	s.init()
	defer s.cleanup()
	s.logCtx.Info().Msg("Starting stream, waiting for terms")
	var terms []string
	select {
	case terms = <-s.terms:
		s.logCtx.Info().Strs("terms", terms).Msg("Got terms to search for")
	case <-s.stop:
		return
	}

	ts, err := s.changeTerms(nil, terms)
	if err != nil {
		return
	}

	s.authenticate()
	defer ts.Stop()
	for {
		select {
		case terms = <-s.terms:
			if len(terms) == 0 {
				continue
			}
			ts, err = s.changeTerms(ts, terms)
			if err != nil {
				return
			}
		case <-s.stop:
			return
		case t, open := <-ts.Messages:
			if !open {
				s.logCtx.Info().Msg("Twitter stream closed")
				return
			}
			switch t := t.(type) {
			case *twitter.StreamLimit:
				s.sampledLog.Info().Int64("undelivered", t.Track).Msg("Search term is to broad, some tweets missed")
				undelivered.Set(float64(t.Track))
			case *twitter.StallWarning:
				s.logCtx.Warn().Str("event", "stall-warning").Int("percentFull", t.PercentFull).Str("code", t.Code).Msg(t.Message)
				percentFull.Set(float64(t.PercentFull))
			case *twitter.Tweet:
				tweetsRecvd.Inc()
				s.writeOutput(t)
			case *twitter.StreamDisconnect:
				s.logCtx.Warn().Str("event", "disconnect").Str("reason", t.Reason).Str("stream", t.StreamName).Send()
				return
			}
		}
	}
}

// Stop the service
func (s *Stream) Stop() {
	close(s.stop)

	<-s.cleanShutdown
}

// SetTerms can be used by clients to change which terms
// are being processed
func (s *Stream) SetTerms(t []string) bool {
	select {
	case s.terms <- t:
		return true
	case <-s.stop:
		return false
	}
}

// NewSink adds a new tweet sink to this stream
// slow consumers will have their messages dropped.
//
// When the stream is done accepting new tweets the output will be closed
func (s *Stream) NewSink(buf int) <-chan *twitter.Tweet {
	o := make(chan *twitter.Tweet, buf)
	s.outputList.Lock()
	s.outputList.output = append(s.outputList.output, o)
	s.outputList.Unlock()
	select {
	case <-s.stop:
		s.RemoveSink(o)
	default:
	}
	return o
}

// RemoveSink removes o from the sink and closes it
//
// Valid only if the output was part of this sink
func (s *Stream) RemoveSink(o <-chan *twitter.Tweet) {
	s.outputList.Lock()
	defer s.outputList.Unlock()
	last := len(s.outputList.output) - 1
	for i, v := range s.outputList.output {
		if v == o {
			s.outputList.output[i] = nil
			switch {
			case i == 0 && last == 0:
				s.outputList.output = s.outputList.output[:0]
			case i == last:
				s.outputList.output = s.outputList.output[:last-1]
			default:
				s.outputList.output[i] = s.outputList.output[last]
				s.outputList.output[last] = nil
				s.outputList.output = s.outputList.output[:last-1]
			}
		}
	}
}

func (s *Stream) writeOutput(t *twitter.Tweet) {
	s.outputList.Lock()
	defer s.outputList.Unlock()
	none := true
	for _, v := range s.outputList.output {
		select {
		case v <- t:
			none = false
		default:
		}
	}
	if none {
		s.dropTweet(t)
	}
}

func (s *Stream) dropTweet(t *twitter.Tweet) {
	droppedTweets.Inc()
}

func (s *Stream) cleanup() {
	defer close(s.cleanShutdown)
	s.outputList.Lock()
	defer s.outputList.Unlock()
	for _, v := range s.outputList.output {
		close(v)
	}
	s.logCtx.Info().Msg("Output closed")
}

func (s *Stream) connect() {
	config := oauth1.NewConfig(os.Getenv("TWITTER_API_KEY"), os.Getenv("TWITTER_API_SECRET_KEY"))
	token := oauth1.NewToken(os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"))
	httpClient := config.Client(oauth1.NoContext, token)

	s.client = twitter.NewClient(httpClient)
}

func (s *Stream) validState() {
}

func (s *Stream) init() {
	s.stop = make(chan struct{})
	s.cleanShutdown = make(chan struct{})
	s.logCtx = log.With().Str("service", "stream").Logger()
	s.sampledLog = s.logCtx.Sample(zerolog.Sometimes)

	s.connect()
}

func (s *Stream) authenticate() {
	user, res, err := s.client.Accounts.VerifyCredentials(&twitter.AccountVerifyParams{})
	if err != nil {
		s.logCtx.Error().Err(err).Int("status", res.StatusCode).Msg("Unable to verify account")
		panic(err)
	}
	s.logCtx.Info().Str("authenticated_as", user.ScreenName).Str("status", user.Status.Text).Msg("Account verified")
}

func (s *Stream) changeTerms(ts *twitter.Stream, terms []string) (*twitter.Stream, error) {
	if ts != nil {
		ts.Stop()
	}
	var err error
	ts, err = s.client.Streams.Filter(&twitter.StreamFilterParams{
		Track: terms,
	})
	if err != nil {
		s.logCtx.Error().Strs("terms", terms).Err(err).Msg("Unable to obtain stream from twitter")
		return nil, err
	}
	return ts, err
}

func (s *Stream) String() string {
	return "stream-service"
}
