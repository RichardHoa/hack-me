# syntax=docker/dockerfile:1

FROM golang:1.24 AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /main

FROM debian:bullseye-slim AS doppler-stage
RUN apt-get update && apt-get install -y apt-transport-https ca-certificates curl gnupg && \
    curl -sLf --retry 3 --tlsv1.2 --proto "=https" 'https://packages.doppler.com/public/cli/gpg.DE2A7741A397C129.key' | gpg --dearmor -o /usr/share/keyrings/doppler-archive-keyring.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/doppler-archive-keyring.gpg] https://packages.doppler.com/public/cli/deb/debian any-version main" | tee /etc/apt/sources.list.d/doppler-cli.list && \
    apt-get update && \
    apt-get -y install doppler

FROM gcr.io/distroless/base-debian11 AS build-release-stage
WORKDIR /

COPY --from=doppler-stage /usr/bin/doppler /usr/bin/doppler
COPY --from=build-stage /main /main

USER nonroot:nonroot

ENTRYPOINT ["/usr/bin/doppler", "run", "--"]

CMD ["/main"]
