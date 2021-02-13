package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dmw2151/ocelot"
	handlers "github.com/dmw2151/ocelot/internal/handlers"
	utils "github.com/dmw2151/ocelot/internal/utils"
	"github.com/pkg/profile"
	log "github.com/sirupsen/logrus"
)

// Set Context and Cancel
var (
	ctx, cancel = context.WithCancel(context.Background())
)

var (
	s3Client, _ = session.NewSession(
		&aws.Config{
			Region:                        aws.String("us-east-1"),
			CredentialsChainVerboseErrors: aws.Bool(true),
			Credentials:                   credentials.NewEnvCredentials(),
		},
	)
	wp = &ocelot.WorkParams{
		HandlerType: "S3",
		NWorkers:    20,
		MaxBuffer:   10,
		Handler:     &handlers.S3Handler{Client: s3Client},
		Host:        os.Getenv("OCELOT_HOST"),
		Port:        os.Getenv("OCELOT_PORT"),
		DialTimeout: time.Duration(time.Millisecond * 1000),
	}
)

func init() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetLevel(log.DebugLevel)
}

func main() {

	defer profile.Start().Stop()
	defaultProducerConfig := utils.OpenYaml("./cmd/cfg/ocelot_server_cfg.yml")

	// Start Empty Producer && Serve
	p := defaultProducerConfig.NewProducer()

	go func() {
		time.Sleep(time.Millisecond * 100)
		// Listen for Incoming Jobs...
		ocelotWP, _ := wp.NewWorkerPool()
		ocelotWP.AcceptWork(ctx, cancel) // With a Timeout of 5s
		fmt.Println("Work's Done...")
	}()

	p.Serve(ctx)

}
