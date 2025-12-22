# Implementation Guide

完整的实施路线图和开发指南。

**Last Updated**: 2025-12-22
**Current Status**: 60% Complete - Core Engine Done

---

## 快速导航

- **MVP 上线路线图**: 见 [MVP_ROADMAP.md](MVP_ROADMAP.md) - 详细的 MVP 实施计划
- **架构设计**: 见 [docs/plans/](docs/plans/) - 完整的设计文档
- **变更日志**: 见 [CHANGELOG.md](CHANGELOG.md) - 版本历史

---

## Phase 1: Core Engine ✅ 已完成 (60%)

### 已实现模块

| 模块 | 文件 | 代码量 | 测试 | 状态 |
|------|------|--------|------|------|
| Schemas | `pkg/schemas/` | 400 行 | - | ✅ |
| Operators | `pkg/operators/` | 800 行 | - | ✅ |
| Planner | `pkg/planner/` | 1,400 行 | 43 个 | ✅ |
| Executor | `pkg/executor/` | 600 行 | 14 个 | ✅ |

**核心能力**:
- 声明式 JobSpec 定义
- 可扩展的操作符系统（trim, scale）
- DAG 构建和拓扑排序
- 元数据传播和资源估算
- FFmpeg 命令生成和执行
- 实时进度解析

### 数据流

```
JobSpec (JSON)
    ↓
[Validator] - 参数验证、类型转换
    ↓
[Planner] - DAG 构建、元数据传播、资源估算
    ↓
[Builder] - 生成 FFmpeg filter_complex 命令
    ↓
[Executor] - 执行进程、解析进度
    ↓
Output Files + Progress
```

---

## Phase 2: MVP 完成 📋 进行中 (40%)

**目标**: 实现可运行的单机版服务

详细计划见 [MVP_ROADMAP.md](MVP_ROADMAP.md)

### 核心任务

1. **Media Prober** (10%) - 🔴 最高优先级
   - ffprobe 包装器
   - 解析输入文件元数据
   - 支持本地和远程文件

2. **Store Module** (10%) - 🔴 高优先级
   - 作业状态存储（内存/SQLite）
   - CRUD 接口
   - 进度更新

3. **API Server** (10%) - 🔴 高优先级
   - REST API（提交、查询、取消作业）
   - HTTP 处理器
   - 中间件（日志、CORS、认证）

4. **错误处理增强** (5%) - 🟡 中优先级
   - FFmpeg 错误解析
   - 错误分类
   - 重试策略

5. **配置管理** (3%) - 🟡 中优先级
   - 环境变量和配置文件
   - FFmpeg 路径、端口等

6. **基础监控** (2%) - 🟢 低优先级
   - Prometheus metrics
   - 基础指标

---

## Phase 3: 生产级增强 📋 待定 (未来)

### 3.1 分布式状态管理
**优先级**: 未来
**参考**: `docs/plans/distributed-state-management-design.md`

- PostgreSQL 数据库层
- Redis 作业队列
- 分布式锁
- 状态机

### 3.2 Worker 协调
**优先级**: 未来

- Worker 注册和心跳
- 作业分发
- 故障恢复
- Watchdog 进程

### 3.3 完整错误处理
**优先级**: 未来
**参考**: `docs/plans/error-handling-design.md`

- 50+ 错误代码分类
- FFmpeg 错误解析（15+ 模式）
- 重试策略（指数退避）
- 熔断器

### 3.4 高级 API 功能
**优先级**: 未来
**参考**: `docs/plans/api-interface-design.md`

- JWT 认证
- 速率限制（令牌桶）
- Webhook 通知
- WebSocket 实时更新

### 3.5 更多操作符
**优先级**: 未来

**音频操作符**:
- `loudnorm` - EBU R128 响度标准化
- `mix` - 音频混合
- `volume`, `fade` - 音量和淡入淡出

**视频操作符**:
- `crop`, `rotate`, `fps`, `pad`

**合成操作符**:
- `concat` - 视频拼接
- `overlay` - 叠加图像/文字
- `drawtext` - 文字渲染
- `thumbnail` - 缩略图生成
- `waveform` - 音频波形

---

## 开发工作流

### 1. 阅读设计文档
在编码前，先理解模块设计：
- 架构设计: `docs/plans/2025-12-14-media-pipeline-architecture-design.md`
- 模块设计: `docs/plans/schemas-detailed-design.md` 等

### 2. TDD 方法
1. 编写测试用例（`*_test.go`）
2. 实现功能代码
3. 运行测试 `go test ./...`
4. 重构优化

### 3. 测试策略

**单元测试**:
```go
func TestTrimOperator(t *testing.T) {
    op := &builtin.TrimOperator{}
    params := map[string]interface{}{
        "start": "00:00:10",
        "duration": "00:05:00",
    }
    err := op.ValidateParams(params)
    assert.NoError(t, err)
}
```

