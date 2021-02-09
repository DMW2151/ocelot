package main

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	ocelot "ocelot/pkg/ocelot"

	"github.com/google/uuid"
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

// Initialize Job Params...
var pConfig = &ocelot.ProducerConfig{
	JobChannelBuffer: 5,
	ListenAddr:       os.Getenv("OCELOT_LISTEN_ADDR"),
}

var jobs = []*ocelot.Job{
	{
		ID:       uuid.New(),
		Interval: time.Millisecond * 300,
		Path:     "https://hello.com/en/index.html",
	},
}

// Define Context
func main() {

	// Start Producer
	p, _ := ocelot.NewProducer(pConfig, jobs)

	// Set Cancel...
	ctx, cancel := context.WithCancel(context.Background())

	// Start Timers for each job Available in the Jobpool
	// on server start

	// TODO (??): defer this until a connection is made available,
	// prevents throttle on start...
	for _, j := range p.JobPool.Jobs {
		go j.StartSchedule(ctx, p.JobPool.JobChan)
	}

	go p.Serve(ctx)

	// Block...
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	// Cancel Context
	<-termChan
	cancel()

}
