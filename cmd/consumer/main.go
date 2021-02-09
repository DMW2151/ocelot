package main

import (
	"context"
	"math/rand"
	ocelot "ocelot/pkg/server"
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

func main() {

	var ctx, cancel = context.WithCancel(context.Background())

	// New Client
	ocelotWP, _ := ocelot.NewWorkerPool()

	// Start Workers...
	// TODO; Better Naming here....
	ocelotWP.Params.StartWorkers()

	// Listen for Incoming Jobs...
	ocelotWP.AcceptWork(ctx, cancel)

}
