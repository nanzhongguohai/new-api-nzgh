#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="$ROOT_DIR/prod.env"
COMPOSE_FILE="$ROOT_DIR/docker-compose.prod.yml"
PROJECT_NAME="new-api-prod-10011"
LEGACY_BINARY_NAME="new-api-10011"
DEFAULT_PROXY_URL="http://127.0.0.1:7890"
RUNTIME_BINARY_PATH="bin/new-api-prod"

cd "$ROOT_DIR"

export HTTP_PROXY="${HTTP_PROXY:-$DEFAULT_PROXY_URL}"
export HTTPS_PROXY="${HTTPS_PROXY:-$DEFAULT_PROXY_URL}"
export ALL_PROXY="${ALL_PROXY:-$DEFAULT_PROXY_URL}"
export http_proxy="${http_proxy:-$HTTP_PROXY}"
export https_proxy="${https_proxy:-$HTTPS_PROXY}"
export all_proxy="${all_proxy:-$ALL_PROXY}"

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD=(docker compose -p "$PROJECT_NAME" --env-file "$ENV_FILE" -f "$COMPOSE_FILE")
else
    echo "❌ 错误: 未找到 docker compose，请先安装 Docker Compose v2"
    exit 1
fi

if [[ ! -f "$ENV_FILE" ]]; then
    echo "❌ 错误: 未找到 $ENV_FILE，请先复制 prod.env.example 并完成配置"
    exit 1
fi

PUBLIC_PORT="$(awk -F= '$1=="PORT" {gsub(/[[:space:]]/, "", $2); print $2}' "$ENV_FILE" | tail -n 1)"
PUBLIC_PORT="${PUBLIC_PORT:-10011}"

echo "------------------------------------------------"
echo "🚀 开始项目内 Nginx 隔离部署 (Port: $PUBLIC_PORT)"
echo "------------------------------------------------"
echo "🌐 使用代理: $HTTP_PROXY"

echo "🔎 步骤 1/8: 校验环境..."
if grep -Eq '^(SQL_DSN|LOG_SQL_DSN)=.*(localhost|127\.0\.0\.1)' "$ENV_FILE"; then
    echo "❌ 错误: 检测到数据库 DSN 仍指向 localhost/127.0.0.1。"
    echo "   容器内访问宿主机数据库时，请改为 host.docker.internal。"
    echo "   参考文档: docs/installation/isolated-nginx-prod.md"
    exit 1
fi

if grep -Eq '^REDIS_CONN_STRING=redis://(localhost|127\.0\.0\.1)' "$ENV_FILE"; then
    echo "❌ 错误: 检测到 Redis 连接仍指向 localhost/127.0.0.1。"
    echo "   容器内访问宿主机 Redis 时，请改为 host.docker.internal。"
    echo "   参考文档: docs/installation/isolated-nginx-prod.md"
    exit 1
fi

"${COMPOSE_CMD[@]}" config >/dev/null

echo "📦 步骤 2/8: 构建前端..."
if ! command -v bun >/dev/null 2>&1; then
    echo "❌ 错误: 未找到 bun，请先安装 bun"
    exit 1
fi
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

echo "🔨 步骤 3/8: 构建后端..."
mkdir -p "$(dirname "$RUNTIME_BINARY_PATH")"
CGO_ENABLED=0 GOEXPERIMENT=greenteagc go build \
    -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" \
    -o "$RUNTIME_BINARY_PATH" main.go

echo "📁 步骤 4/8: 准备目录..."
mkdir -p data logs/app deploy/nginx

echo "🧹 步骤 5/8: 停止旧的 Compose 栈..."
"${COMPOSE_CMD[@]}" down --remove-orphans || true

echo "🛑 步骤 6/8: 清理旧的宿主机直跑进程..."
mapfile -t LISTEN_PIDS < <(lsof -t -iTCP:"$PUBLIC_PORT" -sTCP:LISTEN 2>/dev/null | awk '!seen[$0]++')
if (( ${#LISTEN_PIDS[@]} > 0 )); then
    for pid in "${LISTEN_PIDS[@]}"; do
        cmdline="$(ps -o cmd= -p "$pid" 2>/dev/null || true)"
        if [[ "$cmdline" == *"$LEGACY_BINARY_NAME"* || "$cmdline" == *"./new-api"* || "$cmdline" == *"/new-api"* ]]; then
            echo "Stopping legacy process $pid: $cmdline"
            kill "$pid" 2>/dev/null || true
        else
            echo "❌ 错误: 端口 $PUBLIC_PORT 仍被其他进程占用: $cmdline"
            echo "   请先释放该端口后再部署。"
            exit 1
        fi
    done
fi

sleep 1

echo "🚚 步骤 7/8: 封装并启动项目专属 Nginx + new-api..."
"${COMPOSE_CMD[@]}" up -d --build --remove-orphans

echo "🩺 步骤 8/8: 等待健康检查..."
for _ in $(seq 1 60); do
    if curl -fsS "http://127.0.0.1:${PUBLIC_PORT}/health" >/dev/null 2>&1; then
        echo "服务健康检查通过。"
        break
    fi
    sleep 2
done

if ! curl -fsS "http://127.0.0.1:${PUBLIC_PORT}/health" >/dev/null 2>&1; then
    echo "❌ 错误: 健康检查未通过，请查看容器日志。"
    "${COMPOSE_CMD[@]}" ps
    exit 1
fi

echo "------------------------------------------------"
echo "✅ 部署完成。"
echo "🌍 项目内 Nginx 已监听: http://127.0.0.1:${PUBLIC_PORT}"
echo "📜 查看日志: ${COMPOSE_CMD[*]} logs -f nginx new-api"
echo "📘 迁移说明: docs/installation/isolated-nginx-prod.md"
echo "------------------------------------------------"
