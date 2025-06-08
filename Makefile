all: build

TARGETPLATFORM ?= local

.PHONY: check-deps
check-deps:
	@echo "Checking dependencies..."
	@which docker >/dev/null || ( echo "❌ Docker is required but not installed. Install it with: brew install docker"; exit 1 )
	@docker buildx version >/dev/null 2>&1 || ( echo "❌ Docker Buildx is required but not installed. Install it with: brew install docker-buildx"; exit 1 )
	@docker buildx ls >/dev/null 2>&1 || ( echo "❌ Docker Buildx is not properly configured"; exit 1 )
	@echo "✅ All dependencies are installed and configured"

.PHONY: build
build: check-deps
	@docker build --platform $(TARGETPLATFORM) -o . .
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
