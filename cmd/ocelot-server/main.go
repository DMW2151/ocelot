package main

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dmw2151/ocelot"
	handlers "github.com/dmw2151/ocelot/internal/handlers"
	"github.com/pkg/profile"
	log "github.com/sirupsen/logrus"
)

// Set Context and Cancel
var (
	Wctx, Wcancel = context.WithCancel(context.Background())
	Pctx, Pcancel = context.WithCancel(context.Background())

	s3Client, _ = session.NewSession(
		&aws.Config{
			Region:                        aws.String("us-east-1"),
			CredentialsChainVerboseErrors: aws.Bool(true),
			Credentials:                   credentials.NewEnvCredentials(),
		},
	)

	wp = &ocelot.WorkParams{
		HandlerType: "S3",
		NWorkers:    2,
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

	// Start Empty Producer && Serve
	p := ocelot.NewProducer("./cmd/cfg/ocelot_server_cfg.yml")

	go func() {
		time.Sleep(time.Millisecond * 100)
		// Listen for Incoming Jobs...
		ocelotWP, _ := wp.NewWorkerPool()

		// Call for End of Worker (set to 5s)
		ocelotWP.AcceptWork(Wctx, Wcancel)

		// Call for End of Producer 5s later
		//time.Sleep(time.Millisecond * 5000)

		// ocelotWPV2, _ := wp.NewWorkerPool()
		// ocelotWPV2.AcceptWork(Wctx2, Wcancel2)
		// log.Error("Worker's Done p2")
	}()

	go func() {
		time.Sleep(time.Millisecond * 5000)
		p.ShutDown(Pctx, Pcancel)
	}()

	p.Serve(Pctx, Pcancel)

}
