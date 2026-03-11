#!/bin/bash
# Quality gate hook for Claude Code Stop event
# Fail-fast: stops at the first failing check, exits 2 for Claude to fix.

set -o pipefail

HOOK_LOG="${CLAUDE_PROJECT_DIR:-.}/.claude/hooks/hook-debug.log"
WORKTREE_ID="$(basename "${CLAUDE_PROJECT_DIR:-.}")"
debuglog() {
    echo "[quality-gate@${WORKTREE_ID}] $(date '+%Y-%m-%d %H:%M:%S') $1" >> "$HOOK_LOG"
}
debuglog "=== HOOK STARTED (pid=$$) ==="

cd "${CLAUDE_PROJECT_DIR:-.}"

declare -A TOOL_HINTS
TOOL_HINTS=(
    [go-test]="Read the failing test and the source it tests. Run 'go test -v -run TestName ./pkg/...' to see the full output. Fix the source code, not the test, unless the test itself is wrong."
    [coverage]="Run 'go tool cover -func=coverage.out' to see which functions are uncovered. Add tests for the uncovered code paths."
    [golangci-lint]="Read the file at the reported line. The linter name is shown in brackets. Run 'golangci-lint run ./path/to/pkg/...' to re-check a single package after fixing."
    [golangci-fmt]="Run 'golangci-lint fmt ./...' to auto-fix all formatting issues. If specific files need attention, run 'gofumpt -w <file>' directly."
    [govulncheck]="Run 'govulncheck ./...' to see full vulnerability details. Update the affected dependency: 'go get <module>@latest' then 'go mod tidy'."
    [go-mod-verify]="Run 'go mod verify' to check module integrity. If checksums don't match, run 'go mod download' to re-fetch."
    [go-mod-tidy]="Run 'go mod tidy' to clean up go.mod and go.sum. Check for unused imports in source code."
    [semver-format]="The version string in internal/cli/root.go does not follow semver 2.0 format (MAJOR.MINOR.PATCH[-prerelease][+build]). Read internal/cli/root.go and fix the version constant."
)

fail() {
    local name="$1"
    local cmd="$2"
    local output="$3"
    local hint="${TOOL_HINTS[$name]:-}"

    echo "" >&2
    echo "QUALITY GATE FAILED [$name]:" >&2
    echo "Command: $cmd" >&2
    echo "" >&2
    echo "$output" >&2
    echo "" >&2
    if [ -n "$hint" ]; then
        echo "Hint: $hint" >&2
        echo "" >&2
    fi
    echo "ACTION REQUIRED: You MUST fix the issue shown above. Do NOT stop or explain — read the failing file, edit the source code to resolve it, and the quality gate will re-run automatically." >&2
    debuglog "=== FAILED: $name ==="
    exit 2
}

run_check() {
    local name="$1"; shift
    local cmd="$*"
    debuglog "Running $name..."
    OUTPUT=$("$@" 2>&1) || fail "$name" "$cmd" "$OUTPUT"
}

run_check_nonempty() {
    local name="$1"; shift
    local cmd="$*"
    debuglog "Running $name..."
    OUTPUT=$("$@" 2>&1)
    [ -n "$OUTPUT" ] && fail "$name" "$cmd" "$OUTPUT"
}

# Checks ordered by speed and likelihood of failure.
# Race detector requires CGo/gcc — enabled in CI, skipped locally if gcc is absent.
if command -v gcc &>/dev/null; then
    RACE_FLAG="-race"
else
    RACE_FLAG=""
fi
run_check        "go-test"        go test $RACE_FLAG -coverprofile=coverage.out -covermode=atomic -coverpkg=./... -count=1 -failfast -shuffle=on ./...
run_check        "coverage"       go tool go-test-coverage --config=.testcoverage.yml
run_check        "golangci-lint"  golangci-lint run ./...
run_check_nonempty "golangci-fmt" golangci-lint fmt --diff ./...
run_check        "govulncheck"    go tool govulncheck ./...
run_check        "go-mod-verify"  go mod verify
run_check_nonempty "go-mod-tidy"  go mod tidy -diff

# Version format validation (Dimension 9 — level 2: format only)
CURRENT_VERSION=$(awk -F'"' '/var Version/ {print $2}' internal/cli/root.go 2>/dev/null)
if [ -n "$CURRENT_VERSION" ] && [ "$CURRENT_VERSION" != "dev" ]; then
    SEMVER_RE='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-([0-9A-Za-z-]+\.)*[0-9A-Za-z-]+)?(\+([0-9A-Za-z-]+\.)*[0-9A-Za-z-]+)?$'
    if ! echo "$CURRENT_VERSION" | grep -qE "$SEMVER_RE"; then
        fail "semver-format" "awk -F'\"' '/var Version/ {print \$2}' internal/cli/root.go" "Version '${CURRENT_VERSION}' is not valid semver 2.0."
    fi
fi

debuglog "=== ALL CHECKS PASSED ==="
exit 0
