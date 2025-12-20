# Media Pipeline（中文）

基于 FFmpeg 的声明式、可扩展媒体处理流水线。

## 概述

Media Pipeline 面向生产环境的视频/音频处理场景，提供声明式 JobSpec（描述“做什么”，而不是“怎么做”），并将其编译为可执行的 FFmpeg 处理计划与命令。

## 主要特性

- **声明式 API**：用高层算子表达剪辑、转码与处理意图
- **算子体系**：内置算子（trim、scale、loudnorm 等）与自定义扩展
- **可分布式扩展**：多 Worker 横向扩容
- **类型安全**：参数校验与类型转换
- **可扩展**：无需修改核心即可注册新算子
- **可观测**：进度、日志、指标/链路跟踪（规划中）
- **可靠性**：错误处理、失败恢复/重试（规划中）

## 架构概览

```
┌─────────────┐
│ REST API    │  JobSpec 提交、状态查询
└─────────────┘
       │
       ▼
┌─────────────┐
│  Validator  │  参数校验、SSRF 防护
└─────────────┘
       │
       ▼
┌─────────────┐
│   Planner   │  DAG 构建、资源估算
└─────────────┘
       │
       ▼
┌─────────────┐
│ Job Queue   │  优先级调度（Redis）
└─────────────┘
       │
       ▼
┌─────────────┐
│   Workers   │  FFmpeg 执行、进度解析
└─────────────┘
       │
       ▼
┌─────────────┐
│   Storage   │  S3/GCS 输出上传
└─────────────┘
```

## 快速开始

### 示例：裁剪并缩放视频

```json
{
  "inputs": [
    {
      "id": "video",
      "source": "s3://bucket/input.mp4"
    }
  ],
  "operations": [
    {
      "op": "trim",
      "input": "video",
      "output": "trimmed",
      "params": {
        "start": "00:00:10",
        "duration": "00:05:00"
      }
    },
    {
      "op": "scale",
      "input": "trimmed",
      "output": "scaled",
      "params": {
        "width": 1280,
        "height": 720,
        "algorithm": "lanczos"
      }
    }
  ],
  "outputs": [
    {
      "id": "scaled",
      "destination": "s3://bucket/output.mp4",
      "codec": {
        "video": {
          "codec": "libx264",
          "preset": "medium",
          "crf": 23
        },
        "audio": {
          "codec": "aac",
          "bitrate": "128k"
        }
      }
    }
  ]
}
```

## 项目结构

```
media-pipeline/
├── cmd/
│   ├── api/          # API 服务（规划）
│   └── worker/       # Worker 进程（规划）
├── pkg/
│   ├── schemas/      # 数据结构（JobSpec、ProcessingPlan 等）
│   ├── operators/    # 算子接口与内置算子
│   ├── planner/      # DAG 构建与资源估算
│   ├── executor/     # FFmpeg 命令构建与执行
│   ├── store/        # 数据库/队列（规划）
│   └── api/          # HTTP handlers（规划）
├── internal/
│   └── config/       # 配置（规划）
└── docs/
    └── plans/        # 设计文档
```

## 实现状态

### ✅ 已完成（60%）

- **Schemas**（`pkg/schemas/`）- 4 文件，约 400 行
  - JobSpec、ProcessingPlan、JobStatus
  - Duration（支持 Go duration / timecode / ISO 8601）
  - MediaInfo（音视频元数据）
  - 资源估算结构（NodeEstimates、ResourceEstimates）

- **Operators**（`pkg/operators/`）- 7 文件，约 800 行
  - Operator 接口（6 个核心方法）
  - 参数类型系统（11 种类型）
  - 声明式校验规则与自动类型转换
  - Registry（全局注册与发现）

- **内置算子**（`pkg/operators/builtin/`）
  - `trim`：按时间范围裁剪
  - `scale`：分辨率缩放（lanczos/bicubic 等）

- **Planner**（`pkg/planner/`）- 13 文件，约 1,400 行，43 tests
  - DAG 构建、环检测
  - 拓扑排序与执行 stage 计算
  - 元数据传播
  - 资源估算
  - 集成 planner + 测试

- **Executor**（`pkg/executor/`）- 7 文件，约 600 行，14 tests
  - 从 ProcessingPlan 构建 FFmpeg 命令
  - 实时进度解析
  - 进程执行与取消
  - 错误处理与测试

**合计**：31 文件，约 3,200 行代码 + 1,900 行测试

### 📋 下一步

- **Media Prober**：ffprobe 封装与并行探测
- **Store**：PostgreSQL/Redis（状态机、队列、锁）
- **Error Handling**：错误码体系、FFmpeg 错误解析、重试策略
- **API Server**：REST API、认证、Webhook
- **Worker 协调**：分布式执行与恢复
- **更多算子**：loudnorm、mix、concat、overlay 等

## 设计文档

设计文档位于 `docs/plans/`（目前以英文为主）：

1. [Architecture Design](docs/plans/2025-12-14-media-pipeline-architecture-design.md)
2. [Schemas Detailed Design](docs/plans/schemas-detailed-design.md)
3. [Planner Module Design](docs/plans/planner-detailed-design.md)
4. [Operator Interface Design](docs/plans/operator-interface-design.md)
5. [API Interface Design](docs/plans/api-interface-design.md)
6. [Distributed State Management](docs/plans/distributed-state-management-design.md)
7. [Error Handling Design](docs/plans/error-handling-design.md)

## 参与贡献

实现路线图请参考 `IMPLEMENTATION_GUIDE.md`，总体进度请参考 `PROGRESS.md`。

## 许可证

MIT License，详见 `LICENSE`。

---

**状态**：Core Engine 完成（60%）。已实现 Schemas / Operators / Planner / Executor，并包含较完整测试；下一阶段优先补齐媒体探测、状态管理与错误处理。
