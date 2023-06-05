# Build the zombie-detector binary
FROM quay.io/cybozu/golang:1.20-jammy as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY main.go main.go


# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o zombie-detector main.go

FROM scratch
LABEL org.opencontainers.image.source https://github.com/cybozu-go/zombie-detector

WORKDIR /
COPY --from=builder /workspace/zombie-detector .

ENTRYPOINT ["/zombie-detector"]
