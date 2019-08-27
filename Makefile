TARGET = remiro
PACKAGES := $(go list ./...)

.PHONY: lint-prepare
lint-prepare:
	@go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint
lint: lint-prepare
	@golangci-lint run \
		--enable=golint \
		--enable=gocyclo \
		--enable=goconst \
		--enable=uncovert \
		./...

.PHONY: test
test:
	@go test $(PACKAGES)

.PHONY: build
build:
	@go build -v -o $(TARGET) cmd/main.go

.PHONY: redis-up
redis-up:
	@docker run --name redis-source -p 6380:6379 --rm -d redis
	@docker run --name redis-destination -p 6381:6379 --rm -d redis

.PHONY: redis-down
redis-down:
	@docker stop redis-source
	@docker stop redis-destination
