# Lark Radar 数据迁移计划

> 版本: v2.0
> 日期: 2026-06-09
> 从: Next.js 全栈应用
> 到: Go macOS 数据服务 + Next.js Web 服务（完全解耦）

## 迁移概述

将现有 TypeScript 数据层迁移到 Go，保持 API 格式 100% 兼容，前端无需修改。

**关键原则**:
- macOS 数据服务只暴露 API，不感知 Web 存在
- Web 服务通过 `DATA_API_URL` 配置连接数据服务
- 两者完全独立，可独立部署升级

## 迁移范围

### 需要迁移的模块

| 模块 | 当前位置 | 目标位置 | 复杂度 |
|------|----------|----------|--------|
| HTTP 路由 | `app/api/` | `go-server/api/` | 低 |
| SQLite 连接 | `lib/db.ts` | `go-server/db/` | 低 |
| 数据模型 | `lib/*.ts` (interface) | `go-server/models/` | 低 |
| 飞书调用 | `lib/lark.ts` | `go-server/services/lark.go` | 中 |
| 同步引擎 | `lib/lark-sync.ts` | `go-server/services/lark_sync.go` | 高 |
| 消息存储 | `lib/messages-store.ts` | `go-server/services/` | 中 |
| 统计聚合 | `lib/stats-aggregator.ts` | `go-server/services/` | 中 |
| @提及检测 | `lib/mentions.ts` | `go-server/services/mentions.go` | 中 |
| Dashboard 情报 | `lib/dashboard-intelligence.ts` | `go-server/services/intelligence.go` | 高 |
| 链接情报 | `lib/link-intelligence.ts` | `go-server/services/link_intel.go` | 高 |
| 话题聚合 | `lib/topics.ts` | `go-server/services/topics.go` | 高 |
| 群分类 | `lib/group-classifier.ts` | `go-server/services/classifier.go` | 低 |
| 缓存层 | `lib/cache.ts` | `go-server/cache/` | 低 |
| 配置管理 | `lib/config.ts` | `go-server/services/config.go` | 低 |
| 日期工具 | `lib/range.ts` | `go-server/utils/range.go` | 低 |

### 不需要迁移的模块

| 模块 | 说明 |
|------|------|
| `components/` | React 组件，纯前端 |
| `app/*.tsx` | Next.js 页面，纯前端 |
| `lib/range.ts` | Web 层保留副本（纯函数） |

## 迁移策略

### 阶段一：基础设施（第 1 天）

**目标**: 搭建 Go 项目框架，实现基础 HTTP 服务和数据库连接

**任务清单:**
- [ ] 创建 `go-server/` 目录结构
- [ ] 初始化 Go module
- [ ] 实现 `main.go` HTTP server
- [ ] 实现 `db/db.go` SQLite 连接（复制现有 pragmas）
- [ ] 实现 `db/migrate.go` 表结构迁移（复制现有 SQL）
- [ ] 实现 `db/seed.go` 默认数据种子
- [ ] 实现 `models/` 核心结构体
- [ ] 实现 `utils/json.go` 响应工具
- [ ] 实现 `cache/cache.go` 内存缓存

**验证:**
```bash
cd go-server
go run main.go
# 应启动成功，监听 :3456
curl http://localhost:3456/health
# 应返回 {"ok":true}
```

### 阶段二：核心 API（第 2 天）

**目标**: 实现最常用的 API，支持 Dashboard 基本功能

**任务清单:**
- [ ] `api/setup.go` - GET/POST /api/setup
- [ ] `api/stats.go` - GET /api/stats
- [ ] `api/sessions.go` - GET /api/sessions
- [ ] `api/messages.go` - GET /api/group/:id
- [ ] `api/mentions.go` - GET/POST /api/mentions
- [ ] `api/dates.go` - GET /api/dates
- [ ] `api/dbinfo.go` - GET /api/dbinfo
- [ ] `api/search.go` - GET /api/search

**验证:**
```bash
# 测试各接口
curl "http://localhost:3456/api/stats?range=week"
curl "http://localhost:3456/api/sessions"
# 响应格式应与现有 API 一致
```

### 阶段三：飞书同步（第 3 天）

**目标**: 实现飞书数据同步，这是最关键的功能

**任务清单:**
- [ ] `services/lark.go` - lark-cli 调用封装
- [ ] `services/lark_sync.go` - 同步引擎（从 TS 翻译）
- [ ] `api/sync.go` - POST /api/lark/sync（含 SSE）
- [ ] 消息解析（text/post/card/image 等类型）
- [ ] 批量插入（事务优化）
- [ ] 增量同步（sync_state 表）

**验证:**
```bash
# 触发同步
curl -X POST http://localhost:3456/api/lark/sync \
  -H "content-type: application/json" \
  -d '{"days_back": 1}'

# 检查数据库是否有新消息
sqlite3 ~/.lark-radar/radar.db "SELECT COUNT(*) FROM messages"
```

### 阶段四：高级功能（第 4 天）

**目标**: 实现 Dashboard 情报、话题、链接等高级功能

