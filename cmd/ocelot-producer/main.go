package main

import (
	"context"
	"math/rand"
	"os"
	"time"

	"github.com/dmw2151/ocelot"
	log "github.com/sirupsen/logrus"
)

// Set Context and Cancel
var (
	ctx, cancel           = context.WithCancel(context.Background())
	defaultProducerConfig = &ocelot.ProducerConfig{
		JobChannelBuffer: 0,
		ListenAddr:       os.Getenv("OCELOT_LISTEN_ADDR"), // Guesses unless Env Set
		MaxConnections:   2,
	}
)

func init() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetLevel(log.DebugLevel)
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	defer cancel()

	// Start Empty Producer && Serve
	p := defaultProducerConfig.NewProducer([]*ocelot.Job{})
	p.Serve(ctx)
}
