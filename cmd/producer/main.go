package main

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	server "ocelot/pkg/server"

	log "github.com/sirupsen/logrus"
)

func init() {
	// Logging
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true

	rand.Seed(time.Now().UTC().UnixNano())
}

var sConfig = &server.Config{
	Concurrency: 5,
	Host:        "0.0.0.0:2151",
}

var jConfigs = []*server.JobConfig{
	{
		Path:            "https://hello.com/en/index.html",
		Interval:        time.Millisecond * 300,
		MaxRetries:      3,
		BackoffFunction: server.ExpBackoff,
	},
}

// Define Context
func main() {

	// Start Server
	s := server.NewServer(sConfig, jConfigs)

	// Set Cancel...
	ctx, cancel := context.WithCancel(context.Background())

	// Start Timers
	for _, j := range s.Manager.Jobs {
		go j.StartJob(ctx, s.Manager.JobChan)
	}

	go s.Serve(ctx)

	// Block...
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	// Cancel Context
	<-termChan
	cancel()

}
