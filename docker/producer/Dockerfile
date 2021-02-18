# docker build . --tag ocelot-producer

FROM golang:1.12

COPY ./ $GOPATH/github.com/dmw2151/ocelot

WORKDIR $GOPATH/github.com/dmw2151/ocelot

RUN go get . 

# Build Producer -> GRPC CMD Folder
RUN go build \
    -o ./cmd/ocelot-grpc/producer/producer \
    ./cmd/ocelot-grpc/producer/ 

# Producer needs to dial 
# Where ${OCELOT_HOST}:${OCELOT_PORT} is the listen address
ENV OCELOT_HOST=127.0.0.1\
    OCELOT_PORT=2151\
    OCELOT_PRODUCER_CFG=./cmd/cfg/ocelot_producer_cfg.json

EXPOSE 2151

ENTRYPOINT ["./cmd/ocelot-grpc/producer/producer"]
