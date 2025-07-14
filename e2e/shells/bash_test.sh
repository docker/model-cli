#!/bin/bash

# Bash-specific e2e test for docker model list command with fixture validation
# This script validates the output against the expected fixture

set -e

TEST_MODEL="ai/smollm2"
FAILED_TESTS=0
TOTAL_TESTS=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    FAILED_TESTS=$((FAILED_TESTS + 1))
}

# Load fixture file and normalize lines
load_fixture() {
    local fixture_path="$1"
    if [ ! -f "$fixture_path" ]; then
        log_error "Fixture file not found: $fixture_path"
        return 1
    fi
    # Remove trailing whitespace
    sed 's/[[:space:]]*$//' "$fixture_path"
}

run_test_with_fixture_validation() {
    local test_name="$1"
    local command="$2"
    local fixture_path="$3"
    local expected_exit_code="${4:-0}"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    log_info "Running test: $test_name"
    
    # Load expected output from fixture
    expected_output=$(load_fixture "$fixture_path")

    if [ $? -ne 0 ]; then
        log_error "$test_name - Failed to load fixture"
        return 1
    fi
    
    # Run command and capture output and exit code
    set +e
    output=$(eval "$command" 2>&1)
    exit_code=$?
    set -e

    # Normalize output: remove leading/trailing blank lines and trailing spaces
    normalized_output=$(echo "$output" | sed 's/[[:space:]]*$//')

    if [ $exit_code -eq $expected_exit_code ] && [ "$normalized_output" = "$expected_output" ]; then
        log_success "$test_name"
        return 0
    else
        log_error "$test_name"
        echo "Command: $command"
        echo "Expected exit code: $expected_exit_code, got: $exit_code"
        echo "Expected output:"
        echo "$expected_output"
        echo "Actual normalized output:"
        echo "$normalized_output"
        return 1
    fi
}

# Check prerequisites
log_info "Checking prerequisites..."

# Ensure we're running from the correct directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

# Check if docker model is available
if ! command -v docker &> /dev/null; then
    log_error "docker command not found"
    exit 1
fi

# Check if model runner is running
if ! docker model status | grep -q "Docker Model Runner is running"; then
    log_error "Docker Model Runner is not running"
    exit 1
fi

# Check if test model is available
if ! docker model list | grep -q "$TEST_MODEL"; then
    log_info "Test model $TEST_MODEL not found, pulling..."
    docker model pull "$TEST_MODEL"
fi

log_info "Starting fixture validation test..."

# Test: Fixture format validation
run_test_with_fixture_validation "Fixture format validation" \
    "docker model list" \
    "e2e/fixtures/table_format/with_models.txt"

# Summary
echo
log_info "Test Summary:"
echo "Total tests: $TOTAL_TESTS"
echo "Failed tests: $FAILED_TESTS"
echo "Passed tests: $((TOTAL_TESTS - FAILED_TESTS))"

if [ $FAILED_TESTS -eq 0 ]; then
    log_success "Fixture validation test passed!"
    exit 0
else
    log_error "$FAILED_TESTS tests failed"
    exit 1
fi
