.PHONY: build test vet docker run

build:
	go build -o bin/bosminer-exporter ./cmd/bosminer-exporter

test:
	go test ./...

vet:
	go vet ./...

docker:
	docker build -t bosminer-exporter .

run: build
	./bin/bosminer-exporter --miner.address $(MINER)
