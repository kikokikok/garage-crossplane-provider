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

# Use CMD instead of ENTRYPOINT to ensure compatibility with Crossplane
# Crossplane may override ENTRYPOINT but typically preserves CMD
CMD ["/usr/local/bin/provider"]
