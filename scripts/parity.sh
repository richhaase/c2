#!/usr/bin/env bash
# Diffs the Go build against the TypeScript build across the real data store.
# Both read the same store, so any difference is a port defect.
# Usage: scripts/parity.sh [-v]
set -uo pipefail

cd "$(dirname "$0")/.."

VERBOSE=0
[[ "${1:-}" == "-v" ]] && VERBOSE=1

if [[ -t 1 ]]; then
    RED=$'\033[0;31m'; GREEN=$'\033[0;32m'; YELLOW=$'\033[0;33m'; NC=$'\033[0m'
else
    RED='' GREEN='' YELLOW='' NC=''
fi

COMMANDS=(
    "log -n 25"
    "log -n 25 --json"
    "log --from 2026-01-01 --to 2026-03-01"
    "log -n 5 --json --from 2026-06-01"
    "status"
    "status --json"
    "trend -w 8"
    "trend -w 8 --json"
    "stats weekly -w 12"
    "stats weekly -w 12 --json"
    "stats goal"
    "stats goal --json"
    "stats splits last"
    "stats splits last --json"
    "stats hr-pace -w 8"
    "stats hr-pace -w 8 --json"
    "show last"
    "show last --json"
    "export -f csv"
    "export -f json"
    "export -f jsonl"
    "export -f csv --from 2026-05-01"
    "data info"
    "data info --json"
    "data doctor"
    "note list"
    "note list --json"
    "note list --since 2026-06-01"
    "narrative list"
    "narrative list --json"
    "plan show"
    "playbook show"
    "report --data"
)

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

if [[ ! -x ./bin/c2 ]]; then
    echo "${RED}error:${NC} ./bin/c2 not built — run 'make build' first" >&2
    exit 1
fi

# generated_at is a wall-clock timestamp and always differs between runs.
normalize() {
    sed -e 's/"generated_at": "[^"]*"/"generated_at": "<ts>"/' "$1"
}

pass=0 fail=0 skip=0
failed_commands=()

for cmd in "${COMMANDS[@]}"; do
    eval "bun src/index.ts $cmd" >"$TMP/ts.out" 2>"$TMP/ts.err"
    ts_status=$?
    eval "./bin/c2 $cmd" >"$TMP/go.out" 2>"$TMP/go.err"
    go_status=$?

    if grep -q "unknown command\|unknown flag\|unknown shorthand" "$TMP/go.err"; then
        printf "  %sSKIP%s  %s (not ported yet)\n" "$YELLOW" "$NC" "$cmd"
        skip=$((skip + 1))
        continue
    fi

    normalize "$TMP/ts.out" >"$TMP/ts.norm"
    normalize "$TMP/go.out" >"$TMP/go.norm"

    if diff -q "$TMP/ts.norm" "$TMP/go.norm" >/dev/null && [[ "$ts_status" == "$go_status" ]]; then
        printf "  %sok%s    %s\n" "$GREEN" "$NC" "$cmd"
        pass=$((pass + 1))
    else
        printf "  %sFAIL%s  %s" "$RED" "$NC" "$cmd"
        [[ "$ts_status" != "$go_status" ]] && printf " (exit %s vs %s)" "$ts_status" "$go_status"
        printf "\n"
        fail=$((fail + 1))
        failed_commands+=("$cmd")
        if [[ $VERBOSE == 1 ]]; then
            diff "$TMP/ts.norm" "$TMP/go.norm" | head -30 | sed 's/^/        /'
        fi
    fi
done

echo
echo "parity: $pass passed, $fail failed, $skip not yet ported"
if [[ $fail -gt 0 ]]; then
    echo "re-run with -v to see diffs, or:"
    for c in "${failed_commands[@]}"; do
        echo "  diff <(bun src/index.ts $c) <(./bin/c2 $c)"
    done
    exit 1
fi
