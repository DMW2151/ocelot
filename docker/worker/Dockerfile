# docker build . --tag ocelot-worker

FROM golang:1.12

COPY ./ $GOPATH/github.com/dmw2151/ocelot

WORKDIR $GOPATH/github.com/dmw2151/ocelot

RUN go get . 

RUN go build \
    -o ./cmd/ocelot-grpc/worker/worker \
    ./cmd/ocelot-grpc/worker/ 
    
ENV OCELOT_LISTEN_ADDR=0.0.0.0:2151\    
    OCELOT_WORKER_CFG=./cmd/cfg/ocelot_worker_cfg.json

EXPOSE 2151

ENTRYPOINT ["./cmd/ocelot-grpc/worker/worker"]
