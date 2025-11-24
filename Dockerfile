# syntax=docker/dockerfile:1

# Build the provider binary
FROM golang:1.21-alpine AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o provider cmd/provider/main.go

# Use distroless as minimal runtime image
FROM gcr.io/distroless/static:nonroot

WORKDIR /

# Copy the binary from builder
COPY --from=builder /workspace/provider /usr/local/bin/provider

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/provider"]
