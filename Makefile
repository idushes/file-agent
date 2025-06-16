# Переменные
DOCKER_USERNAME ?= dushes
IMAGE_NAME = file-agent
VERSION ?= latest
FULL_IMAGE_NAME = $(DOCKER_USERNAME)/$(IMAGE_NAME):$(VERSION)

# Цвета для вывода
GREEN = \033[0;32m
RED = \033[0;31m
YELLOW = \033[1;33m
NC = \033[0m # No Color

.PHONY: help build tag push all clean test run stop logs

# Показать справку
help:
	@echo "$(YELLOW)File Agent - Docker Build Commands$(NC)"
	@echo ""
	@echo "$(GREEN)Available commands:$(NC)"
	@echo "  help          - Показать эту справку"
	@echo "  build         - Собрать мультиплатформенный образ (amd64+arm64)"
	@echo "  build-local   - Собрать образ только для текущей платформы"
	@echo "  build-amd64   - Собрать образ только для amd64 (Kubernetes)"
	@echo "  push          - Собрать и загрузить мультиплатформенный образ"
	@echo "  push-local    - Загрузить локально собранный образ"
	@echo "  all           - Собрать и загрузить мультиплатформенный образ"
	@echo "  test          - Протестировать образ локально"
	@echo "  clean         - Удалить локальные образы"
	@echo "  info          - Показать информацию о образе"
	@echo ""
	@echo "$(GREEN)Переменные окружения:$(NC)"
	@echo "  DOCKER_USERNAME - имя пользователя Docker Hub (по умолчанию: dushes)"
	@echo "  VERSION         - версия образа (по умолчанию: latest)"
	@echo ""
	@echo "$(GREEN)Примеры:$(NC)"
	@echo "  make build"
	@echo "  make all VERSION=v1.0.0"
	@echo "  make push DOCKER_USERNAME=myuser VERSION=v1.0.0"

# Собрать мультиплатформенный Docker образ (используя облачный билдер)
build:
	@echo "$(GREEN)Building multiplatform Docker image...$(NC)"
	docker buildx build --platform linux/amd64,linux/arm64 \
		-t $(IMAGE_NAME):$(VERSION) \
		--load .
	@echo "$(GREEN)✅ Multiplatform build completed: $(IMAGE_NAME):$(VERSION)$(NC)"

# Собрать только для amd64 (для тестирования на локальной машине)
build-local:
	@echo "$(GREEN)Building Docker image for local testing...$(NC)"
	docker build -t $(IMAGE_NAME):$(VERSION) .
	@echo "$(GREEN)✅ Local build completed: $(IMAGE_NAME):$(VERSION)$(NC)"

# Собрать образ для amd64 (Kubernetes серверы)
build-amd64:
	@echo "$(GREEN)Building Docker image for amd64...$(NC)"
	docker buildx build --platform linux/amd64 \
		-t $(IMAGE_NAME):$(VERSION) \
		--load .
	@echo "$(GREEN)✅ Build completed for amd64: $(IMAGE_NAME):$(VERSION)$(NC)"

# Тегировать образ для Docker Hub
tag: build
	@echo "$(GREEN)Tagging image for Docker Hub...$(NC)"
	docker tag $(IMAGE_NAME):$(VERSION) $(FULL_IMAGE_NAME)
	@echo "$(GREEN)✅ Tagged: $(FULL_IMAGE_NAME)$(NC)"

# Проверить авторизацию в Docker Hub
check-login:
	@echo "$(YELLOW)Checking Docker Hub login...$(NC)"
	@if ! grep -q "https://index.docker.io/v1/" ~/.docker/config.json 2>/dev/null; then \
		echo "$(RED)❌ Not logged in to Docker Hub. Run 'docker login' first.$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✅ Docker Hub login verified$(NC)"

# Собрать и загрузить мультиплатформенный образ на Docker Hub
push: check-login
	@echo "$(GREEN)Building and pushing multiplatform image to Docker Hub...$(NC)"
	docker buildx build --platform linux/amd64,linux/arm64 \
		-t $(FULL_IMAGE_NAME) \
		--push .
	@echo "$(GREEN)✅ Multiplatform image pushed: $(FULL_IMAGE_NAME)$(NC)"

# Загрузить локально собранный образ на Docker Hub
push-local: check-login
	@echo "$(GREEN)Tagging and pushing local image...$(NC)"
	docker tag $(IMAGE_NAME):$(VERSION) $(FULL_IMAGE_NAME)
	docker push $(FULL_IMAGE_NAME)
	@echo "$(GREEN)✅ Local image pushed: $(FULL_IMAGE_NAME)$(NC)"

# Собрать и загрузить мультиплатформенный образ
all: push
	@echo "$(GREEN)🎉 All done! Multiplatform image available at: $(FULL_IMAGE_NAME)$(NC)"

# Протестировать образ локально
test: build
	@echo "$(GREEN)Testing Docker image...$(NC)"
	@echo "$(YELLOW)Starting container...$(NC)"
	docker run --rm -d \
		--name file-agent-test \
		-p 8082:8082 \
		-e PORT=8082 \
		$(IMAGE_NAME):$(VERSION)
	@echo "$(YELLOW)Waiting for container to start...$(NC)"
	@sleep 3
	@echo "$(YELLOW)Testing health endpoint...$(NC)"
	@if curl -s http://localhost:8082/health | grep -q "OK"; then \
		echo "$(GREEN)✅ Health check passed$(NC)"; \
	else \
		echo "$(RED)❌ Health check failed$(NC)"; \
		docker stop file-agent-test; \
		exit 1; \
	fi
	@echo "$(YELLOW)Stopping test container...$(NC)"
	docker stop file-agent-test
	@echo "$(GREEN)✅ Test completed successfully$(NC)"

# Удалить локальные образы
clean:
	@echo "$(GREEN)Cleaning up local images...$(NC)"
	@docker images -q $(IMAGE_NAME) | xargs -r docker rmi -f
	@docker images -q $(DOCKER_USERNAME)/$(IMAGE_NAME) | xargs -r docker rmi -f
	@echo "$(GREEN)✅ Cleanup completed$(NC)"

# Показать информацию о образе
info:
	@echo "$(GREEN)Image Information:$(NC)"
	@echo "  Local image:    $(IMAGE_NAME):$(VERSION)"
	@echo "  Docker Hub:     $(FULL_IMAGE_NAME)"
	@echo "  Size:           $$(docker images --format 'table {{.Size}}' $(IMAGE_NAME):$(VERSION) 2>/dev/null | tail -n +2 || echo 'Not built')"

# Сборка с кэшем
build-no-cache:
	@echo "$(GREEN)Building Docker image without cache...$(NC)"
	docker build --no-cache -t $(IMAGE_NAME):$(VERSION) .
	@echo "$(GREEN)✅ Build completed: $(IMAGE_NAME):$(VERSION)$(NC)" 