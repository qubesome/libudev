include hack/base.mk

test: ## Run tests.
	go test -race -v ./...

verify: verify-lint verify-dirty ## Run verification checks.

verify-lint: $(GOLANGCI)
	$(GOLANGCI) run
