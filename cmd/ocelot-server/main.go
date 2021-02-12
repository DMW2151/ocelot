package main

import (
	"context"
	"flag"

	utils "github.com/dmw2151/ocelot/internal/utils"
	log "github.com/sirupsen/logrus"
)

// Set Context and Cancel
var (
	ctx, cancel = context.WithCancel(context.Background())
)

func init() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetLevel(log.InfoLevel)
}

func main() {
	textPtr := flag.String("f", "./cmd/cfg/ocelot_server_cfg.yml", "Text to parse. (Required)")
	flag.Parse()

	defaultProducerConfig := utils.OpenYaml(*textPtr)

	defer cancel()

	// Start Empty Producer && Serve
	p := defaultProducerConfig.NewProducer()
	p.Serve(ctx)
}
