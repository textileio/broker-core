# syntax = docker/dockerfile:1-experimental

FROM golang:1.16-buster as builder

RUN mkdir /app 
WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download -x
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  BIN_BUILD_FLAGS="CGO_ENABLED=0 GOOS=linux" make build-brokerd

FROM alpine

# To initialize postgres and pass correct URI to the daemon when running the k8s pod for the first time.
RUN apk --no-cache add postgresql-client
COPY /bin/with-postgres.sh /app/

COPY --from=builder /app/brokerd /app/brokerd

WORKDIR /app
ENTRYPOINT ["./brokerd"]
