package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	ocelot "github.com/dmw2151/ocelot"
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

	go func() {
		// Listen for Incoming Jobs after Server is Up
		time.Sleep(time.Millisecond * 10)
		// Start Default Worker (uses: OCELOT_WORKER_CFG)
		var h ocelot.S3Handler = ocelot.S3Handler{Client: s3Client}
		wp, _ := ocelot.NewWorkerPool(h)

		wp.AcceptWork(Wctx, Wcancel)
	}()

	p, _ := ocelot.NewProducer()

	go func() {
		// Wait 5s Until Shutdown...
		time.Sleep(time.Millisecond * 5000)
		p.ShutDown(Pctx, Pcancel)
	}()

	// Start Default Producer && Serve (uses: OCELOT_SERVER_CFG)
	p.Serve(Pctx, Pcancel)

}
