default: install

# Build the provider binary
build:
	go build -o terraform-provider-sequin

# Install the provider locally for testing
# Installs to the local plugin directory for darwin_arm64
install: build
	mkdir -p ~/.terraform.d/plugins/local/clintdigital/sequin/0.1.0/darwin_arm64
	cp terraform-provider-sequin ~/.terraform.d/plugins/local/clintdigital/sequin/0.1.0/darwin_arm64/

# Run unit tests
test:
	go test -v ./...

# Run acceptance tests (requires SEQUIN_ENDPOINT and SEQUIN_API_KEY env vars)
testacc:
	TF_ACC=1 go test -v ./... -timeout 30m

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint to be installed)
lint:
	golangci-lint run

# Generate documentation
docs:
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	tfplugindocs generate

# Clean build artifacts
clean:
	rm -f terraform-provider-sequin
	rm -rf dist/

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run the provider in debug mode (useful for development)
debug:
	go build -gcflags="all=-N -l" -o terraform-provider-sequin
	./terraform-provider-sequin -debug

.PHONY: build install test testacc fmt lint docs clean deps debug default
