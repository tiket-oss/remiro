TARGET = remiro
COVERAGE_REPORT = coverage.txt

.PHONY: lint-prepare
lint-prepare:
	@echo "Installing golangci-lint"
	@GO111MODULE=off go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint
lint:
	@golangci-lint run \
		--enable=golint \
		--enable=gocyclo \
		--enable=goconst \
		--enable=unconvert \
		./...

.PHONY: test
test:
	@go test -v -race -coverprofile=$(COVERAGE_REPORT) -covermode atomic ./...

.PHONY: build
build:
	@go build -v -o $(TARGET)

.PHONY: redis-up
redis-up:
	@docker run --name redis-source -p 6380:6379 --rm -d redis
	@docker run --name redis-destination -p 6381:6379 --rm -d redis

.PHONY: redis-down
redis-down:
	@docker stop redis-source
	@docker stop redis-destination
