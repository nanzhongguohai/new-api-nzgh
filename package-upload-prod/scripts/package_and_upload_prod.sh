#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$SKILL_DIR/.." && pwd)"

REMOTE_HOST="${REMOTE_HOST:-38.76.215.133}"
REMOTE_USER="${REMOTE_USER:-root}"
REMOTE_DIR="${REMOTE_DIR:-/root/code/new-api/release}"
BASE_IMAGE="${BASE_IMAGE:-new-api-prod-10011-new-api:latest}"
PROJECT_NAME="${PROJECT_NAME:-new-api-prod-10011}"
PUBLIC_PORT="${PUBLIC_PORT:-10011}"
RELEASE_TIMESTAMP="${RELEASE_TIMESTAMP:-$(date +%Y%m%d%H%M%S)}"
RELEASE_NAME="${RELEASE_NAME:-new-api-prod-10011-${RELEASE_TIMESTAMP}}"
RELEASE_DIR="$ROOT_DIR/release/$RELEASE_NAME"
TAR_PATH="$ROOT_DIR/release/$RELEASE_NAME.tar.gz"
SSH_TARGET="${REMOTE_USER}@${REMOTE_HOST}"

require_cmd() {
  local cmd="$1"
  command -v "$cmd" >/dev/null 2>&1 || {
    echo "❌ missing command: $cmd" >&2
    exit 1
  }
}

run_with_proxy() {
  if command -v with_proxy >/dev/null 2>&1; then
    with_proxy "$@"
  else
    "$@"
  fi
}

cleanup_container() {
  if [[ -n "${CONTAINER_ID:-}" ]]; then
    docker rm -f "$CONTAINER_ID" >/dev/null 2>&1 || true
  fi
}

trap cleanup_container EXIT

require_cmd bun
require_cmd go
require_cmd docker
require_cmd ssh
require_cmd rsync
require_cmd tar
require_cmd sha256sum

cd "$ROOT_DIR"
mkdir -p "$ROOT_DIR/bin" "$RELEASE_DIR"

for app_dir in web/default web/classic; do
  if [[ -f "$app_dir/package.json" ]]; then
    (cd "$app_dir" && bun install)
    (cd "$app_dir" && DISABLE_ESLINT_PLUGIN=true VITE_REACT_APP_VERSION="$(cat "$ROOT_DIR/VERSION")" bun run build)
  fi
done

CGO_ENABLED=0 GOEXPERIMENT=greenteagc go build \
  -ldflags "-s -w -X github.com/QuantumNous/new-api/common.Version=$(cat VERSION)" \
  -o bin/new-api-prod main.go

if docker image inspect "$BASE_IMAGE" >/dev/null 2>&1; then
  CONTAINER_ID="$(docker create "$BASE_IMAGE")"
  docker cp bin/new-api-prod "$CONTAINER_ID:/new-api"
  docker commit "$CONTAINER_ID" "$PROJECT_NAME:$RELEASE_TIMESTAMP" >/dev/null
  docker save "$PROJECT_NAME:$RELEASE_TIMESTAMP" -o "$RELEASE_DIR/new-api-prod-10011-image.tar"
else
  run_with_proxy docker build -f Dockerfile.prod -t "$PROJECT_NAME:$RELEASE_TIMESTAMP" .
  docker save "$PROJECT_NAME:$RELEASE_TIMESTAMP" -o "$RELEASE_DIR/new-api-prod-10011-image.tar"
fi

cp Dockerfile.prod "$RELEASE_DIR/Dockerfile.prod"
cp docker-compose.prod.yml "$RELEASE_DIR/docker-compose.prod.yml"
cp prod.env.example "$RELEASE_DIR/prod.env.example"
mkdir -p "$RELEASE_DIR/deploy/nginx"
cp deploy/nginx/nginx.conf "$RELEASE_DIR/deploy/nginx/nginx.conf"

cat > "$RELEASE_DIR/INSTALL.md" <<'EOF'
1. 复制 `prod.env.example` 为 `prod.env`，按实际数据库和 Redis 地址修改。
2. 执行：`docker load -i new-api-prod-10011-image.tar`
3. 在同目录运行：`docker compose -p new-api-prod-10011 --env-file prod.env -f docker-compose.prod.yml up -d`
4. 访问：`http://<服务器IP>:10011/health`
EOF

(cd "$RELEASE_DIR" && sha256sum \
  new-api-prod-10011-image.tar \
  docker-compose.prod.yml \
  Dockerfile.prod \
  INSTALL.md \
  prod.env.example \
  deploy/nginx/nginx.conf \
  > SHA256SUMS)

tar -czf "$TAR_PATH" -C "$ROOT_DIR/release" "$RELEASE_NAME"
sha256sum "$TAR_PATH" >> "$RELEASE_DIR/SHA256SUMS"

if [[ "${SKIP_UPLOAD:-0}" == "1" ]]; then
  echo "skip upload: $RELEASE_DIR"
  echo "skip upload tarball: $TAR_PATH"
  exit 0
fi

ssh "$SSH_TARGET" "mkdir -p '$REMOTE_DIR'"
rsync -a "$RELEASE_DIR" "$TAR_PATH" "$SSH_TARGET:$REMOTE_DIR/"
ssh "$SSH_TARGET" "ls -lh '$REMOTE_DIR/$RELEASE_NAME' '$REMOTE_DIR/$RELEASE_NAME.tar.gz'"
ssh "$SSH_TARGET" "sha256sum '$REMOTE_DIR/$RELEASE_NAME.tar.gz' '$REMOTE_DIR/$RELEASE_NAME/new-api-prod-10011-image.tar'"

echo "release: $RELEASE_DIR"
echo "tarball: $TAR_PATH"
