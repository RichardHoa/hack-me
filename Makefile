GOCMD=go
GOBUILD=$(GOCMD) build
GOFMT=$(GOCMD) fmt
GOVULNCHECK=govulncheck

prod: check build

check: update tidy vet fmt vulncheck gosec syft_and_grype

up:
	docker compose up --build -d

down:
	docker compose down -v

run: 
	doppler run -- go run main.go

air:
	doppler run -- air

test:
	doppler run -- go test ./... -v -failfast



# Builds the application binary.
# Use: `make build`
build:
	@echo "Building the application..."
	sudo docker buildx build --platform linux/amd64 -t hack-me/backend . --target=prod

# Use: `make fmt`
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

gosec:
	@echo "Scanning code for security"
	gosec ./...

# Use: `make vulncheck`
vulncheck:
	@echo "Checking for vulnerabilities..."
	$(GOVULNCHECK) ./...

update:
	@echo "Checking for update"
	go get -u ./...

tidy:
	@echo "Tidy"
	go mod tidy

vet:
	@echo "Checking for suspicious constructs..."
	go vet ./...

syft_and_grype:
	@echo "Checking SBOMS and potential vulnerabilities..."
	syft ./ -o cyclonedx-json | grype


# Phony targets are not files.
.PHONY: all check build fmt vulncheck clean
