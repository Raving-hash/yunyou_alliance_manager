#!/bin/bash
# stop.sh — 停止前后端
# 用法：bash stop.sh

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_DIR="${ROOT}/.pids"

stop_pid() {
  local name="$1" pidfile="$2"
  if [[ -f "$pidfile" ]]; then
    local pid
    pid=$(cat "$pidfile")
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" && echo "[${name}] 已停止 PID $pid"
    else
      echo "[${name}] 进程 $pid 不存在"
    fi
    rm -f "$pidfile"
  else
    echo "[${name}] 未找到 PID 文件，跳过"
  fi
}

stop_pid "后端" "${PID_DIR}/backend.pid"
stop_pid "前端" "${PID_DIR}/frontend.pid"
