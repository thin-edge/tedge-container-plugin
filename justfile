set dotenv-load
set export

TEST_IMAGE := env_var_or_default("TEST_IMAGE", "debian-systemd-docker-cli")

# Initialize a dotenv file for usage with a local debugger
# WARNING: It will override any previously generated dotenv file
init-dotenv:
  @echo "Recreating .env file..."
  @echo "TEST_IMAGE=$TEST_IMAGE" >> .env
  @echo "C8Y_BASEURL=$C8Y_BASEURL" >> .env
  @echo "C8Y_USER=$C8Y_USER" >> .env
  @echo "C8Y_PASSWORD=$C8Y_PASSWORD" >> .env
  @echo "PRIVATE_IMAGE=docker.io/example/app:latest" >> .env
  @echo "REGISTRY1_REPO=docker.io" >> .env
  @echo "REGISTRY1_USERNAME=" >> .env
  @echo "REGISTRY1_PASSWORD=" >> .env

# Run linting
lint:
    golangci-lint run

# Release all artifacts
release *ARGS='':
    mkdir -p output
    go run main.go completion bash > output/completions.bash
    go run main.go completion zsh > output/completions.zsh
    go run main.go completion fish > output/completions.fish

    docker context use default
    goreleaser release --clean --auto-snapshot {{ARGS}}

# Build a release locally (for testing the release artifacts)
release-local:
    just -f "{{justfile()}}" release --snapshot

# Install python virtual environment
venv:
  [ -d .venv ] || python3 -m venv .venv
  ./.venv/bin/pip3 install -r tests/requirements.txt

# Build test images and test artifacts
build-test:
  docker buildx install
  docker build --load -t {{TEST_IMAGE}} -f ./test-images/{{TEST_IMAGE}}/Dockerfile .
  ./tests/data/apps/build.sh

# Run tests
test *args='':
  ./.venv/bin/python3 -m robot.run --outputdir output {{args}} tests

# Download/update vendor packages
update-vendor:
  go mod vendor

# Print yocto licensing string
print-yocto-licenses:
  @echo 'LIC_FILES_CHKSUM = " \'
  @find . -name "LICENSE*" -exec bash -c 'echo -n "    file://src/\${GO_IMPORT}/{};md5=" && md5 -q {}' \; 2>/dev/null | grep -v "/\.venv/" | sed 's|$| \\|g' | sed 's|/\./|/|g'
  @echo '"'
