# syntax = docker/dockerfile:1-experimental

FROM golang:1.16-buster as builder

RUN mkdir /app 
WORKDIR /app 

COPY go.mod go.sum ./
RUN go mod download -x
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux make build-dealsd

FROM alpine
COPY --from=builder /app/dealsd /app/dealsd
WORKDIR /app 
ENTRYPOINT ["./dealsd"]
