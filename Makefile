all: cu

TARGETPLATFORM ?= local

cu:
	@./hack/build.sh

.PHONY: clean
clean:
	rm -f cu

.PHONY: install
install: cu
	@./hack/install.sh

.PHONY: uninstall
uninstall:
	@./hack/uninstall.sh
