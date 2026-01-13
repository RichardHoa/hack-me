GOCMD=go
GOBUILD=$(GOCMD) build
GOFMT=$(GOCMD) fmt
GOVULNCHECK=govulncheck

prod: check build

check: update vet fmt vulncheck gosec syft_and_grype tidy

up:
	docker compose up --build -d

down:
	docker compose down -v

run: 
	doppler run -- go run main.go

air:
	doppler run -- air

test:
	doppler run -- go test ./...

test-debug:
	doppler run -- go test ./... -v -failfast

cpu:
	doppler run -- go test -bench . -cpuprofile cpu.prof ./internal/api_test

cpu_server:
	go tool pprof -http=:8080 cpu.prof

trace:
	doppler run -- go test -trace trace.out ./internal/api_test

trace_server:
	go tool trace trace.out
	
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
	grype db update
	syft ./ -o cyclonedx-json | grype


# Phony targets are not files.
.PHONY: all check build fmt vulncheck clean
