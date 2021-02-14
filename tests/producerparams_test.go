package ocelot

import (
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func init() {
	// Set Logging
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetLevel(log.DebugLevel)
}

var jobs = []*Job{
	{
		ID:     uuid.New(),
		Tdelta: time.Millisecond * 1000,
		Path:   "https://hello.com/en/index.html",
		stgCh:  make(chan *JobInstance, 2),
	},
}

func TestProducerParams_newListener(t *testing.T) {

	t.Run("Valid Listen Addr Produces a New Listener", func(t *testing.T) {

		var pConfig = &ProducerConfig{
			JobChannelBuffer: 5,
			ListenAddr:       "127.0.0.1:8604",
			MaxConn:          1,
		}

		// Create a known connection & Extract Addr Struct
		l, _ := net.Listen("tcp", pConfig.ListenAddr)
		wantAddr := l.Addr().String()
		l.Close()

		// Compare Only Address of Network
		if got, _ := pConfig.newListener(); !reflect.DeepEqual(got.Addr().String(), wantAddr) {
			t.Errorf("ProducerConfig.newListener() = %v, want %v\n", got.Addr().String(), wantAddr)
		}
	})

	t.Run("Invalid Listen Addr Produces Error", func(t *testing.T) {

		var pConfig = &ProducerConfig{
			JobChannelBuffer: 5,
			ListenAddr:       "8.8.8.8:2151", // Nonsense Addr
			MaxConn:          1,
		}

		if _, err := pConfig.newListener(); !(reflect.TypeOf(err) == reflect.TypeOf(&net.OpError{})) {
			t.Errorf(
				"cfg.newListener() called with invalid Addr produces: %v wanted: %v",
				reflect.TypeOf(err), reflect.TypeOf(net.OpError{}),
			)
		}

	})

}

func TestProducerParams_NewProducer(t *testing.T) {

	var pConfig = &ProducerConfig{
		JobChannelBuffer: 5,
		ListenAddr:       "127.0.0.1:8604",
		MaxConn:          1,
	}

	p := pConfig.NewProducer()

	// Check Listen Address, TODO: Bad Test - Fix...
	if p.Listener.Addr().String() == "" {
		t.Error("Addr")
	}

	if !reflect.DeepEqual(p.JobPool.Jobs, jobs) {
		t.Error("Jobs")
	}

	// if !reflect.DeepEqual(cap(p.JobPool.JobChan), pConfig.JobChannelBuffer) {
	// 	t.Error("Chan Buffer")
	// }

	// Check Max Connections = 5 & Allocated; N Open == 0
	if (p.nConn != 0) || (pConfig.MaxConn != len(p.OpenConnections)) {
		t.Error("Connections")
	}

}
