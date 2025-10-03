# DevOps Metrics Makefile

.PHONY: build test clean sample run help

# Build the application
build:
	go build -o devops-metrics main.go

# Generate sample configuration
sample:
	go run main.go --sample-config

# Run the application
run:
	go run main.go

# Run tests (if any)
test:
	go test ./... -v

# Clean generated files
clean:
	rm -f devops-metrics
	rm -f metrics.json metrics.csv
	rm -f config.json

# Show usage
help:
	@echo "Available targets:"
	@echo "  build    - Build the application"
	@echo "  sample   - Generate sample configuration"
	@echo "  run      - Run the application"
	@echo "  test     - Run tests"
	@echo "  clean    - Clean generated files"
	@echo "  help     - Show this help"