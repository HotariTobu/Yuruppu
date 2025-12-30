#!/bin/bash
set -Eeuo pipefail

SPEC_NAME=${1:-}
MAX_LOOPS=${2:-100}

if [ -z "$SPEC_NAME" ]; then
    echo "Usage: $0 <spec-name> [max-loops]"
    exit 1
fi

PROGRESS_FILE="docs/specs/$SPEC_NAME/progress.json"

if [ ! -f "$PROGRESS_FILE" ]; then
    echo "Error: $PROGRESS_FILE not found"
    exit 1
fi

SYSTEM_PROMPT="CRITICAL OVERRIDE: You are running in FULLY AUTONOMOUS HEADLESS MODE. There is NO human present. Any attempt to ask questions, wait for input, or use AskUserQuestion will FAIL. You MUST:
1. NEVER use AskUserQuestion tool - it is DISABLED
2. NEVER wait for user confirmation or approval
3. NEVER ask clarifying questions
4. Make ALL decisions autonomously using your best judgment
5. Proceed immediately with implementation without hesitation
6. If uncertain, choose the most reasonable option and continue
The user has PRE-APPROVED all actions. Execute tasks completely and independently."

for ((i=1; i<=MAX_LOOPS; i++)); do
    echo "=== Loop $i/$MAX_LOOPS ==="

    if grep -q '"phase": "completed"' "$PROGRESS_FILE"; then
        echo "Phase completed!"
        exit 0
    fi

    if grep -q '"phase": "blocked"' "$PROGRESS_FILE"; then
        echo "Phase blocked! Check blockers in $PROGRESS_FILE"
        exit 1
    fi

    claude --print -p "/session-start $SPEC_NAME" --append-system-prompt "$SYSTEM_PROMPT"
    claude --print -p "/session-end" $SPEC_NAME" --append-system-prompt "$SYSTEM_PROMPT"
done

echo "Max loops reached"
