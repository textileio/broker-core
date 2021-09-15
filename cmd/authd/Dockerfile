# syntax = docker/dockerfile:1-experimental

FROM golang:1.17-buster as builder

RUN mkdir /app 
WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download -x
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  BIN_BUILD_FLAGS="CGO_ENABLED=0 GOOS=linux" make build-authd

FROM alpine

RUN apk --no-cache add postgresql-client
COPY /bin/with-postgres.sh /app/

COPY --from=builder /app/authd /app/authd
WORKDIR /app
ENTRYPOINT ["./authd"]
