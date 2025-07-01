.PHONY: all
all: rerun 

build:
	@docker compose down
	@docker compose build

run:
	@docker compose down
	@docker compose up --build -d 

rebuild:
	@docker compose down
	@docker compose build --no-cache 

rerun: 
	@echo "tearing down containers"
	@docker compose down
	@echo "rebuilding docker image"
	@docker compose build --no-cache 
	@echo "running docker containers"
	@docker compose up -d


