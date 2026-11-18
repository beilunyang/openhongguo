#!/usr/bin/env bash
# Compare the Go signer with the real metasec SDK running under unidbg.
#
#   scripts/compare.sh                 build, start openhongguo-signer, compare, stop it
#   scripts/compare.sh --relay URL     use an emulator that is already running
#
# Extra arguments go to the compare program (for example -query 闪婚).
# Needs JDK 21 (JAVA_HOME, else Homebrew openjdk@21) and Go.
set -euo pipefail

here="$(cd "$(dirname "$0")/.." && pwd)"
port="${PORT:-19833}"
relay=""
args=()
while [ $# -gt 0 ]; do
  case "$1" in
    --relay) relay="$2"; shift 2 ;;
    *) args+=("$1"); shift ;;
  esac
done

pid=""
cleanup() { [ -n "$pid" ] && kill "$pid" 2>/dev/null || true; }
trap cleanup EXIT

if [ -z "$relay" ]; then
  export JAVA_HOME="${JAVA_HOME:-/opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home}"
  (cd "$here" && ./gradlew -q installDist)
  log="$(mktemp -t openhongguo-signer.XXXXXX)"
  "$here/build/install/openhongguo-signer/bin/openhongguo-signer" --port "$port" >"$log" 2>&1 &
  pid=$!
  relay="http://127.0.0.1:$port"
  echo "openhongguo-signer starting (log: $log)"
  for _ in $(seq 1 90); do
    grep -q 'HTTP server listening' "$log" && break
    kill -0 "$pid" 2>/dev/null || { echo "openhongguo-signer exited; see $log" >&2; exit 2; }
    sleep 1
  done
  grep -q 'HTTP server listening' "$log" || { echo "openhongguo-signer did not start in 90s; see $log" >&2; exit 2; }
fi

cd "$here/compare"
go run . -relay "$relay" ${args[@]+"${args[@]}"}
