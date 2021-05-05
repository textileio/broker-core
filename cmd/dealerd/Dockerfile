# syntax = docker/dockerfile:1-experimental

FROM golang:1.16-buster as builder

RUN mkdir /app 
WORKDIR /app 

COPY go.mod go.sum ./
RUN go mod download -x
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
  GOOS=linux make build-dealerd

FROM busybox:1-glibc

COPY --from=builder /app/dealerd /app/dealerd

WORKDIR /app 
ENTRYPOINT ["./dealerd"]
