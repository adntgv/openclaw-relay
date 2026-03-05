#!/usr/bin/env bash
set -euo pipefail

cmd=${1:-}
shift || true

case "$cmd" in
  init)
    echo "[relay] init: not implemented yet"
    ;;
  token)
    echo "[relay] token: not implemented yet"
    ;;
  client)
    echo "[relay] client: not implemented yet"
    ;;
  status)
    echo "[relay] status: not implemented yet"
    ;;
  *)
    echo "Usage: relay {init|token|client|status}"
    exit 1
    ;;
 esac
