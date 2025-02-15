# .RECIPEPREFIX := $(.RECIPEPREFIX)<space>
TESTCOVERAGE_THRESHOLD=0

ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))

default: help

.PHONY: help
help: # Shows help for each of the Makefile recipes.
	@echo "\033[00mCommand\033[00m\t\t\t Description"
	@echo "-------\t\t\t -----------"
	@grep -E '^[a-zA-Z0-9 -]+:.*#' Makefile | \
		while read -r l; do \
			cmd=$$(echo $$l | cut -f 1 -d':'); \
			desc=$$(echo $$l | cut -f 2- -d'#'); \
			printf "\033[1;32m%-16s\033[00m\t%s\n" "$$cmd" "$$desc"; \
		done

.PHONY: dep
dep: # Download dependencies.
	go mod download
	go mod tidy

.PHONY: init-tools
init-tools: # Initialize the tools.
	command -v pre-commit >/dev/null || brew install pre-commit
	command -v make >/dev/null || brew install make
	command -v act >/dev/null || brew install act
	[ -f .git/hooks/pre-commit ] || pre-commit install
	go tool -n golangci-lint >/dev/null || go get -tool github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go tool -n betteralign >/dev/null || go get -tool github.com/dkorunic/betteralign/cmd/betteralign@latest
	go tool -n gcov2lcov >/dev/null || go get -tool github.com/jandelgado/gcov2lcov@latest
	go tool -n govulncheck >/dev/null || go get -tool golang.org/x/vuln/cmd/govulncheck@latest
	go tool -n mockgen >/dev/null || go get -tool go.uber.org/mock/mockgen@latest
	go tool -n stringer >/dev/null || go get -tool golang.org/x/tools/cmd/stringer@latest

.PHONY: init
init: init-tools dep # Initialize the project.

.PHONY: build
build: # Build the entire codebase.
	go build -v ./...

.PHONY: generate
generate: # Run auto-generated code generation.
	go generate ./...

.PHONY: clean
clean: # Clean the entire codebase.
	go clean

.PHONY: test
test: # Run the tests.
	go test -failfast -race -count 1 ./...

.PHONY: test-cov
test-cov: # Run the tests with coverage.
	go test -failfast -race -count 1 -coverpkg=./... -coverprofile=${TMPDIR}cov_profile.out ./...
	# gcov2lcov -infile ${TMPDIR}cov_profile.out -outfile ./cov_profile.lcov

.PHONY: test-view-html
test-view-html: # View the test coverage in HTML.
	go tool cover -html ${TMPDIR}cov_profile.out -o ${TMPDIR}cov_profile.html
	open ${TMPDIR}cov_profile.html

.PHONY: test-ci
test-ci: test-cov # Run the tests with coverage and check if it's above the threshold.
	$(eval ACTUAL_COVERAGE := $(shell go tool cover -func=${TMPDIR}cov_profile.out | grep total | grep -Eo '[0-9]+\.[0-9]+'))

	@echo "Quality Gate: checking test coverage is above threshold..."
	@echo "Threshold             : $(TESTCOVERAGE_THRESHOLD) %"
	@echo "Current test coverage : $(ACTUAL_COVERAGE) %"

	@if [ "$(shell echo "$(ACTUAL_COVERAGE) < $(TESTCOVERAGE_THRESHOLD)" | bc -l)" -eq 1 ]; then \
    echo "Current test coverage is below threshold. Please add more unit tests or adjust threshold to a lower value."; \
    echo "Failed"; \
    exit 1; \
  else \
    echo "OK"; \
  fi

.PHONY: lint
lint: # Run the linting command.
	go tool golangci-lint run ./...

.PHONY: check
check: # Runs static analysis tools.
	go tool govulncheck ./...
	go tool betteralign ./...
	go vet ./...

.PHONY: fix
fix: # Fixes code formatting and alignment.
	go tool betteralign -apply ./...
	go fmt ./...

%:
	@:
