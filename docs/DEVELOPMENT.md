# Lark Radar 开发指南

> 版本: v2.0
> 日期: 2026-06-09

## 开发环境要求

| 组件 | 版本 | 用途 |
|------|------|------|
| Node.js | 20+ | Web 服务开发 |
| pnpm | 9+ | 包管理 |
| Go | 1.22+ | macOS 数据服务开发 |
| Docker | 20.10+ | Web 容器化测试 |
| lark-cli | latest | 飞书数据同步 |
| Xcode | 15+ | Swift 菜单栏开发 |

## 架构原则

- **macOS 数据服务**: 只暴露 API，不感知 Web 存在
- **Web 服务**: 通过 `DATA_API_URL` 配置连接数据服务
- **两者独立开发、独立部署**

---

## 项目结构

```
project-root/
├── app/                       # Next.js Web 层
│   ├── api/[...path]/         # catch-all 代理
│   ├── page.tsx               # Dashboard
│   ├── groups/                # 群列表/详情
│   ├── topics/                # 话题雷达
│   ├── signals/               # 实时信号
│   └── ...
├── components/                # React 组件
├── lib/                       # Web 层纯工具
│   └── range.ts               # 日期工具
├── apps/data-service/                 # Go 数据层
│   ├── main.go                # HTTP server
│   ├── api/                   # API handlers
│   ├── db/                    # SQLite 连接
│   ├── models/                # 数据模型
│   ├── services/              # 业务逻辑
│   ├── cache/                 # 内存缓存
│   └── utils/                 # 工具函数
├── macos/                     # Swift 菜单栏
│   └── LarkRadarMenu/
├── scripts/                   # 构建脚本
├── docs/                      # 文档
├── Dockerfile                 # Web 镜像
├── docker-compose.yml         # 本地编排
└── package.json
```

---

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/xuhuanxxx/wechat-radar.git
cd wechat-radar
```

### 2. 安装 Web 依赖

```bash
pnpm install
```

### 3. 安装 Go 依赖

```bash
cd apps/data-service
go mod init lark-radar-server  # 首次
go mod tidy
```

### 4. 配置 lark-cli

```bash
# 安装 lark-cli
npm install -g @larksuiteoapi/lark-cli

# 登录
lark-cli auth login --as user
lark-cli doctor
```

---

## 开发模式

### 方式一：直接连接本地数据层（推荐）

**终端 1 - 启动 Go 数据服务:**

```bash
cd apps/data-service
go run main.go

# 输出: [LarkRadar] listening on :3456
```

**终端 2 - 启动 Web 开发服务器:**

```bash
# 直接连接本地数据层
DATA_API_URL=http://localhost:3456 pnpm dev

# 输出: Next.js ready on http://localhost:3000
```

**浏览器访问:** `http://localhost:3000`

### 方式二：Docker 容器化 Web

**终端 1 - Go 数据服务（同上）**

**终端 2 - Docker 运行 Web:**

```bash
# 构建镜像
docker build -t lark-radar-web:dev .

# 运行（连接 host 数据层）
docker run -p 3000:3000 \
  -e DATA_API_URL=http://host.docker.internal:3456 \
  --add-host=host.docker.internal:host-gateway \
  lark-radar-web:dev
```

### 方式三：完整 macOS 体验

```bash
# 构建 Go 二进制
cd apps/data-service
go build -o ../dist/lark-radar-server main.go

# 构建 Swift 菜单栏
cd ../macos/LarkRadarMenu
swift build

# 手动启动 Go 服务（模拟菜单栏行为）
../dist/lark-radar-server --port 3456

# 启动 Web
DATA_API_URL=http://localhost:3456 pnpm dev
```

---

## 开发工作流

### 修改 Web 层

```bash
# 文件: app/, components/, lib/
# 保存后 Next.js HMR 自动刷新

# 测试代理
curl http://localhost:3000/api/health
# 应返回数据层健康状态
```

### 修改数据层

```bash
# 文件: apps/data-service/
# 修改后需重启 Go 服务

# 热重载（使用 air）
cd apps/data-service
air

# 或手动重启
go run main.go
```

### 修改 Swift 菜单栏

```bash
# 文件: macos/LarkRadarMenu/
# 使用 Xcode 打开项目
cd macos
open LarkRadarMenu.xcodeproj

# 或命令行构建
swift build
```

---

## 代码规范

### TypeScript / React

