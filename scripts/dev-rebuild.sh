#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_DIR="$ROOT_DIR/web"
STAMP_FILE="$WEB_DIR/dist/.build-stamp"
BUN_BIN="${BUN_BIN:-}"

if [[ -z "$BUN_BIN" ]]; then
  if command -v bun >/dev/null 2>&1; then
    BUN_BIN="$(command -v bun)"
  elif [[ -x "/root/.bun/bin/bun" ]]; then
    BUN_BIN="/root/.bun/bin/bun"
  else
    echo "[air] 未找到 bun，可通过 BUN_BIN 指定路径" >&2
    exit 127
  fi
fi

needs_web_build() {
  if [[ ! -f "$STAMP_FILE" ]]; then
    return 0
  fi

  local latest_src
  latest_src="$(
    find \
      "$WEB_DIR/src" \
      "$WEB_DIR/public" \
      -type f \
      \( -name '*.js' -o -name '*.jsx' -o -name '*.ts' -o -name '*.tsx' -o -name '*.css' -o -name '*.json' \) \
      -newer "$STAMP_FILE" \
      -print \
      -quit 2>/dev/null || true
  )"
  if [[ -n "$latest_src" ]]; then
    return 0
  fi

  local extra_files=(
    "$WEB_DIR/index.html"
    "$WEB_DIR/package.json"
    "$WEB_DIR/bun.lockb"
  )
  local file
  for file in "${extra_files[@]}"; do
    if [[ -f "$file" && "$file" -nt "$STAMP_FILE" ]]; then
      return 0
    fi
  done

  return 1
}

mkdir -p "$ROOT_DIR/bin" "$ROOT_DIR/logs" "$ROOT_DIR/tmp/air"

if needs_web_build; then
  echo "[air] 检测到前端变更，开始构建 web"
  (
    cd "$WEB_DIR"
    "$BUN_BIN" run build
  )
  mkdir -p "$(dirname "$STAMP_FILE")"
  touch "$STAMP_FILE"
fi

echo "[air] 开始构建后端"
(
  cd "$ROOT_DIR"
  go build -o ./bin/new-api-hot .
)
