# Lark Radar 部署指南

> 版本: v2.0
> 日期: 2026-06-09

## 概述

Lark Radar 由两个独立组件组成：

1. **macOS 数据服务**: 仅运行在用户 Mac 上，负责 SQLite + lark-cli 同步
2. **Web 服务**: 无状态纯前端，可部署到公网/内网/本地，通过 HTTP/WebSocket API 消费数据

**关键原则**:
- macOS 数据服务不感知 Web 服务的存在、位置、入口
- Web 服务通过**用户在前端页面输入的地址**连接数据服务
- 两者完全独立部署，独立生命周期

**DATA_API_URL 配置方式**:
- 首次访问 Web 时，弹出配置页面让用户输入自己 Mac 的数据服务地址
- 地址存储在浏览器 localStorage 中
- 用户可随时在设置页面修改地址

---

## 1. macOS 数据服务部署

### 1.1 前置要求

- macOS 12+ (Monterey)
- lark-cli 已安装并登录
- Go 1.22+（仅开发/构建时需要）

### 1.2 构建 Go 二进制

```bash
cd apps/data-service

# 本地开发
go build -o lark-radar-server main.go

# 生产构建（优化）
go build -ldflags="-s -w" -o lark-radar-server main.go

# 验证
./lark-radar-server --help
```

### 1.3 直接运行

```bash
# 默认配置
./lark-radar-server

# 指定端口和数据目录
./lark-radar-server --port 3456 --data-dir ~/.lark-radar

# 后台运行
nohup ./lark-radar-server > server.log 2>&1 &
```

### 1.4 构建 macOS App

```bash
# 1. 构建 Go 二进制
cd apps/data-service
go build -ldflags="-s -w" -o ../dist/lark-radar-server main.go

# 2. 构建 Swift 菜单栏
cd ../macos/LarkRadarMenu
swift build -c release

# 3. 打包 app
./scripts/build-macos-app.sh
```

### 1.5 macOS App Bundle 结构

```
LarkRadar.app/
├── Contents/
│   ├── Info.plist
│   ├── MacOS/
│   │   └── LarkRadarMenu          # Swift 菜单栏可执行文件
│   └── Resources/
│       └── lark-radar-server      # Go 数据服务二进制
```

### 1.6 用户安装流程

1. 下载 `LarkRadar.app.zip`
2. 解压并拖入 `/Applications`
3. 首次启动：
   - 点击菜单栏图标
   - 选择「打开设置」配置昵称
   - 数据服务自动启动（:3456）
4. **通过浏览器访问 Web 服务**（Web 入口由用户/管理员提供）

### 1.7 菜单栏功能

- **状态显示**: 🟢 运行中 / 🔴 已停止
- **立即同步**: 手动触发飞书数据同步
- **打开数据目录**: 在 Finder 中打开 `~/.lark-radar/`
- **设置**: 修改配置（昵称、同步间隔等）
- **退出**: 停止数据服务并退出

**注意**: 菜单栏没有"打开 Web"选项——不感知 Web 存在

---

## 2. Web 服务部署

### 2.1 前置要求

- Docker 20.10+
- 知道数据服务地址（用户 Mac 的 IP:3456）

### 2.2 Dockerfile

```dockerfile
# Dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install
COPY . .
RUN pnpm build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000

# DATA_API_URL 在运行时才确定
# ENV DATA_API_URL=http://...

# 仅复制 standalone 构建产物
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

EXPOSE 3000
CMD ["node", "server.js"]
```

### 2.3 构建镜像

```bash
# 构建
docker build -t lark-radar-web:latest .

# 标记版本
docker tag lark-radar-web:latest lark-radar-web:1.0.0
```

### 2.4 运行容器

#### 场景 A: Web 和数据都在本地（开发）

```bash
# macOS 上同时运行数据服务和 Web
docker run -d \
  --name lark-radar-web \
  -p 3000:3000 \
  -e DATA_API_URL=http://host.docker.internal:3456 \
  --add-host=host.docker.internal:host-gateway \
  lark-radar-web:latest
```

#### 场景 B: Web 在公网服务器，数据在用户 Mac