```bash
# 代码检查
pnpm lint

# 类型检查
pnpm tsc --noEmit
```

### Go

```bash
cd apps/data-service

# 格式化
go fmt ./...

# 代码检查
go vet ./...

# 测试
go test ./...

# 构建
go build -o lark-radar-server main.go
```

### Swift

使用 Xcode 内置格式化（Ctrl+I）。

---

## 调试技巧

### Web 层调试

```bash
# 查看代理请求
DATA_API_URL=http://localhost:3456 DEBUG_PROXY=1 pnpm dev

# 浏览器 DevTools
# Network 面板查看 /api/* 请求
```

### 数据层调试

```bash
# 详细日志
cd apps/data-service
LOG_LEVEL=debug go run main.go

# 使用 delve 调试
dlv debug main.go

# 数据库直接查询
sqlite3 ~/.lark-radar/radar.db
```

### 端到端调试

```bash
# 1. 确保数据层运行
curl http://localhost:3456/health

# 2. 确保 Web 代理正常
curl http://localhost:3000/api/health

# 3. 测试具体接口
curl "http://localhost:3000/api/stats?range=week"
```

---

## 测试

### Web 层测试

```bash
# 单元测试
pnpm test

# E2E 测试（Playwright）
pnpm test:e2e
```

### 数据层测试

```bash
cd apps/data-service

# 单元测试
go test ./...

# 集成测试（需要 SQLite）
go test -tags=integration ./...

# 性能测试
go test -bench=. ./...
```

### API 兼容性测试

```bash
# 对比新旧 API 响应
./scripts/api-diff.sh
```

---

## 数据库迁移

### 添加新表

```go
// apps/data-service/db/migrate.go
func migrate(d *sql.DB) {
    d.Exec(`
        CREATE TABLE IF NOT EXISTS new_table (
            id INTEGER PRIMARY KEY,
            name TEXT NOT NULL
        )
    `)
}
```

### 添加新列

```go
func ensureColumn(d *sql.DB, table, column, def string) {
    // 检查列是否存在
    // ALTER TABLE ADD COLUMN
}
```

### 版本控制

```sql
-- meta 表记录版本
INSERT OR REPLACE INTO meta (key, value) VALUES ('schema_version', '2');
```

---

## 提交规范

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型:**
- `feat`: 新功能
- `fix`: 修复
- `perf`: 性能优化
- `refactor`: 重构
- `docs`: 文档
- `test`: 测试
- `chore`: 构建/工具

**示例:**
```
feat(data-service): add link intelligence API

- Implement GET /api/topics/links
- Add article/tool classification
- Cache results for 24h

Closes #123
```

---

## 发布流程

### 1. 版本号更新

```bash
# package.json
npm version 1.1.0

# data-service
echo "1.1.0" > apps/data-service/VERSION
```

### 2. 构建

```bash
# Web
pnpm build

# Go
cd apps/data-service
go build -ldflags="-s -w -X main.version=1.1.0" -o lark-radar-server main.go

# macOS app
./scripts/build-macos-app.sh
```

### 3. 测试

```bash
# 完整测试套件
pnpm test
cd apps/data-service && go test ./...
```

### 4. 打包

```bash
# macOS app
zip -r LarkRadar-1.1.0.app.zip LarkRadar.app

# Web 镜像
docker build -t lark-radar-web:1.1.0 .
docker push lark-radar-web:1.1.0
```

### 5. 发布

```bash
# GitHub Release
git tag v1.1.0
git push origin v1.1.0

# 上传附件
# - LarkRadar-1.1.0.app.zip
# - docker image
```

---

## 常见问题

### Q: Web 代理报 503 错误

A: 检查数据层是否运行：
```bash
curl http://localhost:3456/health
```

### Q: Go 编译报错（CGO）

A: 确保安装了 SQLite 开发库：
```bash
# macOS
brew install sqlite3

# Linux
sudo apt-get install libsqlite3-dev
```

### Q: lark-cli 在 Go 中调用失败

A: 检查 PATH 环境变量：
```bash
which lark-cli
lark-cli doctor
```

### Q: Docker 无法访问 host.docker.internal

A: Linux 需显式添加：
```bash
docker run --add-host=host.docker.internal:host-gateway ...
```

### Q: Swift 菜单栏无法启动 Go 进程

A: 检查 app bundle 中是否包含 Go 二进制：
```bash
ls LarkRadar.app/Contents/Resources/lark-radar-server
```
