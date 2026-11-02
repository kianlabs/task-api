.DEFAULT_GOAL := help

# Konfigurasi default
DATABASE_URL ?= postgres://taskapi:devpassword@127.0.0.1:5432/taskdb?sslmode=disable
TEST_DATABASE_URL ?= postgres://taskapi:devpassword@127.0.0.1:5432/taskdb_test?sslmode=disable

.PHONY: help run build test test-integration vet fmt lint migrate docker-up docker-down smoke

help:
	@echo "task-api development commands:"
	@echo "  make run              Jalankan server aplikasi secara lokal"
	@echo "  make build            Compile binary ke bin/api"
	@echo "  make test             Jalankan unit tests (race detector aktif)"
	@echo "  make test-integration Jalankan integration tests dengan TEST_DATABASE_URL"
	@echo "  make vet              Jalankan go vet ./..."
	@echo "  make fmt              Format kode dengan gofmt -w ."
	@echo "  make lint             Periksa formatting (gofmt -l) dan jalankan go vet"
	@echo "  make migrate          Jalankan semua file SQL di migrations/ berurutan"
	@echo "  make docker-up        Bangun & jalankan container via docker compose"
	@echo "  make docker-down      Hentikan container & bersihkan volume"
	@echo "  make smoke            Jalankan smoke test endpoint ke server aktif"

run:
	go run .

build:
	@mkdir -p bin
	go build -v -o bin/api .

test:
	go test -v -race ./...

test-integration:
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)
	go vet ./...

migrate:
	@for file in $$(ls -1 migrations/*.sql | sort); do \
		echo "Applying $$file..."; \
		psql "$(DATABASE_URL)" -f "$$file" || exit 1; \
	done

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down -v

smoke:
	bash scripts/smoke.sh
