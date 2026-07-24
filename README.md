# Nginx → Caddy 配置转换说明

## 源信息
- **服务器**: 192.168.255.201:999 (root)
- **源配置**: /etc/nginx
- **拉取时间**: 2026-05-21

## 目录结构
```
caddy-config/
├── Caddyfile                     # 主配置（全局选项 + 代码片段 + import）
├── caddy                         # Caddy v2.11.3 二进制（含 replace_response 模块）
├── README.md                     # 本说明文件
├── sites/                        # 按域名拆分的站点配置
│   ├── tnzzz-top.caddy           # 主站（静态）
│   ├── api-tnzzz-top.caddy       # API 网关 + 订阅文件
│   ├── api-800008-best.caddy     # API 网关
│   ├── ai-tnzzz-top.caddy        # AI 流式代理
│   ├── mcp-tnzzz-top.caddy       # MCP 代理
│   ├── todos-tnzzz-top.caddy     # SPA + API
│   ├── grafana-tnzzz-top.caddy   # Grafana 面板
│   ├── loki-tnzzz-top.caddy      # Loki 日志（Token 认证）
│   ├── zt-tnzzz-top.caddy        # ZeroTier 面板
│   ├── ryuri-tnzzz-top.caddy     # Ryuri 服务
│   ├── file-tnzzz-top.caddy      # 文件服务
│   ├── img-tnzzz-top.caddy       # 图片 CDN
│   ├── img-800008-best.caddy     # 图片 CDN
│   ├── webhook-800008-best.caddy # Webhook
│   ├── dmit-la-corona.caddy      # 静态探针
│   ├── dmit-la-wee.caddy         # 静态探针
│   └── data-tnzzz-top.caddy      # URL 重写代理（含 body rewrite）
└── data-proxy-helper/
    ├── main.go                   # 动态上游代理源码
    └── data-proxy-helper         # 编译好的二进制

nginx-config/                     # 原始 Nginx 配置（从服务器拉取）
├── nginx.conf
├── conf.d/  (18 个站点 .conf)
└── njs/proxy_rewrite.js
```

## 架构

```
Caddyfile (全局 + snippets)
  └── import sites/*.caddy
        ├── 静态站点:    直接 file_server
        ├── 反向代理:    直接 reverse_proxy
        └── data.tnzzz.top:  Caddy → data-proxy-helper:9090 → 目标服务器
                                                    ↑
                                        解析路径提取目标 URL
```

## data.tnzzz.top 特殊处理

该站点使用 NJS 脚本实现两个功能，Caddy 标准版不支持：

| Nginx 功能 | Caddy 方案 |
|---|---|
| 响应体 URL 重写 (NJS body_filter) | ✅ `replace_response` 模块 (xcaddy 编译) |
| 动态 upstream (proxy_pass 正则捕获) | ✅ `data-proxy-helper` (独立 Go 程序) |

### 启动方式

```bash
# 1. 启动动态代理助手
./data-proxy-helper -addr 127.0.0.1:9090 &

# 2. 启动 Caddy
./caddy run --config ./Caddyfile
```

### Body rewrite 策略

使用 3-pass 替换补偿 RE2 不支持零宽断言：

1. **Pass 1** — 清理自引用 URL 上的代理前缀（预处理）
2. **Pass 2** — 所有外部绝对 URL 添加代理前缀 (regex)
3. **Pass 3** — 修复被误加前缀的自引用 URL + 防御递归前缀

## 转换对照表

| Nginx | Caddy | 说明 |
|---|---|---|
| `listen 443 ssl` | 自动 | Caddy 默认 HTTPS |
| `ssl_certificate` | `tls <cert> <key>` | 自定义证书 |
| `server_name` | 站点标签 | 域名写在站点名位置 |
| `root` | `root * <path>` | 静态文件根 |
| `proxy_pass http://up` | `reverse_proxy <host>:<port>` | 反向代理 |
| `add_header X Y` | `header X Y` | 响应头 |
| `proxy_buffering off` | `flush_interval -1` | 关闭缓冲 |
| `proxy_read_timeout 600s` | `transport http { read_timeout 600s }` | 超时 |
| `gzip on` / `http2 on` / `http3 on` | 自动 | Caddy 默认 |
| `if ($http_upgrade)` | 自动 | WebSocket 自动处理 |
| `location /path` | `handle /path/*` | 路径匹配 |
| `location ~ regex` | `@name { path_regexp ... }` | 正则匹配 |
| `proxy_hide_header` | `header >Name` | 删除响应头 |
| `include conf.d/*.conf` | `import sites/*.caddy` | 拆分配置 |

## 部署

```bash
# 复制到 Caddy 目录
cp Caddyfile /etc/caddy/Caddyfile
cp -r sites/ /etc/caddy/sites/
cp caddy /usr/local/bin/caddy
cp data-proxy-helper/data-proxy-helper /usr/local/bin/

# 创建日志目录
mkdir -p /var/log/caddy

# 启动 helper
data-proxy-helper -addr 127.0.0.1:9090 &

# 运行 Caddy
caddy run --config /etc/caddy/Caddyfile
```
