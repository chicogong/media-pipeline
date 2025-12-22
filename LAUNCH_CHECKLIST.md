# MVP 上线检查清单

完成所有检查项后，服务可以上线部署。

**更新日期**: 2025-12-22

---

## 🚀 功能完整性

### 核心功能
- [ ] **提交作业**: POST `/api/v1/jobs` 接受 JobSpec，返回 Job ID
- [ ] **查询状态**: GET `/api/v1/jobs/:id` 返回作业状态和进度
- [ ] **列出作业**: GET `/api/v1/jobs` 返回作业列表
- [ ] **取消作业**: DELETE `/api/v1/jobs/:id` 取消运行中的作业

### 作业执行
- [ ] **元数据探测**: 自动探测输入文件的时长、分辨率、码率
- [ ] **参数验证**: 无效参数返回清晰错误信息
- [ ] **DAG 构建**: 正确构建依赖图，检测循环依赖
- [ ] **FFmpeg 执行**: 成功生成并执行 FFmpeg 命令
- [ ] **进度跟踪**: 实时返回进度百分比（0-100%）
- [ ] **输出文件**: 生成的视频文件格式正确，可播放

### 错误处理
- [ ] **文件不存在**: 返回清晰错误信息
- [ ] **参数错误**: 返回具体的参数验证错误
- [ ] **FFmpeg 失败**: 解析 FFmpeg 错误，返回有用提示
- [ ] **作业取消**: 正确清理资源（临时文件、进程）

---

## ✅ 测试覆盖

### 单元测试
- [x] **Schemas**: 数据结构和验证
- [x] **Operators**: 操作符接口和内置操作符
- [x] **Planner**: DAG 构建、排序、估算（43 tests）
- [x] **Executor**: 命令构建、进度解析（14 tests）
- [ ] **Prober**: ffprobe 包装和解析
- [ ] **Store**: 状态存储 CRUD
- [ ] **API**: HTTP 处理器

### 集成测试
- [ ] **完整流程**: JobSpec → Prober → Planner → Executor → Output
- [ ] **trim 操作**: 裁剪视频到指定时间段
- [ ] **scale 操作**: 调整视频分辨率
- [ ] **组合操作**: trim + scale 流水线
- [ ] **并发作业**: 同时处理 3 个作业

### 端到端测试
- [ ] **真实文件**: 使用真实 MP4 文件测试
- [ ] **进度准确性**: 进度百分比与实际处理进度一致
- [ ] **错误恢复**: 模拟 FFmpeg 失败，验证错误处理

**运行测试**:
```bash
go test ./... -v -cover
```

**目标覆盖率**: > 80%

---

## 📝 代码质量

### 代码规范
- [ ] **命名规范**: 变量、函数、类型命名清晰
- [ ] **文档注释**: 所有公共函数有注释
- [ ] **错误处理**: 所有错误都有适当处理
- [ ] **日志输出**: 关键步骤有日志，日志级别合理

### 安全性
- [ ] **输入验证**: 所有外部输入都经过验证
- [ ] **路径注入**: 防止路径遍历攻击
- [ ] **命令注入**: FFmpeg 参数正确转义
- [ ] **敏感信息**: 日志中不包含密码、密钥

### 性能
- [ ] **无内存泄漏**: 长时间运行无内存增长
- [ ] **资源清理**: Context 取消时正确清理
- [ ] **并发安全**: 共享状态有适当的锁保护

**代码检查**:
```bash
go vet ./...
golint ./...
go test -race ./...
```

---

## 📚 文档完善

### 用户文档
- [x] **README.md**: 项目介绍、快速开始、示例
- [x] **MVP_ROADMAP.md**: MVP 路线图和进度
- [x] **IMPLEMENTATION_GUIDE.md**: 开发指南
- [ ] **API_REFERENCE.md**: API 接口文档
  - 请求/响应格式
  - 错误码说明
  - 示例 curl 命令
- [ ] **DEPLOYMENT.md**: 部署指南
  - 环境要求（Go 1.21+, FFmpeg）
  - 启动命令
  - 配置说明

