.PHONY: test bench clean coverage lint

# Default test command
test:
	go test -v -race ./...

# Run benchmarks and save results
bench:
	@echo "Running benchmarks for kbuffer package..."
	@go test -bench=. -benchmem -benchtime=1s -run=^$$ . > result_bench 2>&1
	@echo "Benchmark results saved to result_bench"
	@echo "Summary:"
	@grep -E "Benchmark.*-[0-9]+" result_bench | head -20

# Run tests with coverage
coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	golangci-lint run

# Clean generated files
clean:
	rm -f result_bench coverage.out coverage.html

# Run all tests and benchmarks
all: test bench coverage