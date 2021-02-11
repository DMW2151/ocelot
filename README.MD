# Ocelot

## Examples

### Sample Worker Setup

```golang
package main

// Imports Omitted for Brevity ...

var (
    wp = &ocelot.WorkParams{
        NWorkers:  5,
        MaxBuffer: 10,
        Func:      ocelot.HTTPSuccess,
        Host:      os.Getenv("OCELOT_HOST"),
        Port:      os.Getenv("OCELOT_PORT"),
    }
    ctx, cancel = context.WithCancel(context.Background())
)

func main() {

    // New Client
    ocelotWP, _ := ocelot.NewWorkerPool(wp)

    // Listen for Incoming Jobs...
    ocelotWP.AcceptWork(ctx, cancel)
}
```

### Sample Producer Setup

```golang
package main

// Imports Omitted for Brevity ...

// Producer Config
var pConfig = &ocelot.ProducerConfig{
    JobChannelBuffer: 5,
    MaxConnections:   5,
    ListenAddr:       os.Getenv("OCELOT_LISTEN_ADDR"),
}

// Jobs Template
var jobs = []*ocelot.Job{
    {
        ID:       uuid.New(),
        Interval: time.Millisecond * 300,
        Path:     "https://path/to/resource/",
    },
}

func main() {

    // Define Context && Set Cancel...
    ctx, cancel := context.WithCancel(context.Background())

    // Start Producer
    p, _ := ocelot.NewProducer(pConfig, jobs)

    // Start Timers for each job Available in the Jobpool
    // on server start
    p.Serve(ctx)
}
```

### Testing

```bash
go test -coverprofile=./testcov/c.out ./pkg/ocelot/ && go tool cover -html=./testcov/c.out 
```

### changelog

- 02-09-2021 - Implementing a method to start and stop scheduled jobs user should be able to call a method that puts/removes a job...
