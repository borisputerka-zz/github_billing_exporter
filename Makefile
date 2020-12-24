export GO111MODULE=on

.PHONY: bin
bin: fmt vet
	go build -o bin/github_billing_exporter github.com/borisputerka/github_billing_exporter

.PHONY: fmt
fmt:
	go fmt .

.PHONY: vet
vet:
	go vet .

.PHONY: lint
lint:
	@golangci-lint run ./...
