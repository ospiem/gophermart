#.DEFAULT_GOAL := test

export GOLANGCI_LINT_CACHE=${PWD}/golangci-lint/.cache
export SHELL=/bin/zsh

.PHONY: build
build:
	@go build -o ./cmd/gophermart/gophermart ./cmd/gophermart/*.go

.PHONY: run
run: build
	@./cmd/gophermart/gophermart -d "postgres://gopher:supersecretpass@localhost:5432/gophermart?sslmode=disable" -a localhost:8080 -r http://localhost:8081

.PHONY: debug
debug: build
	@./cmd/gophermart/gophermart -d "postgres://gopher:supersecretpass@localhost:5432/gophermart?sslmode=disable" -l debug -a localhost:8080 -r http://localhost:8081

.PHONY: postgres
postgres:
	@docker compose up -d postgres

.PHONY: lint
lint: _golangci-lint-rm-unformatted-report

.PHONY: _golangci-lint-reports-mkdir
_golangci-lint-reports-mkdir:
	mkdir -p ./golangci-lint

.PHONY: _golangci-lint-run
_golangci-lint-run: _golangci-lint-reports-mkdir
	-docker run --rm \
    -v $(shell pwd):/app \
    -v $(GOLANGCI_LINT_CACHE):/root/.cache \
    -w /app \
    golangci/golangci-lint:v1.55.2 \
        golangci-lint run \
            -c .golangci.yml \
	> ./golangci-lint/report-unformatted.json

.PHONY: _golangci-lint-format-report
_golangci-lint-format-report: _golangci-lint-run
	cat ./golangci-lint/report-unformatted.json | jq > ./golangci-lint/report.json

.PHONY: _golangci-lint-rm-unformatted-report
_golangci-lint-rm-unformatted-report: _golangci-lint-format-report
	rm ./golangci-lint/report-unformatted.json

.PHONY: golangci-lint-clean
golangci-lint-clean:
	sudo rm -rf ./golangci-lint matted.json > ./golangci-lint/report.json

.PHONY:
truncate:
	@docker exec postgres psql -U mcollector -d metrics -c 'truncate table counters, gauges;'ncate table counters, gauges;'