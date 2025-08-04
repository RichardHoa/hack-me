GOCMD=go
GOBUILD=$(GOCMD) build
GOFMT=$(GOCMD) fmt
GOVULNCHECK=govulncheck

prod: check build

check: update tidy vet fmt vulncheck

up:
	docker compose up --build -d

down:
	docker compose down -v

run: 
	doppler run -- go run main.go

air:
	doppler run -- air

# Builds the application binary.
# Use: `make build`
build:
	@echo "Delete all previous images in ECR"
	@IDS_TAGGED=$$(aws ecr list-images --repository-name hack-me/backend --region ap-southeast-1 --filter "tagStatus=TAGGED" --query 'imageIds[*]' --output json); \
	if [ "$$IDS_TAGGED" != "[]" ] && [ "$$IDS_TAGGED" != "null" ]; then \
		echo "Deleting tagged images..."; \
		aws ecr batch-delete-image \
			--repository-name hack-me/backend \
			--region ap-southeast-1 \
			--image-ids "$$IDS_TAGGED"; \
	else \
		echo "No tagged images to delete."; \
	fi

	@IDS_UNTAGGED=$$(aws ecr list-images --repository-name hack-me/backend --region ap-southeast-1 --filter "tagStatus=UNTAGGED" --query 'imageIds[*]' --output json); \
	if [ "$$IDS_UNTAGGED" != "[]" ] && [ "$$IDS_UNTAGGED" != "null" ]; then \
		echo "Deleting untagged images..."; \
		aws ecr batch-delete-image \
			--repository-name hack-me/backend \
			--region ap-southeast-1 \
			--image-ids "$$IDS_UNTAGGED"; \
	else \
		echo "No untagged images to delete."; \
	fi

	@echo "Building the application..."
	docker buildx build --platform linux/amd64 -t hack-me/backend . --target=prod

	@echo "Tag the build image"
	docker tag hack-me/backend:latest 004843574486.dkr.ecr.ap-southeast-1.amazonaws.com/hack-me/backend:latest

	@echo "Push the image into ECR"
	docker push 004843574486.dkr.ecr.ap-southeast-1.amazonaws.com/hack-me/backend:latest

# Use: `make fmt`
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

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


# Phony targets are not files.
.PHONY: all check build fmt vulncheck clean
