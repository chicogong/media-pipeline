# MVP Roadmap - 上线路线图

**目标**: 实现可运行的单机版媒体处理服务
**当前进度**: 60% → 目标 100%
**预计工作量**: 剩余 40%

---

## 已完成 ✅ (60%)

| 模块 | 状态 | 代码量 | 测试 |
|------|------|--------|------|
| Schemas | ✅ 完成 | 400 行 | - |
| Operators | ✅ 完成 | 800 行 | - |
| Planner | ✅ 完成 | 1,400 行 | 43 个 |
| Executor | ✅ 完成 | 600 行 | 14 个 |

**说明**: 核心引擎已实现，可以将 JobSpec 转换为 FFmpeg 命令并执行。

---

## MVP 必需功能 🔴 (30%)

### 1. Media Prober (10%)
**优先级**: 🔴 最高
**工作量**: ~300 行代码

**功能需求**:
```go
// 探测输入文件的元数据
prober := prober.New()
info, err := prober.Probe(ctx, "s3://bucket/input.mp4")
// 返回: 时长、分辨率、码率、编码格式等
```

**实现任务**:
- [ ] `pkg/prober/prober.go` - ffprobe 包装器
- [ ] `pkg/prober/parser.go` - 解析 JSON 输出
- [ ] `pkg/prober/prober_test.go` - 单元测试
- [ ] 支持本地文件和 S3/HTTP URL
- [ ] 错误处理（文件不存在、格式不支持等）

**为什么必需**: Planner 需要输入元数据才能计算输出元数据和资源估算。

---

### 2. Store Module - 简化版 (10%)
**优先级**: 🔴 高
**工作量**: ~400 行代码

**功能需求**:
```go
// MVP: 使用内存或 SQLite 存储
store := store.NewMemoryStore()

// 保存作业
store.SaveJob(ctx, job)

// 查询状态
status, err := store.GetJobStatus(ctx, jobID)

// 更新进度
store.UpdateProgress(ctx, jobID, progress)
```

**实现任务**:
- [ ] `pkg/store/store.go` - Store 接口定义
- [ ] `pkg/store/memory.go` - 内存实现（MVP）
- [ ] `pkg/store/models.go` - 数据模型
- [ ] `pkg/store/store_test.go` - 单元测试

**为什么必需**: 需要持久化作业状态，API 才能查询进度。

---

### 3. API Server (10%)
**优先级**: 🔴 高
**工作量**: ~500 行代码

**功能需求**:
```
POST   /api/v1/jobs          - 提交作业
GET    /api/v1/jobs/:id      - 查询状态
GET    /api/v1/jobs          - 列出作业
DELETE /api/v1/jobs/:id      - 取消作业
```

**实现任务**:
- [ ] `cmd/api/main.go` - 服务器入口
- [ ] `pkg/api/handlers.go` - HTTP 处理器
- [ ] `pkg/api/middleware.go` - 日志、CORS、认证
- [ ] `pkg/api/api_test.go` - 集成测试
- [ ] 配置文件支持（端口、日志等）

**为什么必需**: 对外提供服务接口。

---

## 可选增强 🟡 (10%)

### 4. 错误处理增强 (5%)
**当前状态**: Executor 有基础错误捕获
**优先级**: 🟡 中

**增强任务**:
- [ ] FFmpeg 错误解析（从 stderr 提取有用信息）
- [ ] 错误分类（编码错误、IO 错误、参数错误等）
- [ ] 重试策略（临时错误自动重试）
- [ ] `pkg/errors/ffmpeg.go` - FFmpeg 错误类型

---

### 5. 配置管理 (3%)
**优先级**: 🟡 中

**任务**:
- [ ] `internal/config/config.go` - 配置结构
- [ ] 支持环境变量和配置文件
- [ ] FFmpeg 路径、临时目录、API 端口等

---

### 6. 基础监控 (2%)
**优先级**: 🟢 低（MVP 可选）

**任务**:
- [ ] Prometheus metrics 端点
- [ ] 基础指标：作业数量、成功率、处理时长

---

## MVP 架构（简化版）

```
┌──────────────┐
│   REST API   │  提交 JobSpec，查询状态
│ (cmd/api)    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│    Store     │  内存/SQLite 存储作业状态
│ (pkg/store)  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│    Prober    │  探测输入文件元数据
│ (pkg/prober) │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Planner    │  ✅ 已完成
│ (pkg/planner)│
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Executor   │  ✅ 已完成
│ (pkg/executor)│
└──────────────┘
```

**说明**: MVP 是单进程架构，API 直接调用 Prober → Planner → Executor。

---

## 实施步骤

### Phase 2: Media Prober (第一优先级)
1. 实现 ffprobe 包装器
2. 解析 JSON 输出为 MediaInfo
3. 编写单元测试
4. 集成到 Planner

### Phase 3: Store Module (第二优先级)
1. 定义 Store 接口
2. 实现内存版本（MVP）
3. 编写单元测试

### Phase 4: API Server (第三优先级)
1. 实现核心 HTTP 处理器
2. 集成 Prober + Planner + Executor
3. 实现状态查询
4. 添加中间件（日志、错误处理）
5. 编写集成测试

### Phase 5: 测试和优化
1. 端到端测试（完整工作流）
2. 错误处理改进
3. 性能优化
4. 文档完善

---

## 测试计划

### 单元测试覆盖
- [x] Schemas
- [x] Operators
- [x] Planner (43 tests)
- [x] Executor (14 tests)
- [ ] Prober
- [ ] Store
- [ ] API handlers

### 集成测试
- [ ] 完整作业提交流程
- [ ] 错误场景（文件不存在、FFmpeg 失败等）
- [ ] 并发作业处理

### 端到端测试
- [ ] 真实 MP4 文件处理
- [ ] trim + scale 组合操作
- [ ] 进度跟踪准确性

---

## 上线检查清单

### 功能完整性
- [ ] 可以提交作业并获取 Job ID
- [ ] 可以查询作业状态和进度
- [ ] FFmpeg 执行成功，输出文件正确
- [ ] 错误场景有合理提示

### 代码质量
- [ ] 所有新代码有单元测试
- [ ] 无明显性能问题
- [ ] 错误处理完善
- [ ] 日志输出清晰

### 文档
- [ ] API 接口文档（OpenAPI/Swagger）
- [ ] 部署文档（如何启动服务）
- [ ] 配置说明（环境变量）
- [ ] 示例 JobSpec

### 部署
- [ ] Dockerfile
- [ ] docker-compose.yml（包含 SQLite/内存存储）
- [ ] 环境变量配置示例
- [ ] 健康检查端点 `/health`

---

## 时间估算

**注**: 仅供参考，实际根据实施情况调整

| 阶段 | 工作量 | 说明 |
|------|--------|------|
| Media Prober | 10% | ~半天，较简单 |
| Store Module | 10% | ~半天，内存版本简单 |
| API Server | 10% | ~1天，需要处理集成逻辑 |
| 错误处理 | 5% | ~半天 |
| 测试优化 | 5% | ~半天 |
| **总计** | **40%** | **约 2.5-3 天** |

---

## 成功标准

MVP 上线后应满足：

1. ✅ **可用性**: 可以通过 REST API 提交 trim+scale 作业
2. ✅ **可观测**: 可以查询作业状态和进度百分比
3. ✅ **可靠性**: FFmpeg 执行成功，错误有明确提示
4. ✅ **可部署**: 一条 `docker run` 命令启动服务

---

**下一步**: 开始实施 Phase 2 - Media Prober
