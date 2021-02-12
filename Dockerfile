FROM golang:1.12

COPY ./ $GOPATH/github.com/dmw2151/ocelot

WORKDIR $GOPATH/github.com/dmw2151/ocelot

RUN go get . 

# DO YOU EVEN NEED TO BUILD??
RUN go build -o ./cmd/ocelot-server/server ./cmd/ocelot-server/  &&\
    go build -o ./cmd/ocelot-worker/worker ./cmd/ocelot-worker/ 

ENV OCELOT_HOST=127.0.0.1 \
    OCELOT_PORT=2151 \
    OCELOT_LISTEN_ADDR=0.0.0.0:2151

EXPOSE 2151

ENTRYPOINT ["./cmd/ocelot-server/server"]
