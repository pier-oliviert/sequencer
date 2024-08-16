# -- Common Build Base
# Build the manager binary
FROM golang:1.22 as build-base
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# -- Operator
FROM build-base as build-operator
# Copy the go source
COPY cmd/operator/main.go cmd/operator/main.go
COPY api/ api/
COPY internal/ internal/
COPY pkg/ pkg/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/operator/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot as operator
WORKDIR /
COPY --from=build-operator /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]

# -- Builder
FROM build-base as build-builder

# Copy the go source
COPY cmd/builder/main.go cmd/builder/main.go
COPY api/ api/
COPY internal/ internal/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o builder cmd/builder/main.go

FROM alpine:3 as builder

# Git is not required by the builder and run without it as the
# the builder is using a go native git implementation.
# However, buildx uses git, so as long as we use buildx, git is
# a hard dependency.
RUN apk update && apk upgrade --no-cache
RUN apk add --no-cache git

WORKDIR /
COPY --from=build-builder /workspace/builder .
COPY --from=docker/buildx-bin /buildx /usr/bin/buildx

# USER user:user
ENTRYPOINT ["/builder"]

# -- DNS
FROM build-base as build-dns

# Copy the go source
COPY cmd/dns/main.go cmd/dns/main.go
COPY api/ api/
COPY internal/ internal/
COPY pkg/ pkg/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o dns cmd/dns/main.go

FROM alpine:3 as dns

# Git is not required by the builder and run without it as the
# the builder is using a go native git implementation.
# However, buildx uses git, so as long as we use buildx, git is
# a hard dependency.
RUN apk update && apk upgrade --no-cache
RUN apk add --no-cache git

WORKDIR /
COPY --from=build-dns /workspace/dns .

# USER user:user
ENTRYPOINT ["/dns"]
