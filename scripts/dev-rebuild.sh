#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_DIR="$ROOT_DIR/web"
WEB_APPS=("default" "classic")
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
  local app_dir="$1"
  local stamp_file="$2"

  if [[ ! -f "$stamp_file" ]]; then
    return 0
  fi

  local latest_src
  latest_src="$(
    find \
      "$app_dir/src" \
      "$app_dir/public" \
      -type f \
      \( -name '*.js' -o -name '*.jsx' -o -name '*.ts' -o -name '*.tsx' -o -name '*.css' -o -name '*.json' \) \
      -newer "$stamp_file" \
      -print \
      -quit 2>/dev/null || true
  )"
  if [[ -n "$latest_src" ]]; then
    return 0
  fi

  local extra_files=(
    "$app_dir/index.html"
    "$app_dir/package.json"
    "$app_dir/bun.lock"
    "$app_dir/bun.lockb"
    "$app_dir/vite.config.js"
    "$app_dir/rsbuild.config.ts"
  )
  local file
  for file in "${extra_files[@]}"; do
    if [[ -f "$file" && "$file" -nt "$stamp_file" ]]; then
      return 0
    fi
  done

  return 1
}

mkdir -p "$ROOT_DIR/bin" "$ROOT_DIR/logs" "$ROOT_DIR/tmp/air"

for app in "${WEB_APPS[@]}"; do
  app_dir="$WEB_DIR/$app"
  stamp_file="$app_dir/dist/.build-stamp"
  if [[ ! -f "$app_dir/package.json" ]]; then
    continue
  fi
  if needs_web_build "$app_dir" "$stamp_file"; then
    echo "[air] 检测到 $app 前端变更，开始构建 web/$app"
    (
      cd "$app_dir"
      "$BUN_BIN" install
      DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION="$(cat "$ROOT_DIR/VERSION")" "$BUN_BIN" run build
    )
    mkdir -p "$(dirname "$stamp_file")"
    touch "$stamp_file"
  fi
done

echo "[air] 开始构建后端"
(
  cd "$ROOT_DIR"
  go build -o ./bin/new-api-hot .
)
