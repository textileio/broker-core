# syntax = docker/dockerfile:1-experimental

FROM golang:1.16-buster as builder

RUN apt-get update && apt-get install -y mesa-opencl-icd clang ocl-icd-opencl-dev jq hwloc libhwloc-dev

WORKDIR /tmp
RUN curl https://sh.rustup.rs -sSf > rustup.sh
RUN chmod 755 rustup.sh
RUN ./rustup.sh -y
RUN rm /tmp/rustup.sh
ENV PATH="/root/.cargo/bin:${PATH}"

RUN mkdir /app 
WORKDIR /app 

COPY . .
RUN mkdir -p extern/filecoin-ffi
RUN make build-ffi

RUN go mod download -x
RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux make build-dealerd

FROM alpine
COPY --from=builder /app/dealerd /app/dealerd
WORKDIR /app 
ENTRYPOINT ["./dealerd"]
