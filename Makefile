container_runtime := $(shell which podman || which docker)

$(info using ${container_runtime})

run:
	go run ./cmd/server

unit:
	go test -race -coverprofile cover.out \
		./cmd/... ./internal/...

test:
	make clean
	make up
	@echo wait cluster to start && sleep 5
	make run-tests; status=$$?; make clean; exit $$status
	@echo "test finished"

integration:
	make run-tests

run-tests:
	${container_runtime} run --rm --network=host wb-search-tests:latest

bench:
	go test -bench=. -benchmem ./internal/core

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/searchv1/search.proto

up:
	${container_runtime} compose up --build -d app nats

down:
	${container_runtime} compose down --remove-orphans

clean:
	${container_runtime} compose down --remove-orphans
	rm -rf .cache bin cover.out coverage.out *.test
