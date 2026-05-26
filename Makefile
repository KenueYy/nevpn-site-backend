.PHONY: up down build logs

# Запуск всего стека (фронт + бэк + postgres + redis)
up:
	docker compose up --build -d

down:
	docker compose down

build:
	docker compose build

logs:
	docker compose logs -f
