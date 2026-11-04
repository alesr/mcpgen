#!/usr/bin/env bash
# verify-quickstart.sh — executes the README Quickstart commands in a temp
# directory and fails on any error or unexpected output. Every command here
# mirrors the "Quickstart (60 seconds)" section of README.md.
#
# It needs network access (go module downloads, and npx fetches the MCP
# conformance suite during `mcpgen new`/`check`), so CI runs it as a
# separate job.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

step() { printf '\n==> %s\n' "$*"; }
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

step "build mcpgen from source"
(cd "$REPO_ROOT" && go build -o "$TMP/mcpgen" .) || fail "go build"
export PATH="$TMP:$PATH"
mcpgen version | grep -q '^mcpgen v' || fail "mcpgen version output"

cd "$TMP"

step "1. mcpgen new ./weather --name weather --module example.com/weather --transport http --json"
mcpgen new ./weather --name weather --module example.com/weather --transport http --json > new.json \
  || fail "mcpgen new exited non-zero"
grep -q '"ok": true' new.json || fail "new envelope not ok"
grep -q '"cmd/weather/main.go"' new.json || fail "new did not write cmd/weather/main.go"
grep -q '"mcpgen.json"' new.json || fail "new did not write mcpgen.json"

step "2. mcpgen check ./weather --json"
mcpgen check ./weather --json > check1.json || fail "check exited non-zero (expected exit 0)"
grep -q '"ok": true' check1.json || fail "check envelope not ok"
grep -q '"name": "conformance"' check1.json || fail "check did not run the conformance stage"
grep -q '"status": "passed"' check1.json || fail "check has no passed stage"

step "3. run the server and curl /healthz"
(cd weather && go build -o "$TMP/weather-server" ./cmd/weather) || fail "build generated server"
"$TMP/weather-server" -addr 127.0.0.1:18080 >/dev/null 2>&1 &
SRV=$!
trap 'kill $SRV 2>/dev/null || true; rm -rf "$TMP"' EXIT
for i in $(seq 1 20); do curl -sf http://127.0.0.1:18080/healthz > healthz.txt && break; sleep 0.5; done
grep -q '^ok$' healthz.txt || fail "/healthz did not return ok"
kill $SRV 2>/dev/null || true
wait $SRV 2>/dev/null || true

step "4. mcpgen add tool tools/health.json --dir ./weather --json"
mkdir -p tools
cat > tools/health.json <<'EOF'
{
  "id": "health",
  "title": "Health check",
  "description": "Reports server health.",
  "inputSchema": { "type": "object", "properties": {} },
  "readOnly": true
}
EOF
mcpgen add tool tools/health.json --dir ./weather --json > add.json || fail "add exited non-zero"
grep -q '"ok": true' add.json || fail "add envelope not ok"
grep -q '"id": "health"' add.json || fail "add did not register id health"
grep -q 'internal/mcpapp/tools/handlers/health.go' add.json || fail "add did not create the handler file"
test -f weather/internal/mcpapp/tools/handlers/health.go || fail "handler file missing on disk"

step "5. mcpgen check ./weather --json (final)"
mcpgen check ./weather --json > check2.json || fail "final check exited non-zero"
grep -q '"ok": true' check2.json || fail "final check envelope not ok"
grep -q '2 tools' check2.json || fail "config stage did not see 2 tools"

printf '\nPASS: quickstart commands from README.md all succeeded\n'
