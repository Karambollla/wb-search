run:
	go run ./cmd/server

unit:
	go test -v -count=1 ./api/... ./cmd/... ./internal/...

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./internal/core

proto:
	PATH="$$(go env GOPATH)/bin:$$PATH" protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/searchv1/search.proto

up:
	docker compose up --build

down:
	docker compose down --remove-orphans

clean:
	rm -rf .cache bin coverage.out *.test
