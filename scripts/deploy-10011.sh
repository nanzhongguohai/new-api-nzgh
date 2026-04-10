#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_DIR="${TARGET_DIR:-/root/nzgh/code/new-api-10011}"
TARGET_DEPLOY_SCRIPT="$TARGET_DIR/scripts/prod-deploy.sh"

if [[ ! -d "$TARGET_DIR" ]]; then
  echo "[deploy-10011] 目标目录不存在: $TARGET_DIR" >&2
  exit 1
fi

if ! command -v rsync >/dev/null 2>&1; then
  echo "[deploy-10011] 未找到 rsync" >&2
  exit 127
fi

echo "[deploy-10011] 同步代码到 $TARGET_DIR"
rsync -a --delete \
  --exclude '.git/' \
  --exclude '.cursor/' \
  --exclude '.env' \
  --exclude '.env.production' \
  --exclude '.air.toml' \
  --exclude 'data/' \
  --exclude 'logs/' \
  --exclude 'logs-prod-10011/' \
  --exclude 'tmp/' \
  --exclude 'bin/' \
  --exclude 'web/node_modules/' \
  --exclude 'web/node_modules.bak.*/' \
  --exclude 'web/dist/' \
  --exclude 'docker-compose.local.yml' \
  --exclude 'new-api.service' \
  --exclude 'scripts/prod-*.sh' \
  "$ROOT_DIR/" "$TARGET_DIR/"

if [[ ! -x "$TARGET_DEPLOY_SCRIPT" ]]; then
  echo "[deploy-10011] 目标部署脚本不存在或不可执行: $TARGET_DEPLOY_SCRIPT" >&2
  exit 1
fi

echo "[deploy-10011] 开始执行目标部署"
exec bash "$TARGET_DEPLOY_SCRIPT"
