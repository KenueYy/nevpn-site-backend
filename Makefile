.PHONY: run test-rpc test-go test-all docker-up docker-down clean

run:
	docker compose up -d postgres

test-rpc:
	docker compose up -d postgres
	docker compose up -d api 
	docker compose up k6
	docker compose down 

test-go:
	docker compose up -d postgres_test
	docker compose run --rm tests
	docker compose down 

test-all:
	docker compose up -d postgres
	docker compose up -d api 
	docker compose up --exit-code-from k6 k6
	docker compose up -d postgres_test
	docker compose run --rm tests
	docker compose down -v

docker-up:
	docker compose up -d

docker-down:
	docker compose down -v

clean:
	go clean -cache