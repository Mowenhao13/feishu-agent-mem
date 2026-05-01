.PHONY: build build-mcp test run clean docker-build docker-run

BINARY := bin/mem-service
MCP_BINARY := bin/mcp-server
DOCKER_IMAGE := feishu-agent-mem:latest

build:
	@echo "Building feishu-agent-mem..."
	@go build -o $(BINARY) ./cmd/mem-service/

build-mcp:
	@echo "Building MCP server..."
	@go build -o $(MCP_BINARY) ./cmd/mcp-server/

test:
	@go test ./test/p1 ./test/p2 ./test/p3 ./test/p4 -v

test-all:
	@go test ./test/... -v

run: build
	@./$(BINARY)

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY)
	@rm -rf data/

docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .

docker-run:
	@echo "Running Docker container..."
	@docker run -d \
		--name feishu-agent-mem \
		-p 37777:37777 \
		-v $(PWD)/data:/opt/feishu-agent-mem/data \
		-v $(PWD)/config:/opt/feishu-agent-mem/config \
		--env-file .env \
		$(DOCKER_IMAGE)

docker-stop:
	@docker stop feishu-agent-mem || true
	@docker rm feishu-agent-mem || true

fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Vetting code..."
	@go vet ./...
