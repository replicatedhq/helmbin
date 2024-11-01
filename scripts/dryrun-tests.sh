#!/bin/bash

set -eo pipefail

function plog() {
    local type="$1"
    local message="$2"
    local emoji="$3"
    echo -e "[$(date +'%Y-%m-%d %H:%M:%S')] $emoji [$type] $message"
}

# Build the test container
plog "INFO" "Building test container..." "🔨"
docker build -q -t ec-dryrun ./tests/dryrun > /dev/null

# Get all test functions
tests=$(grep -o 'func Test[^ (]*' ./tests/dryrun/*.go | awk '{print $2}')

# Run tests in separate containers
for test in $tests; do
    plog "INFO" "Starting test: $test" "🚀"
    docker rm -f --volumes "$test" > /dev/null 2>&1 || true
    docker run -d \
        -v "$(pwd)":/ec \
        -w /ec \
        -e GOCACHE=/ec/dev/.gocache \
        -e GOMODCACHE=/ec/dev/.gomodcache \
        --name "$test" \
        ec-dryrun \
        go test -timeout 1m -v ./tests/dryrun/... -run "^$test$" > /dev/null
done

plog "INFO" "Waiting for tests to complete..." "⏳"

# Check test results
failed_tests=()
for test in $tests; do
    exit_code=$(docker wait "$test")
    if [ "$exit_code" -ne 0 ]; then
        failed_tests+=("$test")
        plog "ERROR" "$test failed" "❌"
        docker logs "$test"
    else
        plog "INFO" "$test passed" "✅"
        docker rm -f --volumes "$test" > /dev/null
    fi
done

# Display final summary
if [ ${#failed_tests[@]} -eq 0 ]; then
    plog "SUCCESS" "All tests passed successfully!" "🎉"
    exit 0
else
    plog "FAILURE" "Some tests failed: ${failed_tests[*]}" "🚨"
    exit 1
fi
