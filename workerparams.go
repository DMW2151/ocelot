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
	HandlerType string `yaml:"handler_type"`

	// Number of goroutines to launch taking work
	NWorkers int `yaml:"n_workers"`

	// Job Channel, buffer to in Chan between decoding work
	// and the worker
	MaxBuffer int `yaml:"max_buffer"`

	// IP and Host of Producer
	Host string `yaml:"host"`
	Port string `yaml:"port"`

	// How long to wait on Producer to respond
	DialTimeout time.Duration `yaml:"dial_timeout"`
}

// NewWorkerPool - Factory for creating a WorkerPool object
// from a Config
func NewWorkerPool(h JobHandler) (*WorkerPool, error) {

	// Open the Config File...
	k := parseConfig(
		os.Getenv("OCELOT_WORKER_CFG"),
		&WorkParams{},
	)

	workCfg := k.(*WorkParams)

	// Init Connection
	c, err := net.DialTimeout(
		"tcp",
		fmt.Sprintf("%s:%s", workCfg.Host, workCfg.Port),
		workCfg.DialTimeout,
	)

	// Difficult to Test, but this MUST be a Fatal Exit
	if err != nil {
		log.WithFields(
			log.Fields{"Error": err},
		).Fatal("Failed to Dial Producer")
	}

	log.WithFields(
		log.Fields{
			"Local Addr":  c.LocalAddr().String(),
			"Remote Addr": c.RemoteAddr().String()},
	).Info("New WorkerPool")

	return &WorkerPool{
		Connection: c,
		Params:     workCfg,
		Handler:    h,
		Pending:    make(chan JobInstance, workCfg.MaxBuffer),
	}, nil
}
