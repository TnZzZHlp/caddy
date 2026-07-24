#!/bin/bash
# 监听 Syncthing 同步的 /etc/caddy 变化，自动 reload Caddy
WATCH_DIR="/etc/caddy"
DEBOUNCE_SEC=3

inotifywait -m -r -e close_write -e moved_to --format '%w%f' "$WATCH_DIR" 2>/dev/null | while read f; do
    # 忽略 Syncthing 临时文件
    case "$f" in
        *.syncthing*|*.tmp|*/.stfolder*|*/.stversions*) continue ;;
    esac

    # 证书或配置文件变化都触发 reload
    case "$f" in
        *.pem|*.crt|*.key|*.cer|*.caddy|Caddyfile)
            rel="${f#$WATCH_DIR/}"
            echo "$(date '+%F %T') synced: $rel → reloading caddy"
            sleep $DEBOUNCE_SEC
            systemctl reload caddy 2>/dev/null || true
            ;;
    esac
done
