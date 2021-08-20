# syntax = docker/dockerfile:1-experimental

FROM golang:1.16-buster as builder

RUN mkdir /app 
WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download -x
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  BIN_BUILD_FLAGS="CGO_ENABLED=0 GOOS=linux" make build-auctioneerd

FROM alpine
RUN apk --no-cache add postgresql-client
COPY /bin/with-postgres.sh /app/

COPY --from=builder /app/auctioneerd /app/auctioneerd
COPY --from=builder /app/cmd/auctioneerd/bin/container_daemon /app/start_auctioneer
WORKDIR /app

ENV AUCTIONEER_PATH /data/auctioneerd
RUN mkdir -p $AUCTIONEER_PATH \
  && adduser -D -h $AUCTIONEER_PATH -u 1000 -G users auctioneerd \
  && chown auctioneerd:users $AUCTIONEER_PATH
USER auctioneerd
VOLUME $AUCTIONEER_PATH

ENTRYPOINT ["./start_auctioneer"]
CMD ["daemon"]
