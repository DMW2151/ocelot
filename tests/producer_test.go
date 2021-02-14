// Package ocelot ...
package ocelot

import (
	"context"
	"net"
	"testing"
	"time"
)

// Statics...
var pConfig = &ProducerConfig{
	JobChannelBuffer: 5,
	ListenAddr:       "127.0.0.1:2152",
	MaxConn:          1,
}

// Globals for Testing; reduce boilerplate below...
func TestProducer_ShutDown(t *testing.T) {

	// Check that server actually Shutsdown...
	t.Run("Check Shutdown Closes Server", func(t *testing.T) {

		var (
			ctx, cancel = context.WithCancel(context.Background())
			done        = make(chan bool, 1)
			timeout     = time.NewTimer(time.Millisecond * 1200)
		)

		p := pConfig.NewProducer()

		go func() {
			time.Sleep(1 * time.Second)
			p.ShutDown(ctx, cancel)
			done <- true
		}()

		go p.Serve(ctx)

		select {
		case <-timeout.C:
			t.Fatal("Test didn't finish in time")
		case <-done:
			return
		}
	})

	t.Run("Check Shutdown Closes All Connections", func(t *testing.T) {

		p := pConfig.NewProducer()

		var (
			ctx, cancel = context.WithCancel(context.Background())
		)

		// Start Serving
		go p.Serve(ctx)
		time.Sleep(100 * time.Millisecond)

		// Add connections.
		_, _ = net.Dial("tcp", "127.0.0.1:2152")

		p.ShutDown(ctx, cancel)
		time.Sleep(100 * time.Millisecond)

		if (len(p.OpenConnections) == p.nConn) && (p.nConn == 0) {
			t.Error(
				"Calling Job.Serve and Connecting 1 Client Returns...", // TODO: Message
				len(p.OpenConnections), 1,
			) // Go Vet Will Complain if Return placed here...
		}
	})
}
