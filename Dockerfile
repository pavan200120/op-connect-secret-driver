# Use a smaller base image for a smaller final image
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY main.go .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /op-connect-secret-driver .

# Use a minimal image for the final stage
FROM alpine:latest AS runner

# Add ca-certificates to trust custom CAs if needed
RUN mkdir -p "/run/docker/plugins"  \
    && apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY plugin/config.json .
COPY --from=builder /op-connect-secret-driver /usr/bin/op-connect-secret-driver

# Command to run the driver
CMD ["op-connect-secret-driver"]