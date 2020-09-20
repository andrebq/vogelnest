package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/andrebq/vogelnest/internal/api"
	"github.com/andrebq/vogelnest/internal/tweets"
	"github.com/rs/zerolog/log"
	"github.com/thejerf/suture"
)

var (
	terms       = flag.String("terms", "vogelnest,andrebq", "Terms to track")
	bind        = flag.String("bind", "0.0.0.0", "Address to listen for incoming HTTP requests")
	port        = flag.Int("port", 8080, "Port to listen for incoming requests")
	serveStatic = flag.String("serve-static", "", "When set, serve static files from this directory")
)

func main() {
	flag.Parse()

	rootLogger := log.With().Str("supervisor", "root").Logger()
	rootSupervisor := suture.New("root", suture.Spec{
		Log:     func(s string) { rootLogger.Warn().Msg(s) },
		Timeout: time.Minute,
	})
	stream := tweets.NewStream()
	rootSupervisor.Add(stream)
	rootSupervisor.Add(api.NewServer(*bind, *port, *serveStatic,
		strings.Split(os.Getenv("CORS_ORIGINS"), ","),
		stream.SetTerms, stream.NewSink, stream.RemoveSink))
	rootSupervisor.ServeBackground()
	wait(rootSupervisor)
}

func wait(rootSupervisor *suture.Supervisor) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	unstopped := rootSupervisor.StopWithReport()
	for _, v := range unstopped {
		log.Warn().Str("supervisor", "root").Str("service", v.Name).Msg("Failed to stop on time")
	}
}
