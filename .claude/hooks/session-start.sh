#!/bin/bash
# Session start hook: dependency hygiene checks (non-blocking)

cd "${CLAUDE_PROJECT_DIR:-.}"

WARNINGS=""

# 1. Vulnerability check
if go tool govulncheck -help &>/dev/null 2>&1; then
    VULN_OUTPUT=$(go tool govulncheck ./... 2>&1)
    VULN_EXIT=$?
    if [ $VULN_EXIT -ne 0 ]; then
        WARNINGS="${WARNINGS}VULNERABILITIES (govulncheck):\n${VULN_OUTPUT}\n\n"
    fi
fi

# 2. Module integrity check
VERIFY_OUTPUT=$(go mod verify 2>&1)
VERIFY_EXIT=$?
if [ $VERIFY_EXIT -ne 0 ]; then
    WARNINGS="${WARNINGS}MODULE INTEGRITY (go mod verify):\n${VERIFY_OUTPUT}\n\n"
fi

# 3. Dependency tidiness check
TIDY_OUTPUT=$(go mod tidy -diff 2>&1)
if [ -n "$TIDY_OUTPUT" ]; then
    WARNINGS="${WARNINGS}UNTIDY DEPENDENCIES (go mod tidy -diff):\n${TIDY_OUTPUT}\n\n"
fi

if [ -n "$WARNINGS" ]; then
    echo -e "Session start checks found issues:\n${WARNINGS}" >&2
    echo "These are non-blocking warnings. Consider fixing them during this session." >&2
    echo "Re-run this check with: bash \"${CLAUDE_PROJECT_DIR:-.}\"/.claude/hooks/session-start.sh" >&2
    exit 0
fi

exit 0
