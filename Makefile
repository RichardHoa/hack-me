
GOCMD=go
GOBUILD=$(GOCMD) build
GOFMT=$(GOCMD) fmt
GOVULNCHECK=govulncheck
BINARY_NAME=hack-me-backend
BINARY_UNIX=$(BINARY_NAME)

# Runs all checks and builds the application.
# Use: `make` or `make all`
all: check build

# Formats the code, checks for vulnerabilities.
# Use: `make check`
check: vet fmt vulncheck

# Builds the application binary.
# Use: `make build`
build:
	@echo "Building the application..."
	$(GOBUILD) -o $(BINARY_NAME) .
	@echo "Application built successfully: $(BINARY_NAME)"

# Use: `make fmt`
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Use: `make vulncheck`
vulncheck:
	@echo "Checking for vulnerabilities..."
	$(GOVULNCHECK) ./...

vet:
	@echo "Checking for suspicious constructs..."
	go vet ./...

# Use: `make clean`
clean:
	@echo "Cleaning up..."
	if [ -f $(BINARY_UNIX) ]; then rm $(BINARY_UNIX); fi
	@echo "Cleanup complete."

# Phony targets are not files.
.PHONY: all check build fmt vulncheck clean
