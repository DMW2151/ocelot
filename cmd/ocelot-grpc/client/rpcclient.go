package main

import (
	"github.com/dmw2151/ocelot"
	msg "github.com/dmw2151/ocelot/msg"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

func init() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetLevel(log.DebugLevel)
}

func main() {

	// Generate New Producer...
	p, _ := ocelot.NewProducer()
	p.StartJobs()

	// Generate New Conn Pool w. Max Workers == 5
	// TODO: VSCode Complains on this line...
	cP := msg.ConnPool{
		Sem:             semaphore.NewWeighted(5),
		OpenConnections: make([]msg.OcelotWorkerClient, 5),
	}

	// Register New Worker Address
	msg.RegisterNewStreamingWorker(
		"127.0.0.1:2151", cP,
	)

}
