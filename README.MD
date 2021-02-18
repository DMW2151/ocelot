# Description

"Ocelot" is a framework for deploying jobs scheduled for short intervals across multiple worker nodes. The producer communicates to workers by serializing messages from a job producer to protobuf. Because of this, workers/handlers can be written in any language that supports protobuf. In effect, a personal attempt at something like Celery-beat.

## S.0 - TODO

1. Allow for multiple handlers in same worker group
2. Reduce API surface space, remove unnesc. exported funcs
3. Recycle original TCP tests -> GRPC tests
4. Create example in Python on worker side to prove protobuf works OK
5. Still doing dev on 1.12, Fastforward to 1.15


## S.1 - Examples

```golang
// Minimal Example of Producer Setup
// go run ./cmd/ocelot-grpc/client/rpcclient.go
package main

import (
    "golang.org/x/sync/semaphore"

    "github.com/dmw2151/ocelot"
    msg "github.com/dmw2151/ocelot/msg"
)

func main() {

    // Generate New Producer & Start Scheduled Jobs
    p, _ := ocelot.NewProducer()
    p.StartJobs()

    // Generate New Conn Pool w. Max Workers Slots == 5
    cP := msg.ConnPool{
        Sem:             semaphore.NewWeighted(5),
        OpenConnections: make([]msg.OcelotWorkerClient, 5),
    }

    // Register New Worker Address(es); Jobs directed via
    // GRPC Stream to workers
    msg.RegisterNewStreamingWorker(
        "127.0.0.1:2151", cP,
    )
}
```

```golang
// Start Single Worker (Acting as a GRPC Server)
// go run ./cmd/ocelot-grpc/worker/rpcworker.go
package main

import (
    // AWS/S3 Imports omitted for Brevity
    msg "github.com/dmw2151/ocelot/msg"
)

var (
    err error  //Placeholder Error

    // Define S3 Handler - Included as part of ocelot/msg but could be anything
    // that satisfies ocelot/msg Handler interface. Sample task polls a file's metadata
    // on a fixed interval...
    h = msg.S3Handler{
        Session: session.NewSession(
            &aws.Config{} // Omit Config for brevity
        ),
        Client:  s3.New(session.Must(s3Session, err)),
    }
)

func main() {
    // Start New WorkerPool. Blocks forerver, waiting to execute
    // the handler,
    wp, _ := msg.NewWorkerPool(h)
    wp.Serve()
}
```
