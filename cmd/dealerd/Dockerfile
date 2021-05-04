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

RUN --mount=type=cache,target=/go/pkg/mod go mod download -x
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod go mod download -x &&\
  GOOS=linux make build-dealerd

FROM busybox:1-glibc
COPY --from=builder /lib/x86_64-linux-gnu/libdl-2.28.so /lib/libdl.so.2
COPY --from=builder /lib/x86_64-linux-gnu/libutil-2.28.so /lib/libutil.so.1 
COPY --from=builder /usr/lib/x86_64-linux-gnu/libOpenCL.so.1.0.0 /lib/libOpenCL.so.1
COPY --from=builder /usr/lib/gcc/x86_64-linux-gnu/8/libstdc++.so /lib/libstdc++.so.6
COPY --from=builder /lib/x86_64-linux-gnu/librt-2.28.so /lib/librt.so.1
COPY --from=builder /lib/x86_64-linux-gnu/libgcc_s.so.1 /lib/libgcc_s.so.1
COPY --from=builder /usr/lib/x86_64-linux-gnu/libhwloc.so.5 /lib/libhwloc.so.5
COPY --from=builder /usr/lib/x86_64-linux-gnu/libnuma.so.1 /lib/libnuma.so.1
COPY --from=builder /usr/lib/x86_64-linux-gnu/libltdl.so.7 /lib/libltdl.so.7

COPY --from=builder /app/dealerd /app/dealerd

WORKDIR /app 
ENTRYPOINT ["./dealerd"]
