BINARY         := server
CMD_DIR        := cmd/server
IMAGE_NAME     := medusa:local
CONTAINER_NAME := medusa
ENV_FILE       := .env
PORT           := 8080

.PHONY: all
all: build

.PHONY: build
build:
	@go mod tidy
	@go build -o $(BINARY) $(CMD_DIR)

.PHONY: run
run: build
	@./$(BINARY)

.PHONY: clean
clean:
	@rm -f $(BINARY)

.PHONY: docker-build
docker-build:
	@docker build -t $(IMAGE_NAME) .

.PHONY: docker-stop
docker-stop:
	-@docker stop $(CONTAINER_NAME) 2>/dev/null || true

.PHONY: docker-rm
docker-rm:
	-@docker rm $(CONTAINER_NAME) 2>/dev/null || true

.PHONY: docker-run
docker-run: docker-build docker-stop docker-rm
	docker run --network host \
		--name $(CONTAINER_NAME) \
		--env-file $(ENV_FILE) \
		$(IMAGE_NAME)
	@echo "Container '$(CONTAINER_NAME)' started (host-network)."

.PHONY: docker-logs
docker-logs:
	@docker logs -f $(CONTAINER_NAME)

