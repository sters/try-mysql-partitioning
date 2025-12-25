# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy everything including vendor directory
COPY . .

# Build the application using vendored dependencies
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app ./app/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app .

EXPOSE 8080

CMD ["./app"]
