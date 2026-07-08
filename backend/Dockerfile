# Stage 1: Build
FROM golang:1.20-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy dependency files
COPY go.mod ./
# (Copy go.sum if it exists, otherwise it will be generated during tidy)
# COPY go.sum ./

# Copy source code
COPY . .

# Run mod tidy to fetch dependencies
RUN go mod tidy

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go

# Stage 2: Runtime
FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary
COPY --from=builder /app/main .

# Create logs directory
RUN mkdir -p /var/log/sandbox && chmod 777 /var/log/sandbox

# Expose backend port
EXPOSE 8080

# Command to run
CMD ["./main"]
