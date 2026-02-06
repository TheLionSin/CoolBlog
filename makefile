# Переменная с именами файлов, чтобы не писать их каждый раз
COMPOSE_FILES=-f docker-compose.kafka.yml -f docker-compose.app.yml

# Собрать образы (аналог docker build)
build:
	docker compose $(COMPOSE_FILES) build

# Запустить всё в фоне (аналог docker up -d)
up:
	docker compose $(COMPOSE_FILES) up -d

# Остановить и удалить контейнеры
down:
	docker compose $(COMPOSE_FILES) down

# Посмотреть логи (все сразу)
logs:
	docker compose $(COMPOSE_FILES) logs -f

# Посмотреть логи только конкретного сервиса (например, make logs-app service=consumer)
logs-app:
	docker compose $(COMPOSE_FILES) logs -f $(service)

# Перезапустить конкретный сервис (например, make restart service=consumer)
restart:
	docker compose $(COMPOSE_FILES) restart $(service)