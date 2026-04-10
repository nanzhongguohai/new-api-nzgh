#!/usr/bin/env bash
set -euo pipefail

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PORT=10011
ENV_FILE="prod.env"
BINARY_NAME="new-api-10011"

cd "$ROOT_DIR"

echo "------------------------------------------------"
echo "🚀 开始一键部署 (Port: $PORT)"
echo "------------------------------------------------"

# 1. 前端构建
echo "📦 步骤 1/4: 构建前端..."
if [[ -d "web" ]]; then
    cd web
    if ! command -v bun >/dev/null 2>&1; then
        echo "❌ 错误: 未找到 bun，请先安装 bun (https://bun.sh)"
        exit 1
    fi
    bun install
    DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat ../VERSION) bun run build
    cd ..
else
    echo "⚠️ 警告: web 目录不存在，跳过前端构建"
fi

# 2. 后端构建
echo "🔨 步骤 2/4: 构建后端..."
go build -ldflags "-s -w" -o "$BINARY_NAME" main.go

# 3. 停止旧进程
echo "🛑 步骤 3/4: 停止端口 $PORT 的旧进程..."
OLD_PID=$(lsof -t -i:"$PORT" || true)
if [[ -n "$OLD_PID" ]]; then
    echo "Killing process $OLD_PID"
    kill "$OLD_PID" || kill -9 "$OLD_PID"
fi

# 4. 启动新进程
echo "🛫 步骤 4/4: 启动服务 (使用 $ENV_FILE)..."
if [[ ! -f "$ENV_FILE" ]]; then
    echo "❌ 错误: 未找到 $ENV_FILE，请先配置数据库"
    exit 1
fi

# 创建日志目录
mkdir -p logs

# 导出环境变量并以 setsid 方式后台运行，避免 nohup 在部分环境里退出后子进程未稳定常驻
setsid bash -lc "cd '$ROOT_DIR' && export \$(grep -v '^#' '$ENV_FILE' | xargs) && exec './$BINARY_NAME' >> 'logs/prod-10011.log' 2>&1 </dev/null" >/dev/null 2>&1 &

echo "------------------------------------------------"
echo "✅ 部署完成！"
echo "🌍 访问地址: http://localhost:$PORT"
echo "📜 查看日志: tail -f logs/prod-10011.log"
echo "------------------------------------------------"
