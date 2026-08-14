.PHONY: gen migrate-up migrate-down swagger build run test vet

gen:
	sqlc generate

migrate-up:
	migrate -path migrations -database "mysql://root:root@tcp(localhost:3306)/task" up

migrate-down:
	migrate -path migrations -database "mysql://root:root@tcp(localhost:3306)/task" down 1

swagger:
	swag init -g cmd/server/main.go -o docs

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

vet:
	go vet ./...
