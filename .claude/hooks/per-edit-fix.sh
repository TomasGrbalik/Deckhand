#!/bin/bash
# Per-edit hook: auto-format Go files on every Edit|Write
# Exit 0 = success, Exit 2 = unfixable issue fed back to Claude

INPUT=$(cat)

# Parse file path — prefer jq, fall back to grep+sed for environments without jq
if command -v jq &>/dev/null; then
    FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.filePath // empty')
else
    FILE_PATH=$(echo "$INPUT" | grep -oE '"(file_path|filePath)"\s*:\s*"[^"]*"' | head -1 | sed 's/.*: *"//;s/"$//')
fi

# Only process Go files
if [[ -z "$FILE_PATH" ]] || [[ "$FILE_PATH" != *.go ]]; then
    exit 0
fi

# Verify file exists
if [[ ! -f "$FILE_PATH" ]]; then
    exit 0
fi

fail() {
    local name="$1" cmd="$2" output="$3" hint="$4"
    echo "" >&2
    echo "PER-EDIT CHECK FAILED [$name] in ${FILE_PATH}:" >&2
    echo "Command: $cmd" >&2
    echo "" >&2
    echo "$output" >&2
    echo "" >&2
    if [ -n "$hint" ]; then
        echo "Hint: $hint" >&2
        echo "" >&2
    fi
    echo "ACTION REQUIRED: You MUST fix the issue shown above. Read the file at the reported line, edit the source code to resolve it, and the check will re-run on next edit." >&2
    exit 2
}

# Helper: resolve tool command — prefer go tool when tool directives exist,
# fall back to standalone binary.
resolve_tool() {
    local tool="$1"
    if go tool "$tool" --help &>/dev/null 2>&1; then
        echo "go tool $tool"
    elif command -v "$tool" &>/dev/null; then
        echo "$tool"
    else
        echo ""
    fi
}

# 1. gofumpt (strict superset of gofmt) — format the file
GOFUMPT_CMD=$(resolve_tool gofumpt)
GOFMT_CMD=$(resolve_tool gofmt)
if [ -n "$GOFUMPT_CMD" ]; then
    FMT_OUTPUT=$($GOFUMPT_CMD -w "$FILE_PATH" 2>&1)
    if [ $? -ne 0 ]; then
        fail "gofumpt" "$GOFUMPT_CMD -w $FILE_PATH" "$FMT_OUTPUT" \
            "gofumpt failed — this usually means a syntax error prevents formatting. Read the file at the reported line and fix the syntax error first."
    fi
elif [ -n "$GOFMT_CMD" ]; then
    FMT_OUTPUT=$($GOFMT_CMD -w "$FILE_PATH" 2>&1)
    if [ $? -ne 0 ]; then
        fail "gofmt" "$GOFMT_CMD -w $FILE_PATH" "$FMT_OUTPUT" \
            "gofmt failed — this usually means a syntax error prevents formatting. Read the file at the reported line and fix the syntax error first."
    fi
fi

# 2. goimports — fix imports
GOIMPORTS_CMD=$(resolve_tool goimports)
if [ -n "$GOIMPORTS_CMD" ]; then
    IMP_OUTPUT=$($GOIMPORTS_CMD -w "$FILE_PATH" 2>&1)
    if [ $? -ne 0 ]; then
        fail "goimports" "$GOIMPORTS_CMD -w $FILE_PATH" "$IMP_OUTPUT" \
            "goimports failed. Read the file and check for syntax errors or unresolvable import paths."
    fi
fi

exit 0
