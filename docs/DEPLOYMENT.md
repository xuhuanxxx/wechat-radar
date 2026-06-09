# Lark Radar 部署指南

> 版本: v1.0
> 日期: 2026-06-09

## 概述

Lark Radar 采用三层分离架构：

1. **Web 层**: Docker 容器化部署
2. **数据层**: Go 二进制，内嵌于 macOS app
3. **UI 层**: Swift 菜单栏应用

本文档描述各组件的部署方式。

---

## 1. 数据层部署（macOS）

### 1.1 前置要求

- macOS 12+ (Monterey)
- lark-cli 已安装并登录
- Go 1.22+（仅开发/构建时需要）

### 1.2 构建 Go 二进制

```bash
cd go-server

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
cd go-server
go build -ldflags="-s -w" -o ../dist/lark-radar-server main.go

# 2. 构建 Swift 菜单栏
cd ../macos/LarkRadarMenu
swift build -c release

# 3. 打包 app
# 使用 Xcode Archive 或脚本
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
   - 数据服务自动启动
4. 浏览器访问 `http://localhost:3000`（Web 容器）

---

## 2. Web 层部署（Docker）

### 2.1 前置要求

- Docker 20.10+
- 数据层已在运行（macOS 上 `:3456`）

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
ENV DATA_API_URL=http://host.docker.internal:3456
ENV PORT=3000

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

#### macOS 开发环境

```bash
docker run -d \
  --name lark-radar-web \
  -p 3000:3000 \
  -e DATA_API_URL=http://host.docker.internal:3456 \
  --add-host=host.docker.internal:host-gateway \
  lark-radar-web:latest
```

#### Linux 环境

```bash
# 数据层监听 0.0.0.0:3456（需配置）
docker run -d \
  --name lark-radar-web \
  -p 3000:3000 \
  -e DATA_API_URL=http://192.168.1.100:3456 \
  lark-radar-web:latest
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

# 测试代理
curl http://localhost:3000/api/health
# 应转发到数据层，返回健康状态
```

---

## 3. 完整部署拓扑

### 3.1 本地开发

```
┌─────────────────┐
│  macOS Host     │
│  ┌───────────┐  │
│  │ Go 服务    │  │  :3456
│  │ (终端 1)   │  │
│  └───────────┘  │
│  ┌───────────┐  │
│  │ Web 容器   │  │  :3000
│  │ (终端 2)   │  │
│  └───────────┘  │
│  ┌───────────┐  │
│  │ 浏览器     │  │
│  │ localhost │  │
│  └───────────┘  │
└─────────────────┘
```

### 3.2 生产部署（单用户）

```
┌─────────────────┐     ┌─────────────────┐
│   macOS 电脑     │     │  Docker 主机    │
│  ┌───────────┐  │     │  ┌───────────┐  │
│  │ LarkRadar │  │◄────┤  │ Web 容器   │  │  :3000
│  │ App       │  │ HTTP │  │           │  │
│  │ (Go 3456) │  │      │  └───────────┘  │
│  └───────────┘  │      └─────────────────┘
└─────────────────┘
```

### 3.3 生产部署（多用户/服务器）

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   用户 1        │     │   用户 2        │     │   用户 N        │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │ LarkRadar │  │     │  │ LarkRadar │  │     │  │ LarkRadar │  │
│  │ App       │  │     │  │ App       │  │     │  │ App       │  │
│  │ :3456     │  │     │  │ :3456     │  │     │  │ :3456     │  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │ HTTPS/WSS
                                 ↓
                    ┌─────────────────────────┐
                    │    服务器集群            │
                    │  ┌─────────────────┐   │
                    │  │  Nginx / LB     │   │
                    │  └────────┬────────┘   │
                    │           │            │
                    │  ┌────────┴────────┐   │
                    │  │  Web 容器 x N   │   │
                    │  │  (无状态)        │   │
                    │  └─────────────────┘   │
                    └─────────────────────────┘
```

---

## 4. 环境变量参考

### Web 容器

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `DATA_API_URL` | 是 | - | 数据层服务地址 |
| `PORT` | 否 | 3000 | Web 服务端口 |
| `NODE_ENV` | 否 | production | 运行环境 |

### Go 数据层

| 变量/参数 | 必填 | 默认值 | 说明 |
|-----------|------|--------|------|
| `--port` / `LARK_RADAR_PORT` | 否 | 3456 | HTTP 端口 |
| `--data-dir` / `LARK_RADAR_DATA_DIR` | 否 | ~/.lark-radar | 数据目录 |
| `--log-level` | 否 | info | 日志级别 |

---

## 5. 升级流程

### 5.1 升级数据层

```bash
# 1. 停止旧版本
# 菜单栏选择「退出」或 kill 进程

# 2. 备份数据
cp -r ~/.lark-radar ~/.lark-radar.backup

# 3. 替换二进制
# 下载新版本 LarkRadar.app，覆盖安装

# 4. 启动新版本
# 菜单栏点击启动，自动迁移数据库
```

### 5.2 升级 Web 层

```bash
# 1. 拉取新镜像
docker pull lark-radar-web:latest

# 2. 重启容器
docker-compose down
docker-compose up -d

# 或单容器
docker stop lark-radar-web
docker rm lark-radar-web
docker run -d ... lark-radar-web:latest
```

### 5.3 回滚

```bash
# 数据层：恢复备份
cp -r ~/.lark-radar.backup ~/.lark-radar

# Web 层：使用旧镜像标签
docker run -d ... lark-radar-web:1.0.0
```

---

## 6. 故障排查

### 6.1 Web 无法连接数据层

```bash
# 检查数据层是否运行
curl http://localhost:3456/health

# 检查 Web 容器网络
docker exec lark-radar-web wget -qO- http://host.docker.internal:3456/health

# Linux 需确保数据层监听 0.0.0.0
# macOS 确保使用 host.docker.internal
```

### 6.2 数据层无法启动

```bash
# 检查端口占用
lsof -i :3456

# 检查数据目录权限
ls -la ~/.lark-radar

# 查看日志
./lark-radar-server 2>&1 | tee server.log
```

### 6.3 lark-cli 未认证

```bash
# 在 macOS 终端执行
lark-cli auth login --as user
lark-cli doctor
```

---

## 7. 性能调优

### 7.1 Go 数据层

```bash
# 设置 GOMAXPROCS（容器内）
export GOMAXPROCS=4

# 限制内存（systemd/cgroup）
# 建议：512MB 足够
```

### 7.2 Web 容器

```bash
# 限制资源
docker run -m 512m --cpus=1 ...

# 使用缓存层（CDN）
# 静态资源缓存 1 年
```

### 7.3 数据库

- WAL 模式已默认启用
- 定期 VACUUM（每月一次）
- 大数据量时考虑分表（按日期）