**任务清单:**
- [ ] `services/intelligence.go` - Dashboard 情报生成
- [ ] `api/topics.go` - GET/POST /api/topics
- [ ] `services/topics.go` - 话题聚合（codex CLI 调用）
- [ ] `api/links.go` - GET /api/topics/links
- [ ] `services/link_intel.go` - 链接情报
- [ ] `api/groups.go` - CRUD /api/groups
- [ ] `api/group-tags.go` - GET/POST /api/group-tags
- [ ] `api/ai-classify.go` - GET/POST /api/ai-classify
- [ ] `api/lark.go` - GET/POST /api/lark/*
- [ ] `api/message-links.go` - /api/message-links/*

**验证:**
```bash
# 测试话题构建
curl -X POST http://localhost:3456/api/topics/build \
  -H "content-type: application/json" \
  -d '{"date":"2026-06-09"}'

# 测试链接情报
curl "http://localhost:3456/api/topics/links?date=2026-06-09"
```

### 阶段五：Web 代理层（第 5 天上午）

**目标**: 实现 Next.js catch-all 代理，切换流量到 Go 服务

**任务清单:**
- [ ] 创建 `app/api/[...path]/route.ts`
- [ ] 实现 HTTP 代理（GET/POST/DELETE）
- [ ] 实现 SSE 流式透传
- [ ] 添加错误处理（数据层不可用时）
- [ ] 删除旧的 `app/api/` 独立 routes

**验证:**
```bash
# 启动 Go 服务
cd go-server && go run main.go

# 启动 Web
DATA_API_URL=http://localhost:3456 pnpm dev

# 测试代理
curl http://localhost:3000/api/health
curl "http://localhost:3000/api/stats?range=week"
# 应返回与直接访问 :3456 相同的结果
```

### 阶段六：清理与验证（第 5 天下半天）

**任务清单:**
- [ ] 删除 Web 层残留数据模块（`lib/db.ts`, `lib/cache.ts` 等）
- [ ] 更新 `package.json` scripts
- [ ] 构建验证 `pnpm build`
- [ ] Docker 构建验证
- [ ] 端到端功能测试
- [ ] 性能对比测试

**验证:**
```bash
# 完整构建
pnpm build

# Docker 构建
docker build -t lark-radar-web:test .

# 启动测试
docker run -p 3000:3000 \
  -e DATA_API_URL=http://host.docker.internal:3456 \
  lark-radar-web:test
```

## 代码映射参考

### TypeScript → Go 类型映射

| TypeScript | Go |
|-----------|-----|
| `string` | `string` |
| `number` | `int` / `int64` / `float64` |
| `boolean` | `bool` |
| `Date` | `time.Time` |
| `Array<T>` | `[]T` |
| `Map<K, V>` | `map[K]V` |
| `Set<T>` | `map[T]struct{}` |
| `interface` | `struct` |
| `type` alias | `type` alias |
| `Promise<T>` | 返回值 `T, error` |
| `async/await` | 顺序执行（Go 并发用 goroutine） |

### 关键代码片段映射

**SQLite 查询:**
```typescript
// TS
const rows = db().prepare('SELECT * FROM messages WHERE date = ?').all(date) as MessageRow[];

// Go
rows, err := db.Query("SELECT * FROM messages WHERE date = ?", date)
```

**缓存:**
```typescript
// TS (node-cache)
const cached = cache.get<T>(key);
cache.set(key, value, ttl);

// Go (sync.Map)
var cache sync.Map
cache.Store(key, value)
value, ok := cache.Load(key)
```

**HTTP 路由:**
```typescript
// TS (Next.js)
export async function GET(req: NextRequest) {
  return NextResponse.json(data);
}

// Go (标准库)
func handleStats(w http.ResponseWriter, r *http.Request) {
  json.NewEncoder(w).Encode(data)
}
```

**SSE 流式:**
```typescript
// TS
const stream = new ReadableStream({
  start(controller) {
    controller.enqueue(enc.encode(`data: ${JSON.stringify(obj)}\n\n`));
  }
});

// Go
func handleSyncStream(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "text/event-stream")
  flusher := w.(http.Flusher)
  fmt.Fprintf(w, "data: %s\n\n", json)
  flusher.Flush()
}
```

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| API 格式不兼容 | 高 | 逐接口对比测试，使用相同 JSON 结构 |
| SQLite 行为差异 | 中 | 使用相同 pragmas，充分测试 WAL 模式 |
| 性能下降 | 中 | Go 通常更快，但仍需基准测试 |
| 同步引擎 bug | 高 | 保留旧版本可回滚，充分测试 |
| 开发时间超期 | 中 | 分阶段交付，每阶段可独立验证 |

## 回滚计划

如果迁移出现问题，可快速回滚到旧架构：

1. **保留旧代码**: 迁移期间不删除 `app/api/` 和 `lib/` 的旧文件，只是不再使用
2. **Git 分支**: 在 `go-migration` 分支开发，可随时切回 `main`
3. **数据库兼容**: 不修改数据库 schema，新旧版本共用同一数据库
4. **快速切换**: 只需修改 Web 代理配置，指向旧 API 或新 API

```bash
# 回滚到旧架构
git checkout main
pnpm install
pnpm build
pnpm start
```

## 验收标准

- [ ] 所有 20+ API 接口响应格式与旧版本一致
- [ ] Dashboard 页面正常加载，数据正确
- [ ] 飞书同步功能正常，消息入库
- [ ] 话题构建功能正常
- [ ] 链接情报功能正常
- [ ] 搜索功能正常
- [ ] 群分类功能正常
- [ ] Docker 容器可正常运行
- [ ] macOS app 可正常打包运行
- [ ] 性能不低于旧版本（或更好）
