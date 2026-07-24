#!/bin/bash
# Caddy 部署（文件由 Syncthing 同步，本脚本只装服务）
set -e

echo "=== 停止 Nginx ==="
systemctl disable nginx.service --now 2>/dev/null || true
systemctl disable nginx-auto-reload.service --now 2>/dev/null || true

echo "=== 安装 ==="
chmod +x /etc/caddy/bin/caddy /etc/caddy/service/caddy-sync-watch.sh
mkdir -p /var/log/caddy

cp /etc/caddy/service/caddy.service /etc/systemd/system/
cp /etc/caddy/service/caddy-sync-watch.service /etc/systemd/system/
systemctl daemon-reload

echo "=== 校验 ==="
/etc/caddy/bin/caddy validate --config /etc/caddy/Caddyfile

echo "=== 启动 ==="
systemctl enable --now caddy
systemctl enable --now caddy-sync-watch

echo ""
echo "✓ 完成"
systemctl status caddy --no-pager -l
