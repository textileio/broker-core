# syntax = docker/dockerfile:1-experimental

FROM golang:1.17-buster as builder

RUN mkdir /app 
WORKDIR /app 

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download -x
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  GOOS=linux make build-dealerd

FROM alpine

RUN apk --no-cache add postgresql-client

COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/lib/*-linux-gnu*/libssl.so* /usr/lib/
COPY --from=builder /usr/lib/*-linux-gnu*/libcrypto.so* /usr/lib/

COPY /bin/with-postgres.sh /app/
COPY --from=builder /app/dealerd /app/dealerd

WORKDIR /app 
ENTRYPOINT ["./dealerd"]