```bash
# 公网服务器上运行 Web
# 用户 Mac 需配置端口转发或 VPN，使公网可访问 :3456

# 方式 1: 用户 Mac 使用 ngrok/frp 暴露端口
# ngrok http 3456 → 获得 https://xxx.ngrok.io

# Web 服务配置
docker run -d \
  --name lark-radar-web \
  -p 3000:3000 \
  -e DATA_API_URL=https://xxx.ngrok.io \
  lark-radar-web:latest

# 方式 2: 公司 VPN/内网穿透
# 用户 Mac 在 VPN 中获得固定内网 IP
docker run -d \
  --name lark-radar-web \
  -p 3000:3000 \
  -e DATA_API_URL=http://192.168.100.50:3456 \
  lark-radar-web:latest
```

#### 场景 C: Web 在公司内网服务器

```bash
# 内网服务器
docker run -d \
  --name lark-radar-web \
  -p 3000:3000 \
  -e DATA_API_URL=http://user-mac.local:3456 \
  lark-radar-web:latest

# 用户 Mac 需在同一内网，或 VPN 接入
```

#### docker-compose.yml

```yaml
version: '3.8'

services:
  web:
    image: lark-radar-web:latest
    ports:
      - "3000:3000"
    environment:
      # 根据场景修改此地址
      - DATA_API_URL=http://host.docker.internal:3456
    extra_hosts:
      - "host.docker.internal:host-gateway"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:3000"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### 2.5 验证部署

```bash
# 检查容器状态
docker ps

# 查看日志
docker logs lark-radar-web

# 测试代理（从 Web 服务器）
curl http://localhost:3000/api/health
# 应转发到数据层，返回健康状态
```

---

## 3. 部署拓扑示例

### 拓扑 A: 个人本地使用

```
┌─────────────────────────────────────────────┐
│                用户 Mac                       │
│  ┌─────────────────┐  ┌─────────────────┐  │
│  │  Web 服务        │  │  macOS 数据服务  │  │
│  │  (Docker :3000) │  │  (Go :3456)     │  │
│  │                 │◄─┤                 │  │
│  └─────────────────┘  └─────────────────┘  │
│           ↑                                │
│      浏览器 localhost:3000                  │
└─────────────────────────────────────────────┘

DATA_API_URL=http://host.docker.internal:3456
```

### 拓扑 B: 公网 Web + 本地数据

```
┌─────────────────────────┐         ┌─────────────────────────┐
│      公网服务器          │         │        用户 Mac          │
│  ┌───────────────────┐  │         │  ┌───────────────────┐  │
│  │   Web 服务         │  │  HTTP   │  │   macOS 数据服务   │  │
│  │   lark.app        │◄─┼─────────┼──┤   :3456           │  │
│  │   (Docker)        │  │         │  │                   │  │
│  └───────────────────┘  │         │  └───────────────────┘  │
│           ↑             │         │                         │
│      用户浏览器         │         │  ngrok/frp/VPN 暴露端口  │
└─────────────────────────┘         └─────────────────────────┘

DATA_API_URL=https://user1.ngrok.io (每个用户不同)
```

### 拓扑 C: 公司内网 Web + 本地数据

```
┌─────────────────────────┐         ┌─────────────────────────┐
│      公司内网服务器       │         │        用户 Mac          │
│  ┌───────────────────┐  │         │  ┌───────────────────┐  │
│  │   Web 服务         │  │  HTTP   │  │   macOS 数据服务   │  │
│  │   lark.local      │◄─┼─────────┼──┤   :3456           │  │
│  │   (Docker)        │  │         │  │                   │  │
│  └───────────────────┘  │         │  └───────────────────┘  │
│           ↑             │         │                         │
│      员工浏览器         │         │  同一内网/VPN           │
└─────────────────────────┘         └─────────────────────────┘

DATA_API_URL=http://user-mac.local:3456
```

### 拓扑 D: 多用户共享 Web

```
                         ┌─────────────────────────┐
                         │      公网服务器          │
                         │  ┌───────────────────┐  │
                         │  │   Web 服务         │  │
                         │  │   lark.app        │  │
                         │  └───────────────────┘  │
                         │           ↑             │
                         └───────────┼─────────────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              │                      │                      │
              ↓ HTTP                 ↓ HTTP                 ↓ HTTP
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│   用户 1 Mac     │      │   用户 2 Mac     │      │   用户 3 Mac     │
│  ┌───────────┐  │      │  ┌───────────┐  │      │  ┌───────────┐  │
│  │ 数据服务   │  │      │  │ 数据服务   │  │      │  │ 数据服务   │  │
│  │ :3456     │  │      │  │ :3456     │  │      │  │ :3456     │  │
│  └───────────┘  │      │  └───────────┘  │      │  └───────────┘  │
└─────────────────┘      └─────────────────┘      └─────────────────┘

