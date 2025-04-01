include hack/base.mk

test: ## Run tests.
	go test -race -v ./...
	@rm -rf build

verify: generate verify-lint verify-dirty ## Run verification checks.

verify-lint: $(GOLANGCI)
	$(GOLANGCI) run
