.DEFAULT_GOAL: all

all: up down

.PHONY: setup
setup:
	mkdir -p storage/

.PHONY: up
up: setup
	docker compose -f docker-compose.yaml up --build

.PHONY: down
down:
	docker compose -f docker-compose.yaml down

.PHONY: clean
clean:
	rm -rf storage/