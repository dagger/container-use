all: build

# Install Dagger locally if not available
.PHONY: install-dagger
install-dagger:
	@if ! which dagger >/dev/null 2>&1 && [ ! -f ./bin/dagger ]; then \
		echo "Installing Dagger locally..."; \
		mkdir -p bin; \
		curl -L https://dl.dagger.io/dagger/install.sh | DAGGER_INSTALL_DIR=./bin sh; \
	fi

.PHONY: build
build: install-dagger
	@which docker >/dev/null || ( echo "Please follow instructions to install Docker at https://docs.docker.com/get-started/get-docker/"; exit 1 )
	@export PATH="./bin:$$PATH" && docker build --platform local -o . .
	@ls cu

.PHONY: clean
clean:
	rm -f cu

.PHONY: find-path
find-path:
	@PREFERRED_DIR="$$HOME/.local/bin"; \
	if echo "$$PATH" | grep -q "$$PREFERRED_DIR"; then \
		echo "$$PREFERRED_DIR"; \
	else \
		for dir in $$(echo "$$PATH" | tr ':' ' '); do \
			if [ -w "$$dir" ]; then \
				echo "$$dir"; \
				break; \
			fi; \
		done; \
	fi

.PHONY: install
install: build
	@DEST=$$(make -s find-path | tail -n 1); \
	if [ -z "$$DEST" ]; then \
		echo "No writable directory found in \$PATH"; exit 1; \
	fi; \
	echo "Installing cu to $$DEST..."; \
	mv cu "$$DEST/"

# Run the MCP server with proper PATH setup
.PHONY: run-server
run-server: install-dagger
	@export PATH="./bin:$$PATH" && ./cu stdio

# Helper target to check if everything is set up correctly
.PHONY: check-setup
check-setup: install-dagger
	@echo "Checking setup..."
	@export PATH="./bin:$$PATH" && which dagger >/dev/null && echo "✓ Dagger is available" || echo "✗ Dagger not found"
	@which docker >/dev/null && echo "✓ Docker is available" || echo "✗ Docker not found"
	@[ -f ./cu ] && echo "✓ cu binary exists" || echo "✗ cu binary not found"
