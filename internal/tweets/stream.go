package tweets

import (
	"errors"
	"os"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
)

type (
	// Stream is a suture Service which streams tweets into channels
	Stream struct {
		stop          chan struct{}
		cleanShutdown chan struct{}

		logCtx     zerolog.Logger
		sampledLog zerolog.Logger

		terms []string

		client *twitter.Client
		output []chan *twitter.Tweet
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

// NewStream with tweets containing the given terms
func NewStream(terms []string) *Stream {
	return &Stream{
		terms: terms,
	}
}

// Serve requests and panics if any error happens
func (s *Stream) Serve() {
	s.validState()
	s.init()
	defer s.cleanup()
	s.authenticate()
	s.logCtx.Info().Msg("Starting stream")

	ts, err := s.client.Streams.Filter(&twitter.StreamFilterParams{
		Track: s.terms,
	})
	if err != nil {
		s.logCtx.Error().Strs("terms", s.terms).Err(err).Msg("Unable to obtain stream from twitter")
		return
	}
	defer ts.Stop()
	for {
		select {
		case <-s.stop:
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

func (s *Stream) writeOutput(t *twitter.Tweet) {
	for _, v := range s.output {
		select {
		case v <- t:
		default:
		}
	}
	s.dropTweet(t)
}

func (s *Stream) dropTweet(t *twitter.Tweet) {
	droppedTweets.Inc()
}

func (s *Stream) cleanup() {
	defer close(s.cleanShutdown)
	for _, v := range s.output {
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
	if len(s.terms) == 0 {
		panic(errors.New("empty terms"))
	}
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

func (s *Stream) String() string {
	return "stream-service"
}
