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

// Initialize Job Params...
var (
	pConfig = &ocelot.ProducerConfig{
		JobChannelBuffer: 0,
		ListenAddr:       os.Getenv("OCELOT_LISTEN_ADDR"),
		MaxConnections:   2,
	}
	jobs = make([]*ocelot.Job, 10)
)

func newMock() *ocelot.Job {
	return &ocelot.Job{
		ID:          uuid.New(),
		Interval:    time.Millisecond * 3000,
		Path:        "https://path/to/fake/domain/index.html",
		StagingChan: make(chan *ocelot.JobInstance, 0),
	}
}

func init() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetLevel(log.DebugLevel)
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {

	// Mock up test batches...
	for i := 0; i < 10; i++ {
		jobs[i] = newMock()
	}

	// Set Cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Producer && Serve
	p := pConfig.NewProducer(jobs)

	p.Serve(ctx)

}
