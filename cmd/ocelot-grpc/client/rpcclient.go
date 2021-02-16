package main

import (
	"github.com/dmw2151/ocelot"
	log "github.com/sirupsen/logrus"
)

func init() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetLevel(log.DebugLevel)
}

func main() {

	p, _ := ocelot.NewProducer()
	p.StartJobs()
	p.RegisterNewStreamingWorker("127.0.0.1:2151")

}