每个用户通过不同 URL 访问同一 Web，但连接自己的数据服务
Web 前端可通过 localStorage 记住用户的 DATA_API_URL
```

---

## 4. 配置参考

### Web 服务（前端配置）

| 配置项 | 存储位置 | 说明 |
|--------|----------|------|
| `DATA_API_URL` | `localStorage` | 用户 Mac 的数据服务地址 |

**首次配置流程:**
1. 用户浏览器打开 Web 服务
2. 弹出配置页面，提示输入数据服务地址
3. 用户输入 `http://192.168.1.100:3456`（自己 Mac 的地址）
4. 点击"测试连接"验证连通性
5. 保存到 localStorage，进入主界面

**修改配置:**
- 设置页面 → 数据服务地址 → 修改 → 测试 → 保存

### Web 服务（服务器端环境变量）

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `PORT` | 否 | 3000 | Web 服务端口 |
| `NODE_ENV` | 否 | production | 运行环境 |
| `WS_ENABLED` | 否 | true | 是否启用 WebSocket |

### macOS 数据服务

| 变量/参数 | 必填 | 默认值 | 说明 |
|-----------|------|--------|------|
| `--port` / `LARK_RADAR_PORT` | 否 | 3456 | HTTP 端口 |
| `--data-dir` / `LARK_RADAR_DATA_DIR` | 否 | ~/.lark-radar | 数据目录 |
| `--log-level` | 否 | info | 日志级别 |
| `--cors-origins` | 否 | * | CORS 允许来源 |

---

## 5. 网络配置

### 5.1 本地开发

```bash
# Web 和数据都在本地
# 数据服务监听 127.0.0.1:3456（默认）
# Web 通过 host.docker.internal 访问
```

### 5.2 公网访问（用户 Mac 暴露端口）

```bash
# 方式 1: ngrok
ngrok http 3456
# 获得 https://xxx.ngrok.io

# 方式 2: frp（自建穿透）
# frpc.ini
[radar]
type = http
local_port = 3456
custom_domain = user1.radar.company.com

# 方式 3: 路由器端口转发
# 公网 IP:3456 → 内网 Mac:3456
```

### 5.3 VPN/内网

```bash
# 用户 Mac 加入公司 VPN
# 获得固定内网 IP: 10.0.x.x
# Web 服务器通过内网 IP 访问
```

---

## 6. 升级流程

### 6.1 升级 macOS 数据服务

```bash
# 1. 停止旧版本
# 菜单栏选择「退出」

# 2. 备份数据
cp -r ~/.lark-radar ~/.lark-radar.backup

# 3. 替换 app
# 下载新版本 LarkRadar.app，覆盖安装

# 4. 启动新版本
# 菜单栏点击启动
```

### 6.2 升级 Web 服务

```bash
# 1. 拉取新镜像
docker pull lark-radar-web:latest

# 2. 重启容器
docker-compose down
docker-compose up -d
```

### 6.3 回滚

```bash
# 数据服务：恢复备份
cp -r ~/.lark-radar.backup ~/.lark-radar

# Web 服务：使用旧镜像
docker run -d ... lark-radar-web:1.0.0
```

---

## 7. 故障排查

### 7.1 Web 无法连接数据服务

```bash
# 从 Web 服务器测试连通性
curl $DATA_API_URL/health

# 检查数据服务是否运行（用户 Mac）
curl http://localhost:3456/health

# 检查网络连通性
ping user-mac-ip

# 检查防火墙
# macOS: 系统设置 → 网络 → 防火墙
```

### 7.2 跨域错误

```bash
# 检查数据服务 CORS 配置
curl -H "Origin: https://lark.app" \
  -I http://user-mac:3456/api/stats

# 应返回 Access-Control-Allow-Origin: *
```

### 7.3 数据服务无法启动

```bash
# 检查端口占用
lsof -i :3456

# 检查数据目录权限
ls -la ~/.lark-radar

# 查看日志
./lark-radar-server 2>&1 | tee server.log
```

---

## 8. 性能调优

### 8.1 Go 数据服务

```bash
# 设置 GOMAXPROCS
export GOMAXPROCS=4

# 限制内存
# 建议：512MB 足够
```

### 8.2 Web 服务

```bash
# 限制资源
docker run -m 512m --cpus=1 ...

# 使用 CDN 缓存静态资源
```

### 8.3 数据库

- WAL 模式已默认启用
- 定期 VACUUM（每月一次）
