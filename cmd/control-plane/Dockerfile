# 1: Builder
FROM golang:1.26-alpine AS builder
WORKDIR /app

# Copy dependency files from the root
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the binary
RUN go build -o control-plane ./cmd/control-plane/main.go

# 2: Final Image
FROM alpine:latest

WORKDIR /root/

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/control-plane .

# Expose the gRPC port
EXPOSE 50051

# Run the binary
CMD ["./control-plane"]