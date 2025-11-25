# Build stage
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with automatic platform detection
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -o provider cmd/provider/main.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /workspace/provider /usr/local/bin/provider

USER 65532:65532

# Use ENTRYPOINT to ensure container always has a command
# This is required for Crossplane provider installation via Provider CRD
# as Crossplane may set empty command/args arrays in the pod spec
ENTRYPOINT ["/usr/local/bin/provider"]
