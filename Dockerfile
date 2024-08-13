# Build the manager binary

FROM --platform=$BUILDPLATFORM golang:1.23 as base

ARG BUILDARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY config/ config/
COPY api/ api/
COPY controllers/ controllers/

FROM base as builder
ARG TARGETARCH

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GO111MODULE=on go build -a -o zfs-operator main.go

FROM ubuntu:22.04 as bin
USER root

RUN apt-get update && apt-get install zfsutils-linux curl -y

WORKDIR /
COPY --from=builder /workspace/zfs-operator .

EXPOSE 3260

ENTRYPOINT ["/zfs-operator"]
