# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy the Go module file and the source code
COPY go.mod .
COPY main.go .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-caller .

# Final stage
FROM alpine:latest

WORKDIR /root/

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/api-caller .

# Command to run the executable
CMD ["./api-caller"]