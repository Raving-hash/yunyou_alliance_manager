#!/bin/bash
# restart-backend.sh — 重启后端
# 用法：bash restart-backend.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_DIR="${ROOT}/.pids"
mkdir -p "${PID_DIR}"

BACKEND_BIN="${PID_DIR}/yunyoumanager_dev"
BACKEND_PID="${PID_DIR}/backend.pid"
BACKEND_LOG="${PID_DIR}/backend.log"

if [[ -f "${BACKEND_PID}" ]]; then
  pid=$(cat "${BACKEND_PID}")
  if kill -0 "$pid" 2>/dev/null; then
    kill "$pid"
    for _ in 1 2 3; do
      kill -0 "$pid" 2>/dev/null || break
      sleep 1
    done
    kill -0 "$pid" 2>/dev/null && kill -9 "$pid" 2>/dev/null || true
    echo "[后端] 已停止 PID $pid"
  fi
  rm -f "${BACKEND_PID}"
fi

# 端口兜底清理
pids=$(lsof -ti:8080 2>/dev/null || true)
[[ -n "$pids" ]] && { echo "[清理] 端口 8080 强制释放..."; echo "$pids" | xargs kill -9 2>/dev/null || true; }

echo "[后端] 编译中..."
cd "${ROOT}"
if ! go build -o "${BACKEND_BIN}" . 2>&1 | tee /dev/stderr; then
  echo "[后端] 编译失败"
  exit 1
fi

echo "[后端] 启动..."
"${BACKEND_BIN}" > "${BACKEND_LOG}" 2>&1 &
echo $! > "${BACKEND_PID}"
echo "[后端] PID=$(cat ${BACKEND_PID})  日志: ${BACKEND_LOG}"
