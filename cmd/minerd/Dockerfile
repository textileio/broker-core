# syntax = docker/dockerfile:1-experimental

FROM golang:1.16-buster as builder

RUN mkdir /app 
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download -x
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux make build-minerd

FROM alpine
COPY --from=builder /app/minerd /app/minerd
WORKDIR /app

ENV REPO_PATH /data/minerd
RUN mkdir -p $REPO_PATH \
  && adduser -D -h $REPO_PATH -u 1000 -G users minerd \
  && chown minerd:users $REPO_PATH
USER minerd
VOLUME $REPO_PATH

ENTRYPOINT ["./minerd"]
CMD ["--repo=/data/minerd"]
