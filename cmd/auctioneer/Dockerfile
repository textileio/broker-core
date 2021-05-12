# syntax = docker/dockerfile:1-experimental

FROM golang:1.16-buster as builder

RUN mkdir /app 
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download -x
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
  BIN_BUILD_FLAGS="CGO_ENABLED=0 GOOS=linux" make build-auctioneer

FROM alpine
COPY --from=builder /app/auctioneer /app/auctioneer
WORKDIR /app

ENV AUCTIONEER_PATH /data/auctioneer
RUN mkdir -p $AUCTIONEER_PATH \
  && adduser -D -h $AUCTIONEER_PATH -u 1000 -G users auctioneer \
  && chown auctioneer:users $AUCTIONEER_PATH
USER auctioneer
VOLUME $AUCTIONEER_PATH

ENTRYPOINT ["./auctioneer"]
CMD ["daemon"]
