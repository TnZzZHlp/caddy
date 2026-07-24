#!/bin/bash
# 监听 Syncthing 同步的 /etc/caddy 变化。
# 配置变更 reload Caddy；二进制变更必须 restart Caddy 才会生效。
WATCH_DIR="/etc/caddy"
DEBOUNCE_SEC=3

inotifywait -m -r -e close_write -e moved_to --format '%w%f' "$WATCH_DIR" 2>/dev/null | while IFS= read -r f; do
    # 忽略 Syncthing 临时文件
    case "$f" in
        *.syncthing*|*.tmp|*/.stfolder*|*/.stversions*) continue ;;
    esac

    # Caddy 在启动时才加载二进制；替换 bin/caddy 后必须重启服务。
    case "$f" in
        "$WATCH_DIR/bin/caddy")
            rel="${f#$WATCH_DIR/}"
            echo "$(date '+%F %T') synced: $rel → restarting caddy"
            sleep "$DEBOUNCE_SEC"
            systemctl restart caddy
            continue
            ;;
    esac

    # 证书或配置文件变化触发无中断 reload。
    case "$f" in
        *.pem|*.crt|*.key|*.cer|*.caddy|Caddyfile)
            rel="${f#$WATCH_DIR/}"
            echo "$(date '+%F %T') synced: $rel → reloading caddy"
            sleep "$DEBOUNCE_SEC"
            systemctl reload caddy 2>/dev/null || true
            ;;
    esac
done
