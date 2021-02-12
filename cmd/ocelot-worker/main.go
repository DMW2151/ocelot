package main

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	ocelot "github.com/dmw2151/ocelot"
	handlers "github.com/dmw2151/ocelot/internal/handlers"
	log "github.com/sirupsen/logrus"
)

func init() {
	// Logging Params...
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetLevel(log.InfoLevel)
}

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

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// New Client
	ocelotWP, _ := wp.NewWorkerPool()

	// Listen for Incoming Jobs...
	ocelotWP.AcceptWork(ctx, cancel)

}
