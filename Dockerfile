FROM --platform=linux/ppc64le  registry.access.redhat.com/ubi9/go-toolset:1.20.10-2.1699551725 as builder
ARG RUNTIME_IMAGE=quay.io/centos/centos:stream9-minimal
ARG TARGETOS
ARG TARGETARCH

#WORKDIR /workspace
WORKDIR         /opt/app-root/src
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY apis/ apis/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=ppc64le go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM --platform=linux/ppc64le quay.io/centos/centos:stream9-minimal
WORKDIR /
COPY --from=builder     /opt/app-root/src/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]