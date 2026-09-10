# Build the zombie-detector binary
FROM ghcr.io/cybozu/golang:1.26-noble as builder
ARG TARGETOS
ARG TARGETARCH
ARG GOPROXY

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
# the netrc secret carries Takumi Guard credentials when the build is run from CI;
# it is optional so that local builds without it still work
RUN --mount=type=secret,id=netrc,target=/root/.netrc \
    GOPROXY=${GOPROXY} go mod download

# Copy the go source
COPY cmd/ cmd/
COPY main.go main.go

# Build
# GOPROXY=off: all modules are already in the local cache from `go mod download` above,
# so this step must not need network access.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} GOPROXY=off go build -a -o zombie-detector main.go

FROM scratch
LABEL org.opencontainers.image.source https://github.com/cybozu-go/zombie-detector

WORKDIR /
COPY --from=builder /workspace/zombie-detector .

ENTRYPOINT ["/zombie-detector"]
