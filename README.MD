# Desc

"Ocelot" is a (WIP) framework for deploying scheduled jobs across multiple worker nodes. Ocelot communicates to workers by serializing messages from a job scheduler/producer to protobuf. Because of this, workers/handlers can be written in any language that has protobuf support. In effect, a personal attempt at something like Celery-beat.

## Examples

Start a Producer (Acting as a GRPC client to Workers)

```golang
// Minimal Example of Worker Node Launch
package main

import (
	"github.com/dmw2151/ocelot"
)

func main() {
	// Where is your imported handler...
	wp, _ := ocelot.NewWorkerPool(h)
	wp.Serve()
}
```

Start Single Worker (Acting as a GRPC Server)

```golang

package main

import (
	"github.com/dmw2151/ocelot"
)

func main() {
	p, _ := ocelot.NewProducer()
	p.StartJobs()
	p.RegisterNewStreamingWorker("127.0.0.1:2151")
}
```
