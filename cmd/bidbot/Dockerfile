# syntax = docker/dockerfile:1-experimental

FROM golang:1.16-buster as builder

RUN mkdir /app 
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download -x
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
  BIN_BUILD_FLAGS="CGO_ENABLED=0 GOOS=linux" make build-bidbot

FROM alpine
COPY --from=builder /app/bidbot /app/bidbot
WORKDIR /app

ENV REPO_PATH /data/bidbot
RUN mkdir -p $REPO_PATH \
  && adduser -D -h $REPO_PATH -u 1000 -G users bidbot \
  && chown bidbot:users $REPO_PATH
USER bidbot
VOLUME $REPO_PATH

ENTRYPOINT ["./bidbot"]
CMD ["--repo=/data/bidbot"]
