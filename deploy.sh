#!/bin/bash
# deploy.sh — 一键部署 yunyoumanager 到 Debian 12
# 用法：sudo bash deploy.sh
# 需要在项目根目录执行，需要 root 权限

set -euo pipefail

# ── 配置 ────────────────────────────────────────────────────────────────────
APP_NAME="yunyoumanager"
APP_USER="yunyou"
APP_PORT="8080"
NGINX_PORT="80"
INSTALL_DIR="/opt/${APP_NAME}"
DATA_DIR="/var/lib/${APP_NAME}"
BINARY="${INSTALL_DIR}/${APP_NAME}"
FRONTEND_DIR="/var/www/${APP_NAME}"
GO_VERSION="1.22.4"
NODE_MAJOR="20"

# ── 颜色 ────────────────────────────────────────────────────────────────────
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# ── 检查 root ────────────────────────────────────────────────────────────────
[[ $EUID -ne 0 ]] && error "请使用 root 权限运行：sudo bash deploy.sh"

# 记录项目源码目录（脚本所在目录）
SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
info "项目源码目录：${SRC_DIR}"

# ────────────────────────────────────────────────────────────────────────────
# 1. 安装系统依赖
# ────────────────────────────────────────────────────────────────────────────
info "更新 apt 并安装基础依赖…"
apt-get update -q
apt-get install -y -q curl wget gcc sqlite3 nginx

# ── Go ───────────────────────────────────────────────────────────────────────
if ! command -v go &>/dev/null || [[ "$(go version | awk '{print $3}')" < "go${GO_VERSION}" ]]; then
  info "安装 Go ${GO_VERSION}…"
  ARCH=$(dpkg --print-architecture)
  case $ARCH in
    amd64) GO_ARCH="amd64" ;;
    arm64) GO_ARCH="arm64" ;;
    *)     error "不支持的架构：${ARCH}" ;;
  esac
  GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
  wget -q "https://go.dev/dl/${GO_TAR}" -O "/tmp/${GO_TAR}"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "/tmp/${GO_TAR}"
  rm "/tmp/${GO_TAR}"
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
  info "Go $(go version) 安装完成"
else
  info "Go 已就绪：$(go version)"
fi

export PATH=$PATH:/usr/local/go/bin

# ── Node.js ──────────────────────────────────────────────────────────────────
if ! command -v node &>/dev/null || [[ "$(node --version | cut -d. -f1 | tr -d 'v')" -lt "$NODE_MAJOR" ]]; then
  info "安装 Node.js ${NODE_MAJOR}.x…"
  curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | bash -
  apt-get install -y -q nodejs
  info "Node.js $(node --version) 安装完成"
else
  info "Node.js 已就绪：$(node --version)"
fi

# ────────────────────────────────────────────────────────────────────────────
# 2. 创建系统用户
# ────────────────────────────────────────────────────────────────────────────
if ! id "${APP_USER}" &>/dev/null; then
  info "创建系统用户 ${APP_USER}…"
  useradd --system --no-create-home --shell /usr/sbin/nologin "${APP_USER}"
fi

# ────────────────────────────────────────────────────────────────────────────
# 3. 构建前端
# ────────────────────────────────────────────────────────────────────────────
info "构建前端…"
cd "${SRC_DIR}/frontend"
npm install --silent
npm run build --silent

info "部署前端静态文件到 ${FRONTEND_DIR}…"
mkdir -p "${FRONTEND_DIR}"
cp -r dist/. "${FRONTEND_DIR}/"

# ────────────────────────────────────────────────────────────────────────────
# 4. 构建后端
# ────────────────────────────────────────────────────────────────────────────
info "构建后端…"
cd "${SRC_DIR}"
export GOPATH="/tmp/go-build-${APP_NAME}"
go mod download
CGO_ENABLED=1 go build -ldflags="-s -w" -o "${APP_NAME}" .

# ────────────────────────────────────────────────────────────────────────────
# 5. 安装二进制
# ────────────────────────────────────────────────────────────────────────────
info "安装到 ${INSTALL_DIR}…"
mkdir -p "${INSTALL_DIR}"
cp "${APP_NAME}" "${BINARY}"
chmod +x "${BINARY}"

mkdir -p "${DATA_DIR}"
chown -R "${APP_USER}:${APP_USER}" "${DATA_DIR}"

# ────────────────────────────────────────────────────────────────────────────
# 6. systemd 服务
# ────────────────────────────────────────────────────────────────────────────
info "配置 systemd 服务…"
cat > "/etc/systemd/system/${APP_NAME}.service" <<EOF
[Unit]
Description=云游联盟管理系统后端
After=network.target

[Service]
Type=simple
User=${APP_USER}
WorkingDirectory=${DATA_DIR}
ExecStart=${BINARY}
Restart=on-failure
RestartSec=5
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

# config.go 里的 DBPath 是相对路径 "data/yunyou.db"
# WorkingDirectory=${DATA_DIR}，所以数据库在 ${DATA_DIR}/data/yunyou.db
mkdir -p "${DATA_DIR}/data"
chown -R "${APP_USER}:${APP_USER}" "${DATA_DIR}"

systemctl daemon-reload
systemctl enable "${APP_NAME}"
systemctl restart "${APP_NAME}"
info "后端服务已启动"

# ────────────────────────────────────────────────────────────────────────────
# 7. nginx 配置
# ────────────────────────────────────────────────────────────────────────────
info "配置 nginx…"
cat > "/etc/nginx/sites-available/${APP_NAME}" <<EOF
server {
    listen ${NGINX_PORT};
    server_name _;

    root ${FRONTEND_DIR};
    index index.html;

    # 前端 SPA 路由
    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # 后端 API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:${APP_PORT};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_read_timeout 60s;
    }

    # 上传大小限制（CSV 文件通常很小，10MB 足够）
    client_max_body_size 10M;
}
EOF

# 启用站点，移除默认站点
ln -sf "/etc/nginx/sites-available/${APP_NAME}" "/etc/nginx/sites-enabled/${APP_NAME}"
rm -f /etc/nginx/sites-enabled/default

nginx -t
systemctl enable nginx
systemctl restart nginx
info "nginx 已配置并重启"

# ────────────────────────────────────────────────────────────────────────────
# 完成
# ────────────────────────────────────────────────────────────────────────────
SERVER_IP=$(hostname -I | awk '{print $1}')
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  部署成功！${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "  访问地址：http://${SERVER_IP}"
echo -e "  后端端口：${APP_PORT}（仅本机）"
echo -e "  数据目录：${DATA_DIR}/data/"
echo -e "  查看日志：journalctl -u ${APP_NAME} -f"
echo -e "  重启服务：systemctl restart ${APP_NAME}"
echo ""
