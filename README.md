# Caddy 自定义构建与 dynamic_proxy 模块

## 源信息
- **服务器**: 192.168.255.201:999 (root)
- **源配置**: /etc/nginx
- **拉取时间**: 2026-05-21

## 仓库内容

运行时配置不纳入版本控制：`Caddyfile` 和 `sites/` 由部署环境或配置同步机制
单独提供，并被 `.gitignore` 忽略。仓库仅保留可复现的 Caddy 构建源码和服务文件。

```
caddy/
├── cmd/caddy/main.go             # 可复现的自定义 Caddy 构建入口
├── dynamic_proxy/                # dynamic_proxy 模块源码与测试
├── go.mod                        # Caddy 与扩展模块的精确依赖
├── Makefile                      # make build / make test
├── deploy.sh                     # 部署前校验并安装 systemd 服务
├── service/                      # systemd 服务与同步监视脚本
├── README.md                     # 本说明文件
└── .gitignore                    # 忽略运行时配置和编译产物
```

## 架构

```
Caddyfile（部署环境提供，不纳入 Git）
  └── import sites/*.caddy
        ├── 静态站点:    直接 file_server
        ├── 反向代理:    直接 reverse_proxy
        └── data.tnzzz.top:  Caddy + dynamic_proxy → 目标服务器
                                           ↑
                              从请求路径提取目标 URL
```

## data.tnzzz.top 特殊处理

该站点使用两个非标准 Caddy 模块实现 NJS 中的两个功能：

| Nginx 功能 | Caddy 方案 |
|---|---|
| 响应体 URL 重写 (NJS body_filter) | ✅ `replace_response` 模块 (xcaddy 编译) |
| 动态 upstream (proxy_pass 正则捕获) | ✅ 本仓库的 `dynamic_proxy` 模块 |

### 构建自定义 Caddy

```bash
# 下载依赖、执行模块测试并构建二进制
make test
make build

# 验证自定义模块已编入二进制
./build/caddy list-modules | grep '^http.handlers.dynamic_proxy$'

# 仅做外部 Caddyfile 的适配检查（不加载证书）
make adapt CADDYFILE=/etc/caddy/Caddyfile
```

`dynamic_proxy` 仅接受 `http` 和 `https` 目标；它将 `prefix` 后的绝对 URL
作为上游地址。该能力本质上是动态代理，必须在站点层做好访问控制，不能把
不可预测的用户输入直接暴露给不受信任的客户端。

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
# 在 CI 或构建主机上生成自定义 Caddy
make test build

# 将构建产物部署到 Caddy 目录
cp build/caddy /etc/caddy/bin/caddy

# Caddyfile 与 sites/ 由独立配置同步流程提供，不从本仓库复制

# 创建日志目录
mkdir -p /var/log/caddy

# 安装并启动 systemd 服务（会验证 dynamic_proxy 模块存在）
bash /etc/caddy/deploy.sh
```
