// Package ocelot ...
package ocelot

import (
	"fmt"
	"net"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// WorkParams is a factory struct for WorkerPools
type WorkParams struct {
	// Name of Handler...
	HandlerType string `json:"handler_type"`

	// Number of goroutines to launch taking work
	NWorkers int `json:"n_workers"`

	// Job Channel, buffer to in Chan between decoding work
	// and the worker
	MaxBuffer int `json:"max_buffer"`

	// IP and Host of Producer
	Host string `json:"host"`
	Port string `json:"port"`

	// How long to wait on Producer to respond
	DialTimeout time.Duration `json:"dial_timeout"`
}

// NewWorkerPool - Factory for creating a WorkerPool object
// from a Config
func NewWorkerPool(h Handler) (*WorkerPool, error) {

	// Open the Config File...
	k := parseConfig(
		os.Getenv("OCELOT_WORKER_CFG"),
		&WorkParams{},
	)

	workCfg := k.(*WorkParams)

	// Init Connection
	l, err := net.Listen(
		"tcp",
		fmt.Sprintf("%s:%s", workCfg.Host, workCfg.Port),
	)

	// Difficult to Test, but this MUST be a Fatal Exit
	if err != nil {
		log.WithFields(
			log.Fields{
				"Error": err,
				"Host":  workCfg.Host,
				"Port":  workCfg.Port,
			},
		).Fatal("Could Not Listen on Addr")
	}

	return &WorkerPool{
		Listener: l,
		Params:   workCfg,
		Handler:  h,
		Pending:  make(chan *JobInstanceMsg),
		Results:  make(chan *JobInstanceMsg),
	}, nil
}
