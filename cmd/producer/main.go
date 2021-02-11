package main

import (
	"context"
	"math/rand"
	"os"
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
	log.SetLevel(log.DebugLevel)
	rand.Seed(time.Now().UTC().UnixNano())
}

// Initialize Job Params...
var pConfig = &ocelot.ProducerConfig{
	JobChannelBuffer: 0,
	ListenAddr:       os.Getenv("OCELOT_LISTEN_ADDR"),
	MaxConnections:   2,
}

var jobs = []*ocelot.Job{
	{
		ID:          uuid.New(),
		Interval:    time.Millisecond * 250,
		Path:        "https://hello.com/en/index.html",
		StagingChan: make(chan *ocelot.JobInstance, 2),
	},
}

var newJob = ocelot.Job{
	ID:          uuid.New(),
	Interval:    time.Millisecond * 3000,
	Path:        "https://www.holland.com/global/tourism.htm",
	StagingChan: make(chan *ocelot.JobInstance, 2),
}

// Define Context
func main() {

	// Set Cancel...
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Producer
	p := pConfig.NewProducer(jobs)

	// Start Timers for each job Available in the Jobpool
	// on server start
	// p.Serve(ctx)

	// Add && Remove some Tasks...
	p.Serve(ctx)

	// TODO; THIS ROUTINE PRODUCES ERROR...
	// time.Sleep(1 * time.Second)
	// p.JobPool.StopJob()

	// time.Sleep(1 * time.Second)

	// // Start a New Job
	// go p.JobPool.StartJob(ctx, &newJob)

	// time.Sleep(10 * time.Second)

}
