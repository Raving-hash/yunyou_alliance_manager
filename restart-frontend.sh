#!/bin/bash
# restart-frontend.sh — 重启前端
# 用法：bash restart-frontend.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_DIR="${ROOT}/.pids"
mkdir -p "${PID_DIR}"

FRONTEND_PID="${PID_DIR}/frontend.pid"
FRONTEND_LOG="${PID_DIR}/frontend.log"

if [[ -f "${FRONTEND_PID}" ]]; then
  pid=$(cat "${FRONTEND_PID}")
  if kill -0 "$pid" 2>/dev/null; then
    kill "$pid"
    echo "[前端] 已停止 PID $pid"
  fi
  rm -f "${FRONTEND_PID}"
fi

cd "${ROOT}/frontend"
echo "[前端] 重启 npm run dev ..."
npm run dev > "${FRONTEND_LOG}" 2>&1 &
echo $! > "${FRONTEND_PID}"
echo "[前端] PID=$(cat ${FRONTEND_PID})  日志: ${FRONTEND_LOG}"
echo "[前端] http://localhost:5173"
