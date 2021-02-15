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
	wctx, wcancel = context.WithCancel(context.Background())
	pctx, pcancel = context.WithCancel(context.Background())

	s3Client, _ = session.NewSession(
		&aws.Config{
			Region:                        aws.String("us-east-1"),
			CredentialsChainVerboseErrors: aws.Bool(true),
			Credentials:                   credentials.NewEnvCredentials(),
		},
	)
	h = ocelot.S3Handler{Client: s3Client}
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
		wp, _ := ocelot.NewWorkerPool(h)
		wp.AcceptWork(wctx, wcancel)
	}()

	p, _ := ocelot.NewProducer()

	go func() {
		// Wait 5s Until Shutdown...
		time.Sleep(time.Millisecond * 5000)
		p.ShutDown(pctx, pcancel)
	}()

	// Start Default Producer && Serve (uses: OCELOT_SERVER_CFG)
	p.Serve(pctx, pcancel)

}
