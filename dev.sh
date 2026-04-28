#!/bin/bash
# dev.sh — 一键启动前后端（开发模式）
# 用法：bash dev.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_DIR="${ROOT}/.pids"
mkdir -p "${PID_DIR}"

BACKEND_BIN="${PID_DIR}/yunyoumanager_dev"
BACKEND_PID="${PID_DIR}/backend.pid"
FRONTEND_PID="${PID_DIR}/frontend.pid"
BACKEND_LOG="${PID_DIR}/backend.log"
FRONTEND_LOG="${PID_DIR}/frontend.log"

# ── 停旧进程 ────────────────────────────────────────────────────────────────
stop_pid() {
  local name="$1" pidfile="$2"
  if [[ -f "$pidfile" ]]; then
    local pid
    pid=$(cat "$pidfile")
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid"
      # 等待进程真正退出（最多3秒）
      for _ in 1 2 3; do
        kill -0 "$pid" 2>/dev/null || break
        sleep 1
      done
      kill -0 "$pid" 2>/dev/null && kill -9 "$pid" 2>/dev/null || true
      echo "[${name}] 已停止 PID $pid"
    fi
    rm -f "$pidfile"
  fi
}

stop_pid "后端" "${BACKEND_PID}"
stop_pid "前端" "${FRONTEND_PID}"

# 端口兜底：如果还有进程占用端口，强制清理
kill_port() {
  local port="$1"
  local pids
  pids=$(lsof -ti:"$port" 2>/dev/null || true)
  if [[ -n "$pids" ]]; then
    echo "[清理] 端口 $port 被占用，强制释放..."
    echo "$pids" | xargs kill -9 2>/dev/null || true
  fi
}
kill_port 8080
kill_port 5173

# ── 构建后端 ────────────────────────────────────────────────────────────────
echo "[后端] 编译中..."
cd "${ROOT}"
if ! go build -o "${BACKEND_BIN}" . 2>&1 | tee /dev/stderr; then
  echo "[后端] 编译失败，退出"
  exit 1
fi

# ── 启动后端 ────────────────────────────────────────────────────────────────
echo "[后端] 启动..."
"${BACKEND_BIN}" > "${BACKEND_LOG}" 2>&1 &
echo $! > "${BACKEND_PID}"
echo "[后端] PID=$(cat ${BACKEND_PID})  日志: ${BACKEND_LOG}"

# ── 启动前端 ────────────────────────────────────────────────────────────────
echo "[前端] 启动 npm run dev..."
cd "${ROOT}/frontend"
npm run dev > "${FRONTEND_LOG}" 2>&1 &
echo $! > "${FRONTEND_PID}"
echo "[前端] PID=$(cat ${FRONTEND_PID})  日志: ${FRONTEND_LOG}"

echo ""
echo "后端: http://localhost:8080"
echo "前端: http://localhost:5173"
echo ""
echo "停止全部: bash stop.sh"
echo "查看后端日志: tail -f ${BACKEND_LOG}"
echo "查看前端日志: tail -f ${FRONTEND_LOG}"
