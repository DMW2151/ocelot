// Package ocelot ...
package ocelot

import (
	"net"
	"os"
	"time"

	utils "github.com/dmw2151/ocelot/internal"
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
	ListenAddr string `json:"listen_addr"`

	// How long to wait on Producer to respond
	DialTimeout time.Duration `json:"dial_timeout"`
}

// NewWorkerPool - Factory for creating a WorkerPool object
// from a Config
func NewWorkerPool(h Handler) (*WorkerPool, error) {

	// Open the Config File...
	k := utils.ParseConfig(
		os.Getenv("OCELOT_WORKER_CFG"),
		&WorkParams{},
	)

	workCfg := k.(*WorkParams)

	// Init Connection & Listen for connections from
	l, err := net.Listen("tcp", workCfg.ListenAddr)

	// Difficult to Test, but this must be a Fatal Exit
	if err != nil {
		log.WithFields(
			log.Fields{
				"Error": err,
				"Addr":  workCfg.ListenAddr,
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
