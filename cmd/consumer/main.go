package main

import (
	"context"
	"math/rand"
	ocelot "ocelot/pkg/ocelot"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	// Logging Params...
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true

	// For Testing Backoff/ HTTP Failures...
	rand.Seed(time.Now().UTC().UnixNano())
}

var wp = &ocelot.WorkParams{
	NWorkers:  5,
	MaxBuffer: 10,
	Func:      ocelot.HTTPSuccess,
	Host:      os.Getenv("OCELOT_HOST"),
	Port:      os.Getenv("OCELOT_PORT"),
}

func main() {

	var ctx, cancel = context.WithCancel(context.Background())

	// New Client
	ocelotWP, _ := ocelot.NewWorkerPool(wp)

	// Listen for Incoming Jobs...
	ocelotWP.AcceptWork(ctx, cancel)

}
