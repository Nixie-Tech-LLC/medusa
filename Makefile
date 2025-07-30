.PHONY: all
all: rerun 

build:
	@docker compose down
	@docker compose build

run:
	@docker compose down
	@docker compose up --build

rebuild:
	@docker compose down
	@docker compose build --no-cache 

rerun: 
	@echo "tearing down containers"
	@docker compose down
	@echo "rebuilding docker image"
	@docker compose build --no-cache 
	@echo "running docker containers"
	@docker compose up

db: 
	@echo "tearing down containers and volumes" 
	@docker compose down -v 
	@echo "rebuilding docker image" 
	@docker compose build --no-cache 
	@echo "running docker containers" 
	@docker compose up
