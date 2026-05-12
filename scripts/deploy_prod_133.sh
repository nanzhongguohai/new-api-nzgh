#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

REMOTE_HOST="${REMOTE_HOST:-38.76.215.133}"
REMOTE_USER="${REMOTE_USER:-root}"
REMOTE_DIR="${REMOTE_DIR:-/root/code/new-api}"
PROJECT_NAME="${PROJECT_NAME:-new-api-prod-10011}"
PUBLIC_PORT="${PUBLIC_PORT:-10011}"

SSH_TARGET="${REMOTE_USER}@${REMOTE_HOST}"
REMOTE_COMPOSE_CMD="docker compose -p \"$PROJECT_NAME\" --env-file \"$REMOTE_DIR/prod.env\" -f \"$REMOTE_DIR/docker-compose.prod.yml\""

cd "$ROOT_DIR"

require_cmd() {
    local cmd="$1"
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "❌ 错误: 未找到 $cmd"
        exit 1
    fi
}

require_cmd ssh
require_cmd rsync
require_cmd bun
require_cmd go
require_cmd curl

if [[ ! -f prod.env ]]; then
    echo "❌ 错误: 未找到本地 prod.env"
    exit 1
fi

echo "------------------------------------------------"
echo "🚀 开始远程部署到 $REMOTE_HOST (Port: $PUBLIC_PORT)"
echo "------------------------------------------------"

echo "🔎 步骤 1/7: 构建前端..."
WEB_APPS=("web/default" "web/classic")
for app_dir in "${WEB_APPS[@]}"; do
    if [[ ! -f "$app_dir/package.json" ]]; then
        echo "⚠️ 警告: $app_dir 不存在，跳过"
        continue
    fi
    (
        cd "$app_dir"
        bun install
        DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION="$(cat "$ROOT_DIR/VERSION")" bun run build
    )
done

echo "🔨 步骤 2/7: 构建后端..."
mkdir -p bin
CGO_ENABLED=0 GOEXPERIMENT=greenteagc go build \
    -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" \
    -o ./bin/new-api-prod main.go

echo "📤 步骤 3/7: 同步部署文件到远端..."
ssh "$SSH_TARGET" "mkdir -p '$REMOTE_DIR/bin' '$REMOTE_DIR/deploy/nginx' '$REMOTE_DIR/logs/app' '$REMOTE_DIR/data'"
rsync -a bin/new-api-prod "$SSH_TARGET:$REMOTE_DIR/bin/"
rsync -a docker-compose.prod.yml prod.env Dockerfile.prod "$SSH_TARGET:$REMOTE_DIR/"
rsync -a deploy/nginx/nginx.conf "$SSH_TARGET:$REMOTE_DIR/deploy/nginx/"

echo "🧪 步骤 4/7: 远端配置校验..."
ssh "$SSH_TARGET" "cd '$REMOTE_DIR' && $REMOTE_COMPOSE_CMD config >/dev/null"

echo "🚚 步骤 5/7: 远端滚动更新..."
ssh "$SSH_TARGET" "cd '$REMOTE_DIR' && $REMOTE_COMPOSE_CMD up -d --build --remove-orphans"

echo "🩺 步骤 6/7: 远端健康检查..."
ssh "$SSH_TARGET" "curl -fsS 'http://127.0.0.1:${PUBLIC_PORT}/health' >/dev/null"

echo "🌍 步骤 7/7: 外网连通性检查..."
curl -fsS "http://${REMOTE_HOST}:${PUBLIC_PORT}/health" >/dev/null

echo "------------------------------------------------"
echo "✅ 远程部署完成。"
echo "🌍 远端监听: http://${REMOTE_HOST}:${PUBLIC_PORT}"
echo "📜 远端日志: ssh ${SSH_TARGET} \"cd '${REMOTE_DIR}' && $REMOTE_COMPOSE_CMD logs -f nginx new-api\""
echo "------------------------------------------------"
