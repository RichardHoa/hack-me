# syntax=docker/dockerfile:1

FROM golang:1.25 AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download
# Copy all source code
COPY . .
# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /main


# ==================================================================
# Doppler Stage - Only used for the 'prod' target
# This stage downloads the Doppler CLI into a temporary image.
# ==================================================================
FROM debian:bullseye-slim AS doppler-stage
RUN apt-get update && apt-get install -y apt-transport-https ca-certificates curl gnupg && \
    curl -sLf --retry 3 --tlsv1.2 --proto "=https" 'https://packages.doppler.com/public/cli/gpg.DE2A7741A397C129.key' | gpg --dearmor -o /usr/share/keyrings/doppler-archive-keyring.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/doppler-archive-keyring.gpg] https://packages.doppler.com/public/cli/deb/debian any-version main" | tee /etc/apt/sources.list.d/doppler-cli.list && \
    apt-get update && \
    apt-get -y install doppler


# ==================================================================
# Development Target - 'dev'
# This is the final image for development. It runs the app directly
# and is intended to be used with a .env file via Docker Compose.
# ==================================================================
FROM gcr.io/distroless/base-debian11 AS dev
WORKDIR /
# Copy only the compiled application binary
COPY --from=build-stage /main /main
USER nonroot:nonroot
# The entrypoint is the application itself, no Doppler.
ENTRYPOINT ["/main"]


FROM gcr.io/distroless/base-debian11 AS prod
WORKDIR /
# Copy the Doppler CLI from the doppler-stage
COPY --from=doppler-stage /usr/bin/doppler /usr/bin/doppler
# Copy the compiled application binary
COPY --from=build-stage /main /main
USER nonroot:nonroot
# The entrypoint uses Doppler to run the application
ENTRYPOINT ["/usr/bin/doppler", "run", "--"]
CMD ["/main"]