**集成测试**:
- 完整作业流程
- 真实媒体文件
- 错误场景

**端到端测试**:
- API → Prober → Planner → Executor
- 真实 FFmpeg 执行
- 进度跟踪验证

### 4. 代码质量
- 所有公共函数有文档注释
- 错误处理完善
- 日志输出清晰
- 避免魔法数字

---

## 依赖管理

### 当前依赖
```bash
# 已在 go.mod
go get github.com/google/uuid
```

### MVP 所需依赖
```bash
# HTTP 路由
go get github.com/gorilla/mux

# SQLite（可选，用于 Store）
go get github.com/mattn/go-sqlite3
```

### 未来依赖（生产级）
```bash
# 数据库
go get github.com/lib/pq                    # PostgreSQL
go get github.com/redis/go-redis/v9         # Redis

# 监控
go get github.com/prometheus/client_golang  # Metrics
go get go.opentelemetry.io/otel            # Tracing
```

---

## 配置管理

创建 `internal/config/config.go`:

```go
type Config struct {
    Server   ServerConfig
    FFmpeg   FFmpegConfig
    Storage  StorageConfig
    Database DatabaseConfig  // MVP 可选
}

type ServerConfig struct {
    Host string `env:"HOST" default:"0.0.0.0"`
    Port int    `env:"PORT" default:"8080"`
}

type FFmpegConfig struct {
    BinPath string `env:"FFMPEG_PATH" default:"ffmpeg"`
    TempDir string `env:"TEMP_DIR" default:"/tmp"`
}

type StorageConfig struct {
    Type   string `env:"STORAGE_TYPE" default:"memory"` // memory|sqlite|s3
    Path   string `env:"STORAGE_PATH" default:"./data"`
}
```

---

## 部署

### MVP 单机部署

**Dockerfile**:
```dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o api cmd/api/main.go

FROM alpine:latest
RUN apk add --no-cache ffmpeg
COPY --from=builder /app/api /usr/local/bin/
EXPOSE 8080
CMD ["api"]
```

**运行**:
```bash
docker build -t media-pipeline:mvp .
docker run -p 8080:8080 media-pipeline:mvp
```

### 生产级部署（未来）

**docker-compose.yml**:
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: media_pipeline
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"

  redis:
    image: redis:7
    ports:
      - "6379:6379"

  api:
    build: .
    command: /app/api
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis

  worker:
    build: .
    command: /app/worker
    deploy:
      replicas: 3
    depends_on:
      - postgres
      - redis
```

---

## 成功标准

### MVP 阶段（当前目标）
- ✅ 通过 REST API 提交 trim+scale 作业
- ✅ 查询作业状态和进度
- ✅ FFmpeg 执行成功
- ✅ 错误有明确提示
- ✅ 一条命令启动服务

### 生产级阶段（未来）
- ✅ 水平扩展多个 Worker
- ✅ 作业自动重试
- ✅ Webhook 通知
- ✅ Prometheus 指标
- ✅ 分布式追踪
- ✅ 50+ 内置操作符

---

## 推荐实施顺序

1. ✅ **Schemas** - 数据结构（已完成）
2. ✅ **Operators** - 操作符接口（已完成）
3. ✅ **Planner** - DAG 规划器（已完成）
4. ✅ **Executor** - FFmpeg 执行器（已完成）
5. 🔄 **Media Prober** - 元数据探测（当前）
6. 📋 **Store** - 状态存储
7. 📋 **API Server** - REST 接口
8. 📋 **Error Handling** - 错误处理增强
9. 📋 **Configuration** - 配置管理
10. 📋 **Deployment** - Docker 打包

**详细的 MVP 任务分解**: 见 [MVP_ROADMAP.md](MVP_ROADMAP.md)

---

## 文档索引

- [MVP 上线路线图](MVP_ROADMAP.md) - MVP 实施计划
- [变更日志](CHANGELOG.md) - 版本历史
- [架构设计](docs/plans/2025-12-14-media-pipeline-architecture-design.md) - 系统架构
- [Schemas 设计](docs/plans/schemas-detailed-design.md) - 数据结构
- [Planner 设计](docs/plans/planner-detailed-design.md) - 规划器
- [Operator 设计](docs/plans/operator-interface-design.md) - 操作符接口
- [API 设计](docs/plans/api-interface-design.md) - REST API
- [状态管理设计](docs/plans/distributed-state-management-design.md) - 分布式状态
- [错误处理设计](docs/plans/error-handling-design.md) - 错误处理

---

**下一步**: 开始实施 Media Prober - 见 [MVP_ROADMAP.md](MVP_ROADMAP.md#phase-2-media-prober-第一优先级)
