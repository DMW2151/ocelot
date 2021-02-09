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
	JobChannelBuffer: 5,
	ListenAddr:       os.Getenv("OCELOT_LISTEN_ADDR"),
	MaxConnections:   1,
}

var jobs = []*ocelot.Job{
	{
		ID:       uuid.New(),
		Interval: time.Millisecond * 1000,
		Path:     "https://hello.com/en/index.html",
	},
}

// Define Context
func main() {

	// Set Cancel...
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Producer
	p, _ := ocelot.NewProducer(pConfig, jobs)

	// Start Timers for each job Available in the Jobpool
	// on server start
	p.Serve(ctx)

}
