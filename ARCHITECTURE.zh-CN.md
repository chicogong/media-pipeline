# Media Pipeline 架构文档

完整的架构文档与详细图表。

## 目录

- [系统概览](#系统概览)
- [模块架构](#模块架构)
- [数据流](#数据流)
- [算子系统](#算子系统)
- [规划器架构](#规划器架构)
- [执行器架构](#执行器架构)
- [API 层](#api-层)
- [未来分布式架构](#未来分布式架构)

## 系统概览

### 高层架构

```mermaid
graph TB
    subgraph "客户端层"
        WebApp[Web 应用]
        Mobile[移动应用]
        CLI[CLI 工具]
    end

    subgraph "API 层"
        Gateway[API 网关<br/>REST 端点]
        Auth[认证<br/>未来功能]
        RateLimit[限流<br/>未来功能]
    end

    subgraph "业务逻辑层"
        direction TB
        Validator[验证器<br/>JobSpec 验证]
        Prober[媒体探测器<br/>FFprobe 封装]
        Planner[规划器<br/>DAG 构建]
        Executor[执行器<br/>FFmpeg 运行]
    end

    subgraph "数据层"
        Store[Store 接口]
        MemStore[内存存储<br/>当前]
        PostgresStore[PostgreSQL 存储<br/>未来]
        RedisCache[Redis 缓存<br/>未来]
    end

    subgraph "存储层"
        Local[本地文件系统]
        S3[S3/对象存储<br/>未来]
        NFS[网络存储<br/>未来]
    end

    WebApp --> Gateway
    Mobile --> Gateway
    CLI --> Gateway

    Gateway --> Auth
    Auth --> RateLimit
    RateLimit --> Validator

    Validator --> Prober
    Prober --> Planner
    Planner --> Executor

    Gateway --> Store
    Store --> MemStore
    Store -.->|未来| PostgresStore
    Store -.->|未来| RedisCache

    Executor --> Local
    Executor -.->|未来| S3
    Executor -.->|未来| NFS

    style Gateway fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Validator fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style Prober fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style Planner fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style Executor fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style MemStore fill:#FF9800,stroke:#333,stroke-width:2px,color:#fff
```

## 模块架构

### 核心模块

```mermaid
graph LR
    subgraph "pkg/schemas"
        JobSpec[JobSpec<br/>任务定义]
        ProcessingPlan[ProcessingPlan<br/>执行计划]
        JobStatus[JobStatus<br/>任务状态]
        MediaInfo[MediaInfo<br/>媒体元数据]
    end

    subgraph "pkg/operators"
        Interface[Operator 接口<br/>6 个核心方法]
        Registry[全局注册表<br/>算子查找]
        TypeSystem[类型系统<br/>11 种参数类型]
        Validation[验证框架<br/>声明式规则]

        subgraph "内置算子"
            Trim[Trim<br/>时间范围]
            Scale[Scale<br/>分辨率]
        end
    end

    subgraph "pkg/planner"
        DAGBuilder[DAG 构建器<br/>图构建]
        Topo[拓扑排序<br/>Kahn 算法]
        Estimator[资源估算器<br/>CPU/内存/磁盘]
        Metadata[元数据传播<br/>类型推断]
    end

    subgraph "pkg/executor"
        CommandBuilder[命令构建器<br/>FFmpeg 参数]
        ProcessManager[进程管理器<br/>执行与取消]
        ProgressParser[进度解析器<br/>实时更新]
    end

    JobSpec --> DAGBuilder
    DAGBuilder --> ProcessingPlan
    ProcessingPlan --> CommandBuilder

    Interface --> Trim
    Interface --> Scale
    Registry --> Interface

    DAGBuilder --> Registry
    CommandBuilder --> Registry

    style JobSpec fill:#FFC107,stroke:#333,stroke-width:2px
    style ProcessingPlan fill:#FFC107,stroke:#333,stroke-width:2px
    style Interface fill:#9C27B0,stroke:#333,stroke-width:2px,color:#fff
    style DAGBuilder fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style CommandBuilder fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
```

## 数据流

### 任务处理数据流

```mermaid
flowchart TD
    Start([客户端提交 JobSpec])

    subgraph "验证阶段"
        V1[解析 JSON]
        V2[验证模式]
        V3[检查参数]
        V4[SSRF 防护]
    end

    subgraph "探测阶段"
        P1[提取输入 URL]
        P2[下载样本<br/>未来功能]
        P3[运行 FFprobe]
        P4[解析 MediaInfo]
    end

    subgraph "规划阶段"
        PL1[构建依赖图]
        PL2[检测环]
        PL3[拓扑排序]
        PL4[传播元数据]
        PL5[估算资源]
    end

    subgraph "执行阶段"
        E1[生成 FFmpeg 命令]
        E2[启动进程]
        E3[解析进度]
        E4[更新状态]
    end

    Finish([任务完成])
    Error([任务失败])

    Start --> V1
    V1 --> V2
    V2 --> V3
    V3 --> V4
    V4 -->|有效| P1
    V4 -->|无效| Error

    P1 --> P2
    P2 --> P3
    P3 --> P4
    P4 -->|成功| PL1
    P4 -->|失败| Error

    PL1 --> PL2
    PL2 -->|无环| PL3
    PL2 -->|检测到环| Error
    PL3 --> PL4
    PL4 --> PL5
    PL5 --> E1

    E1 --> E2
    E2 --> E3
    E3 --> E4
    E4 -->|成功| Finish
    E4 -->|错误| Error

    style Start fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Finish fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Error fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
```

### Store 数据模型

```mermaid
erDiagram
    JOB {
        string JobID PK
        timestamp Created
        timestamp Updated
        JobState Status
        JobSpec Spec
        ProcessingPlan Plan
        Progress Progress
        ErrorInfo Error
        timestamp StartedAt
        timestamp CompletedAt
        OutputFile[] OutputFiles
        int RetryCount
        string WorkerID
    }

    JOB_SPEC {
        Input[] Inputs
        Operation[] Operations
        Output[] Outputs
    }

    INPUT {
        string ID
        string Source
    }

    OPERATION {
        string Op
        string Input
        string Output
        map Params
    }

    OUTPUT {
        string ID
        string Destination
        CodecConfig Codec
    }

    PROCESSING_PLAN {
        Node[] Nodes
        Edge[] Edges
        Stage[] Stages
        ResourceEstimates Resources
    }

    JOB ||--|| JOB_SPEC : 包含
    JOB ||--o| PROCESSING_PLAN : 生成
    JOB_SPEC ||--|{ INPUT : 有
    JOB_SPEC ||--|{ OPERATION : 有
    JOB_SPEC ||--|{ OUTPUT : 有
```

## 算子系统

### 算子生命周期

```mermaid
stateDiagram-v2
    [*] --> Registration: 系统启动

    Registration --> Idle: 在 GlobalRegistry 中注册
    Idle --> Validation: 调用 Plan()

    Validation --> MetadataGen: Validate() 成功
    Validation --> Error: Validate() 失败

    MetadataGen --> CommandGen: EstimateOutputMetadata()
    CommandGen --> Idle: BuildCommand()

    Error --> [*]
    Idle --> [*]: 关闭

    note right of Registration
        init() 函数
        注册算子
    end note

    note right of Validation
        类型检查，
        参数验证，
        范围检查
    end note

    note right of MetadataGen
        推断输出格式、
        分辨率、时长
    end note

    note right of CommandGen
        生成 FFmpeg
        滤镜参数
    end note
```

### 类型系统

```mermaid
graph TB
    subgraph "类型层次"
        Type[参数类型]

        String[String<br/>文本值]
        Int[Int<br/>整数]
        Float[Float<br/>小数]
        Bool[Bool<br/>true/false]
        Duration[Duration<br/>时间跨度]

        subgraph "复杂类型"
            Object[Object<br/>嵌套 map]
            Array[Array<br/>列表]
            Enum[Enum<br/>固定选项]
        end

        subgraph "媒体类型"
            Resolution[Resolution<br/>1920x1080]
            Timecode[Timecode<br/>HH:MM:SS.mmm]
            Codec[Codec<br/>视频/音频配置]
        end
    end

    subgraph "验证规则"
        Required[必填]
        Range[范围<br/>最小/最大值]
        Pattern[模式<br/>正则]
        Custom[自定义<br/>验证函数]
    end

    Type --> String
    Type --> Int
    Type --> Float
    Type --> Bool
    Type --> Duration
    Type --> Object
    Type --> Array
    Type --> Enum
    Type --> Resolution
    Type --> Timecode
    Type --> Codec

    String --> Required
    Int --> Range
    String --> Pattern
    Type --> Custom

    style Type fill:#9C27B0,stroke:#333,stroke-width:2px,color:#fff
    style Required fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
    style Range fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
    style Pattern fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
```

### 内置算子

```mermaid
graph LR
    subgraph "输入算子"
        Download[Download<br/>未来]
        Probe[Probe<br/>元数据]
    end

    subgraph "变换算子"
        Trim[Trim<br/>✅ 已实现]
        Scale[Scale<br/>✅ 已实现]
        Loudnorm[Loudnorm<br/>未来]
        Mix[Mix 音频<br/>未来]
        Overlay[Overlay 视频<br/>未来]
        Concat[拼接<br/>未来]
    end

    subgraph "输出算子"
        Encode[编码<br/>编解码器配置]
        Upload[上传<br/>S3/GCS 未来]
    end

    Input[输入媒体] --> Download
    Download --> Probe
    Probe --> Trim
    Trim --> Scale
    Scale --> Loudnorm
    Loudnorm --> Mix
    Mix --> Overlay
    Overlay --> Concat
    Concat --> Encode
    Encode --> Upload
    Upload --> Output[输出媒体]

    style Trim fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Scale fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Loudnorm fill:#9E9E9E,stroke:#333,stroke-width:2px
    style Mix fill:#9E9E9E,stroke:#333,stroke-width:2px
    style Overlay fill:#9E9E9E,stroke:#333,stroke-width:2px
    style Concat fill:#9E9E9E,stroke:#333,stroke-width:2px
```

## 规划器架构

### DAG 构建过程

```mermaid
flowchart TD
    Start([JobSpec 输入])

    subgraph "图构建"
        B1[创建输入节点]
        B2[创建操作节点]
        B3[创建输出节点]
        B4[从依赖构建边]
    end

    subgraph "验证"
        V1{有环？}
        V2{所有输入已解析？}
        V3{有效的算子？}
    end

    subgraph "优化"
        O1[拓扑排序]
        O2[计算阶段]
        O3[并行组]
    end

    subgraph "元数据传播"
        M1[输入元数据]
        M2[算子转换]
        M3[输出元数据]
    end

    subgraph "资源估算"
        R1[估算 CPU 使用]
        R2[估算内存]
        R3[估算磁盘 I/O]
        R4[估算时长]
    end

    Finish([ProcessingPlan 输出])
    Error([规划错误])

    Start --> B1
    B1 --> B2
    B2 --> B3
    B3 --> B4

    B4 --> V1
    V1 -->|否| V2
    V1 -->|是| Error
    V2 -->|是| V3
    V2 -->|否| Error
    V3 -->|是| O1
    V3 -->|否| Error

    O1 --> O2
    O2 --> O3

    O3 --> M1
    M1 --> M2
    M2 --> M3

    M3 --> R1
    R1 --> R2
    R2 --> R3
    R3 --> R4

    R4 --> Finish

    style Start fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Finish fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Error fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
```

### 执行阶段

```mermaid
gantt
    title 并行执行阶段
    dateFormat  X
    axisFormat %s

    section 阶段 0
    输入节点 1    :0, 10
    输入节点 2    :0, 10

    section 阶段 1
    Trim 操作  :10, 30

    section 阶段 2
    Scale 操作 :40, 50

    section 阶段 3
    编码输出 1 :90, 110
    编码输出 2 :90, 110

    section 依赖关系
    阶段 1 等待阶段 0 :crit, 10, 0
    阶段 2 等待阶段 1 :crit, 40, 0
    阶段 3 等待阶段 2 :crit, 90, 0
```

## 执行器架构

### FFmpeg 执行流程

```mermaid
sequenceDiagram
    participant E as 执行器
    participant CB as 命令构建器
    participant PM as 进程管理器
    participant FF as FFmpeg 进程
    participant PP as 进度解析器
    participant C as 回调

    E->>CB: BuildCommand(plan)
    CB->>CB: 生成 FFmpeg 参数
    CB-->>E: []string (命令)

    E->>PM: Start(command)
    PM->>FF: exec.CommandContext()
    FF-->>PM: 进程已启动

    loop 每行 stderr 输出
        FF->>PP: stderr 输出
        PP->>PP: 解析进度
        PP->>C: OnProgress(frame, fps, bitrate)
        C-->>E: 更新任务状态
    end

    FF-->>PM: 进程退出
    PM-->>E: 成功/错误

    alt 成功
        E->>E: 验证输出文件
        E-->>E: 完成
    else 错误
        E->>E: 解析错误消息
        E-->>E: 失败
    end
```

## API 层

### API 请求流程

```mermaid
sequenceDiagram
    participant C as 客户端
    participant M as 中间件链
    participant H as 处理器
    participant S as 存储
    participant BG as 后台 Worker

    C->>M: HTTP 请求

    Note over M: 日志中间件
    M->>M: 记录请求详情

    Note over M: CORS 中间件
    M->>M: 添加 CORS 头

    Note over M: 恢复中间件
    M->>M: 设置 panic 恢复

    M->>H: 转发到处理器
    H->>H: 解析请求体
    H->>H: 验证输入

    H->>S: CreateJob(job)
    S-->>H: job_id

    H->>BG: 启动异步处理
    Note over BG: goroutine

    H-->>M: 响应 (201 Created)
    M->>M: 记录响应
    M-->>C: HTTP 响应

    Note over BG: 后台处理
    BG->>S: UpdateStatus(validating)
    BG->>BG: Probe → Plan → Execute
    BG->>S: UpdateStatus(completed)
```

### 中间件链

```mermaid
graph LR
    Request[HTTP 请求]

    subgraph "中间件链"
        Log[日志<br/>请求/响应]
        CORS[CORS<br/>头部]
        Recovery[Panic 恢复<br/>错误处理]
        Auth[认证<br/>未来]
        RateLimit[限流<br/>未来]
    end

    Handler[路由处理器]
    Response[HTTP 响应]

    Request --> Log
    Log --> CORS
    CORS --> Recovery
    Recovery --> Auth
    Auth --> RateLimit
    RateLimit --> Handler
    Handler --> Response

    style Log fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style CORS fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style Recovery fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style Auth fill:#9E9E9E,stroke:#333,stroke-width:2px
    style RateLimit fill:#9E9E9E,stroke:#333,stroke-width:2px
```

## 未来分布式架构

### Worker 池架构

```mermaid
graph TB
    subgraph "API 层"
        API1[API 服务器 1]
        API2[API 服务器 2]
        API3[API 服务器 3]
    end

    subgraph "消息队列"
        Queue[(Redis 队列<br/>优先级 + FIFO)]
    end

    subgraph "Worker 池"
        W1[Worker 1<br/>执行器]
        W2[Worker 2<br/>执行器]
        W3[Worker 3<br/>执行器]
        W4[Worker 4<br/>执行器]
    end

    subgraph "共享状态"
        DB[(PostgreSQL<br/>任务状态)]
        Cache[(Redis<br/>热数据)]
    end

    subgraph "共享存储"
        NFS[NFS/网络存储<br/>媒体文件]
    end

    Client([客户端]) --> API1
    Client --> API2
    Client --> API3

    API1 --> Queue
    API2 --> Queue
    API3 --> Queue

    Queue --> W1
    Queue --> W2
    Queue --> W3
    Queue --> W4

    W1 --> DB
    W2 --> DB
    W3 --> DB
    W4 --> DB

    W1 --> Cache
    W2 --> Cache
    W3 --> Cache
    W4 --> Cache

    W1 <--> NFS
    W2 <--> NFS
    W3 <--> NFS
    W4 <--> NFS

    style API1 fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style API2 fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style API3 fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style W1 fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style W2 fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style W3 fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style W4 fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
```

### 横向扩展策略

```mermaid
graph TB
    subgraph "流量增长"
        T1[100 请求/秒] --> T2[500 请求/秒] --> T3[1000 请求/秒]
    end

    subgraph "扩展策略"
        S1[1 API + 2 Worker]
        S2[3 API + 5 Worker]
        S3[5 API + 10 Worker]
    end

    subgraph "资源分配"
        R1[轻量: 2 CPU, 4GB RAM]
        R2[中等: 8 CPU, 16GB RAM]
        R3[重度: 16 CPU, 32GB RAM]
    end

    T1 --> S1
    T2 --> S2
    T3 --> S3

    S1 --> R1
    S2 --> R2
    S3 --> R3

    style T1 fill:#FFC107,stroke:#333,stroke-width:2px
    style T2 fill:#FF9800,stroke:#333,stroke-width:2px,color:#fff
    style T3 fill:#FF5722,stroke:#333,stroke-width:2px,color:#fff
```

## 技术栈

```mermaid
mindmap
  root((Media Pipeline))
    后端
      Go 1.21
        net/http
        context
        encoding/json
      FFmpeg 8.0
        libx264
        libx265
        aac
    数据
      PostgreSQL 15
        ACID
        主从复制
      Redis 7
        缓存
        队列
        发布/订阅
      内存存储
        线程安全
        MVP
    基础设施
      Docker
        多阶段构建
        Alpine Linux
      Docker Compose
        服务编排
        健康检查
      Kubernetes
        未来扩展
        HA 部署
    监控
      Prometheus
        指标
        告警
      Grafana
        仪表板
        可视化
      Loki
        日志聚合
        查询
```

---

**文档版本**: 1.0
**最后更新**: 2024-12-22
**状态**: 生产就绪