### 示例文档
- [ ] **examples/**: 示例目录
  - `examples/trim.json` - 裁剪示例
  - `examples/scale.json` - 缩放示例
  - `examples/trim-scale.json` - 组合操作示例
  - `examples/curl-submit.sh` - 提交作业脚本

### 设计文档
- [x] **架构设计**: `docs/plans/2025-12-14-media-pipeline-architecture-design.md`
- [x] **模块设计**: 7 个详细设计文档

---

## 🐳 部署就绪

### Docker 打包
- [ ] **Dockerfile**: 构建 API 服务镜像
  - 多阶段构建（builder + runtime）
  - 包含 FFmpeg
  - 镜像大小 < 200MB
- [ ] **docker-compose.yml**: 单机部署配置
  - API 服务
  - 可选：SQLite 持久化
- [ ] **.dockerignore**: 排除不必要文件

**构建镜像**:
```bash
docker build -t media-pipeline:mvp .
```

**镜像大小**: _____ MB（目标 < 200MB）

### 配置管理
- [ ] **config.yaml**: 默认配置文件
  - Server 配置（host, port）
  - FFmpeg 配置（bin_path, temp_dir）
  - Storage 配置（type, path）
- [ ] **环境变量**: 支持环境变量覆盖
  - `PORT`, `FFMPEG_PATH`, `STORAGE_TYPE`
- [ ] **.env.example**: 环境变量示例文件

### 健康检查
- [ ] **Health Endpoint**: GET `/health`
  - 返回服务状态
  - 检查 FFmpeg 可用性
  - 检查存储可访问
- [ ] **Readiness Endpoint**: GET `/ready`
  - 服务准备就绪
  - 可以处理请求

**健康检查**:
```bash
curl http://localhost:8080/health
# 预期: {"status": "healthy", "ffmpeg": "ok", "storage": "ok"}
```

---

## 🔧 启动验证

### 本地启动
```bash
# 1. 启动服务
go run cmd/api/main.go

# 2. 检查健康
curl http://localhost:8080/health

# 3. 提交测试作业
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d @examples/trim-scale.json

# 4. 查询状态
curl http://localhost:8080/api/v1/jobs/{job_id}
```

### Docker 启动
```bash
# 1. 构建镜像
docker build -t media-pipeline:mvp .

# 2. 启动容器
docker run -p 8080:8080 media-pipeline:mvp

# 3. 验证服务
curl http://localhost:8080/health
```

### 验证清单
- [ ] 服务成功启动，无错误日志
- [ ] 健康检查返回 200 OK
- [ ] 可以提交作业，获得 Job ID
- [ ] 可以查询作业状态
- [ ] FFmpeg 成功执行
- [ ] 输出文件正确生成

---

## 📊 监控和日志

### 日志
- [ ] **结构化日志**: 使用 JSON 格式输出
- [ ] **日志级别**: 支持 DEBUG, INFO, WARN, ERROR
- [ ] **请求日志**: 记录所有 API 请求
- [ ] **错误日志**: 记录所有错误和堆栈

**日志示例**:
```json
{
  "level": "info",
  "time": "2025-12-22T10:00:00Z",
  "msg": "job started",
  "job_id": "123e4567-e89b-12d3-a456-426614174000",
  "operation": "trim"
}
```

### 基础监控（可选）
- [ ] **Metrics Endpoint**: GET `/metrics`
  - 作业总数、成功数、失败数
  - 平均处理时长
  - 当前运行作业数
- [ ] **Prometheus 格式**: 支持 Prometheus 抓取

---

## 🎯 性能基准

### 性能目标
- [ ] **API 响应时间**: < 100ms（不含作业执行）
- [ ] **作业提交**: < 500ms（包含验证和规划）
- [ ] **并发处理**: 可同时处理 5 个作业
- [ ] **内存使用**: < 500MB（空闲状态）

### 压力测试
```bash
# 使用 Apache Bench 测试
ab -n 100 -c 10 http://localhost:8080/api/v1/jobs
```

**结果记录**:
- 请求数: _____
- 并发数: _____
- 平均响应时间: _____ ms
- 成功率: _____% （目标 > 99%）

---

## ✨ 可选增强

这些功能不是 MVP 必需，但可以提升用户体验：

- [ ] **Swagger UI**: 交互式 API 文档
- [ ] **Web Dashboard**: 简单的 Web 界面查看作业
- [ ] **作业历史**: 查询历史作业（最近 100 个）
- [ ] **速率限制**: 防止 API 滥用
- [ ] **认证**: API Key 或 JWT 认证
- [ ] **Webhook**: 作业完成后回调通知

---

## 🚦 上线决策

### 必须完成（阻塞上线）
所有标记为 🔴 的检查项必须完成。

### 建议完成（不阻塞上线）
所有标记为 🟡 的检查项建议完成，但可以后续迭代。

### 可选增强（未来迭代）
所有标记为 🟢 的检查项可以在未来版本中添加。

---

## ✅ 最终确认

上线前，确认以下所有项：

- [ ] 所有核心功能测试通过
- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试通过
- [ ] 文档完整（README, API, Deployment）
- [ ] Docker 镜像构建成功
- [ ] 本地和 Docker 启动验证通过
- [ ] 无已知 P0/P1 Bug
- [ ] 性能满足基准要求

**签署**: ___________（开发者）日期: ___________

---

**上线命令**:
```bash
# 构建镜像
docker build -t media-pipeline:v0.1.0 .

# 推送到仓库（如果需要）
docker tag media-pipeline:v0.1.0 your-registry/media-pipeline:v0.1.0
docker push your-registry/media-pipeline:v0.1.0

# 启动服务
docker run -d -p 8080:8080 --name media-pipeline media-pipeline:v0.1.0

# 验证
curl http://localhost:8080/health
```

---

**下一步**: 开始实施 Media Prober - 见 [MVP_ROADMAP.md](MVP_ROADMAP.md#phase-2-media-prober-第一优先级)
