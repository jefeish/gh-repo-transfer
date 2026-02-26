#!/bin/bash

# Simple test script for gh-repo-transfer
# Tests multiple use cases with execution timing

set -e

# Test repositories and target
REPOS=(
    "github-innersource/gh-repo-inspect-test-main"
    "github-innersource/gh-repo-inspect-test-sub"
)
TARGET_ORG="jester-lab"

OPTIONS=(
    "deps"
    "transfer"
)

echo "=========================================="
echo "Testing gh-repo-transfer"
echo "=========================================="
echo "Source repos: ${REPOS[0]}, ${REPOS[1]}"
echo "Target org: $TARGET_ORG"
echo ""

# Function to run test with timing
run_test() {
    local test_name="$1"
    local cmd="$2"
    
    echo "Testing: $test_name"
    echo "Command: $cmd"
    
    start_time=$(date +%s.%3N)
    if eval "$cmd"; then
        end_time=$(date +%s.%3N)
        duration=$(echo "$end_time - $start_time" | bc -l)
        printf "✓ PASSED (%.3fs)\n\n" "$duration"
    else
        echo "✗ FAILED"
        echo ""
    fi
}

# Test cases
echo "1. Basic help"
run_test "Help command" "./gh-repo-transfer --help"

echo "2. Dependency analysis"
run_test "Single repo deps" "./gh-repo-transfer deps ${REPOS[0]} --format json"
run_test "Batch repos deps" "./gh-repo-transfer deps ${REPOS[0]} ${REPOS[1]} --format json"
run_test "Deps with target org" "./gh-repo-transfer deps ${REPOS[0]} --target-org $TARGET_ORG"

echo "3. Transfer operations"
run_test "Transfer dry-run" "./gh-repo-transfer transfer ${REPOS[0]} --target-org $TARGET_ORG --dry-run"
run_test "Transfer with assign" "./gh-repo-transfer transfer ${REPOS[0]} --target-org $TARGET_ORG --dry-run --assign"
run_test "Transfer with enforce" "./gh-repo-transfer transfer ${REPOS[0]} --target-org $TARGET_ORG --dry-run --enforce"
run_test "Transfer assign + enforce" "./gh-repo-transfer transfer ${REPOS[0]} --target-org $TARGET_ORG --dry-run --assign --enforce"

echo "4. Batch transfer operations"
run_test "Batch transfer dry-run" "./gh-repo-transfer transfer ${REPOS[0]} ${REPOS[1]} --target-org $TARGET_ORG --dry-run"
run_test "Batch with assign" "./gh-repo-transfer transfer ${REPOS[0]} ${REPOS[1]} --target-org $TARGET_ORG --dry-run --assign"

echo "5. Output formats"
run_test "Table format" "./gh-repo-transfer deps ${REPOS[0]} --format table"
run_test "YAML format" "./gh-repo-transfer deps ${REPOS[0]} --format yaml"

echo "6. Verbose mode"
run_test "Verbose deps" "./gh-repo-transfer deps ${REPOS[0]} --verbose"
run_test "Verbose transfer" "./gh-repo-transfer transfer ${REPOS[0]} --target-org $TARGET_ORG --dry-run --assign --verbose"

echo "=========================================="
echo "All tests completed"
echo "=========================================="