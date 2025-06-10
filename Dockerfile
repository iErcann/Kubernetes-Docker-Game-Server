# Start from the official Go image as the build stage
FROM golang:1.22-bullseye as builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum first (for layer caching)
COPY src/go.mod src/go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY src .

# Build the server binary
RUN go build -o game-server ./cmd/server

# Run the tests in the container
FROM builder AS run-test-stage
RUN go test -v ./...

# Deploy the application binary into a lean image
FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=builder /app/game-server /game-server

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/game-server"]
