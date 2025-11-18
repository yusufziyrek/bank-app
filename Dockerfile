# syntax=docker/dockerfile:1

## Stage 1: Build the Go binary
FROM golang:1.25.4 AS builder

WORKDIR /app

# Download dependencies first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Run unit tests (set RUN_TESTS=true to enable during docker build)
ARG RUN_TESTS=false
RUN if [ "$RUN_TESTS" = "true" ]; then \
        go test ./... ; \
    else \
        echo "Skipping tests" ; \
    fi

# Build the statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/bank-app ./cmd

## Stage 2: Create the runtime image
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy CA certificates (required for outbound TLS calls, e.g. to PostgreSQL with SSL)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled binary
COPY --from=builder /app/bin/bank-app /usr/local/bin/bank-app

# Application defaults (override via environment variables or secrets manager)
ENV APP_PORT=8080 \
    APP_ENV=production \
    PG_HOST=postgres \
    PG_PORT=5432 \
    PG_USER=postgres \
    PG_PASS=postgres \
    PG_DB=bankapp \
    CARD_ENCRYPTION_KEY=change-me \
    JWT_SECRET=change-me \
    JWT_TTL=60

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/bank-app"]
