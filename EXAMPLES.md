# Media Pipeline 使用示例

实用的 API 使用示例和常见场景。

## 目录

- [基础示例](#基础示例)
- [API 认证](#api-认证)
- [云存储 S3](#云存储-s3)
- [视频处理](#视频处理)
- [音频处理](#音频处理)
- [批量处理](#批量处理)
- [高级用法](#高级用法)
- [错误处理](#错误处理)

## 基础示例

### 启动服务

```bash
# 使用 Docker Compose 启动
docker-compose up -d

# 检查服务状态
curl http://localhost:8081/health
```

### 创建简单任务

```bash
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {"id": "input1", "source": "test.mp4"}
      ],
      "operations": [
        {
          "op": "trim",
          "input": "input1",
          "output": "trimmed",
          "params": {
            "start": "00:00:10",
            "duration": "00:00:30"
          }
        }
      ],
      "outputs": [
        {"id": "trimmed", "destination": "output.mp4"}
      ]
    }
  }'
```

### 查询任务状态

```bash
# 获取任务详情
JOB_ID="job_1234567890"
curl http://localhost:8081/api/v1/jobs/$JOB_ID

# 响应示例
# {
#   "job_id": "job_1234567890",
#   "status": "processing",
#   "progress": {
#     "overall_percent": 45.5,
#     "current_step": "processing"
#   },
#   "created_at": "2024-12-22T10:30:00Z",
#   "updated_at": "2024-12-22T10:30:15Z"
# }
```

### 列出所有任务

```bash
# 列出所有任务
curl http://localhost:8081/api/v1/jobs

# 按状态筛选
curl "http://localhost:8081/api/v1/jobs?status=completed"

# 分页
curl "http://localhost:8081/api/v1/jobs?limit=10&offset=0"
```

### 取消任务

```bash
# 取消正在处理的任务
curl -X DELETE http://localhost:8081/api/v1/jobs/$JOB_ID
```

## API 认证

Media Pipeline 支持两种认证方式：**JWT Token** 和 **API Key**。

### 认证方式对比

| 特性 | JWT Token | API Key |
|------|-----------|---------|
| 使用场景 | 用户会话、临时访问 | 服务间调用、长期访问 |
| 有效期 | 可配置过期时间 | 可设置过期时间或永久有效 |
| 携带方式 | `Authorization: Bearer <token>` | `X-API-Key: <key>` |
| 包含信息 | UserID、Email、Role | UserID |
| 可撤销性 | 过期前无法撤销 | 可随时撤销 |

### 1. JWT Token 认证

#### 生成 JWT Token（Go 代码示例）

```go
package main

import (
    "fmt"
    "time"
    "your-project/pkg/auth"
)

func main() {
    // 创建 JWT 管理器
    jwtManager := auth.NewJWTManager("your-secret-key", 24*time.Hour)

    // 生成 token
    token, err := jwtManager.Generate("user123", "user@example.com", "admin")
    if err != nil {
        panic(err)
    }

    fmt.Printf("JWT Token: %s\n", token)
    // 输出: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
}
```

#### 使用 JWT Token 访问 API（curl）

```bash
# 设置 token 变量
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 创建任务（需要认证）
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [{"id": "video", "source": "input.mp4"}],
      "operations": [
        {
          "op": "trim",
          "input": "video",
          "output": "trimmed",
          "params": {"start": "00:00:10", "duration": "00:00:30"}
        }
      ],
      "outputs": [{"id": "trimmed", "destination": "output.mp4"}]
    }
  }'
```

#### 验证和刷新 Token（Go 代码示例）

```go
// 验证 token
claims, err := jwtManager.Verify(token)
if err != nil {
    fmt.Printf("Token 无效: %v\n", err)
    return
}
fmt.Printf("User ID: %s, Email: %s, Role: %s\n",
    claims.UserID, claims.Email, claims.Role)

// 刷新 token（延长有效期）
newToken, err := jwtManager.Refresh(token)
if err != nil {
    fmt.Printf("刷新失败: %v\n", err)
    return
}
fmt.Printf("新 Token: %s\n", newToken)
```

### 2. API Key 认证

#### 生成 API Key（Go 代码示例）

```go
package main

import (
    "fmt"
    "time"
    "your-project/pkg/auth"
)

func main() {
    // 创建 API Key 管理器
    apiKeyManager := auth.NewAPIKeyManager()

    // 生成永久有效的 API Key
    apiKey, err := apiKeyManager.Generate("user123", "Production Key", nil)
    if err != nil {
        panic(err)
    }

    fmt.Printf("API Key: %s\n", apiKey.Key)
    fmt.Printf("Created: %s\n", apiKey.CreatedAt)
    // 输出: sk_1a2b3c4d5e6f7g8h9i0j...

    // 生成带过期时间的 API Key
    expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 天后过期
    tempKey, err := apiKeyManager.Generate("user456", "Temp Key", &expiresAt)
    if err != nil {
        panic(err)
    }
    fmt.Printf("临时 API Key: %s (过期时间: %s)\n",
        tempKey.Key, tempKey.ExpiresAt)
}
```

#### 使用 API Key 访问 API（curl）

```bash
# 设置 API Key 变量
API_KEY="sk_1a2b3c4d5e6f7g8h9i0j..."

# 创建任务（使用 API Key 认证）
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [{"id": "video", "source": "input.mp4"}],
      "operations": [
        {
          "op": "scale",
          "input": "video",
          "output": "scaled",
          "params": {"width": 1280, "height": 720}
        }
      ],
      "outputs": [{"id": "scaled", "destination": "output.mp4"}]
    }
  }'

# 查询任务状态
curl http://localhost:8081/api/v1/jobs/$JOB_ID \
  -H "X-API-Key: $API_KEY"
```

#### 管理 API Key（Go 代码示例）

```go
// 列出用户的所有 API Key
keys := apiKeyManager.List("user123")
fmt.Printf("用户有 %d 个 API Key:\n", len(keys))
for _, key := range keys {
    fmt.Printf("- %s (%s) - 已撤销: %v\n",
        key.Name, key.Key[:15]+"...", key.Revoked)
}

// 撤销 API Key
err := apiKeyManager.Revoke("sk_1a2b3c4d5e6f7g8h9i0j...")
if err != nil {
    fmt.Printf("撤销失败: %v\n", err)
}

// 删除 API Key
err = apiKeyManager.Delete("sk_1a2b3c4d5e6f7g8h9i0j...")
if err != nil {
    fmt.Printf("删除失败: %v\n", err)
}

// 获取 API Key 总数
count := apiKeyManager.Count()
fmt.Printf("系统中共有 %d 个有效的 API Key\n", count)
```

### 3. 角色权限控制（RBAC）

#### 配置中间件和角色权限（Go 代码示例）

```go
package main

import (
    "net/http"
    "time"
    "your-project/pkg/auth"
)

func main() {
    // 创建认证管理器
    jwtManager := auth.NewJWTManager("secret-key", 24*time.Hour)
    apiKeyManager := auth.NewAPIKeyManager()

    // 创建认证中间件（必须认证）
    authMiddleware := auth.NewAuthMiddleware(jwtManager, apiKeyManager, false)

    // 创建可选认证中间件（允许匿名访问）
    optionalAuthMiddleware := auth.NewAuthMiddleware(jwtManager, apiKeyManager, true)

    // 需要认证的端点
    http.Handle("/api/v1/jobs",
        authMiddleware.Handler(http.HandlerFunc(createJobHandler)))

    // 只允许 admin 角色访问
    http.Handle("/api/v1/admin/stats",
        authMiddleware.Handler(
            auth.RequireRole("admin")(
                http.HandlerFunc(adminStatsHandler))))

    // 公开端点（可选认证）
    http.Handle("/api/v1/health",
        optionalAuthMiddleware.Handler(http.HandlerFunc(healthHandler)))

    http.ListenAndServe(":8081", nil)
}

func createJobHandler(w http.ResponseWriter, r *http.Request) {
    // 从请求上下文获取用户信息
    userID, ok := auth.GetUserID(r)
    if !ok {
        http.Error(w, "未认证", http.StatusUnauthorized)
        return
    }

    email, _ := auth.GetUserEmail(r)
    role, _ := auth.GetUserRole(r)
    authMethod, _ := auth.GetAuthMethod(r)

    // 处理任务创建...
    w.Write([]byte(fmt.Sprintf(
        "任务已创建 - User: %s, Email: %s, Role: %s, Method: %s",
        userID, email, role, authMethod)))
}

func adminStatsHandler(w http.ResponseWriter, r *http.Request) {
    // 这个 handler 只有 admin 角色能访问
    w.Write([]byte("系统统计信息..."))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    // 可选认证：可以检查是否有用户信息
    if userID, ok := auth.GetUserID(r); ok {
        w.Write([]byte(fmt.Sprintf("健康检查 - 已认证用户: %s", userID)))
    } else {
        w.Write([]byte("健康检查 - 匿名访问"))
    }
}
```

#### 不同角色的访问示例

```bash
# 生成 admin 用户的 token
# (假设已通过代码生成)
ADMIN_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 生成普通用户的 token
USER_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Admin 用户访问管理端点 - 成功
curl http://localhost:8081/api/v1/admin/stats \
  -H "Authorization: Bearer $ADMIN_TOKEN"
# 响应: 200 OK

# 普通用户访问管理端点 - 失败
curl http://localhost:8081/api/v1/admin/stats \
  -H "Authorization: Bearer $USER_TOKEN"
# 响应: 403 Forbidden
# {
#   "error": "Forbidden: Insufficient permissions"
# }
```

### 4. 在客户端 SDK 中使用认证

#### Python SDK（带认证）

```python
import requests
import time

class MediaPipeline:
    def __init__(self, base_url="http://localhost:8081", auth_token=None, api_key=None):
        self.base_url = base_url
        self.auth_token = auth_token
        self.api_key = api_key

    def _get_headers(self):
        """获取认证 headers"""
        headers = {"Content-Type": "application/json"}

        if self.auth_token:
            headers["Authorization"] = f"Bearer {self.auth_token}"
        elif self.api_key:
            headers["X-API-Key"] = self.api_key

        return headers

    def create_job(self, spec):
        """创建处理任务（需要认证）"""
        response = requests.post(
            f"{self.base_url}/api/v1/jobs",
            json={"spec": spec},
            headers=self._get_headers()
        )
        response.raise_for_status()
        return response.json()

    def get_job(self, job_id):
        """获取任务状态（需要认证）"""
        response = requests.get(
            f"{self.base_url}/api/v1/jobs/{job_id}",
            headers=self._get_headers()
        )
        response.raise_for_status()
        return response.json()

# 使用 JWT Token
pipeline_jwt = MediaPipeline(
    auth_token="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
)

# 使用 API Key
pipeline_apikey = MediaPipeline(
    api_key="sk_1a2b3c4d5e6f7g8h9i0j..."
)

# 创建任务
job_spec = {
    "inputs": [{"id": "video", "source": "input.mp4"}],
    "operations": [
        {
            "op": "trim",
            "input": "video",
            "output": "trimmed",
            "params": {"start": "00:00:10", "duration": "00:01:00"}
        }
    ],
    "outputs": [{"id": "trimmed", "destination": "output.mp4"}]
}

result = pipeline_jwt.create_job(job_spec)
print(f"任务已创建: {result['job_id']}")
```

#### Node.js SDK（带认证）

```javascript
const axios = require('axios');

class MediaPipeline {
  constructor(baseUrl = 'http://localhost:8081', options = {}) {
    this.baseUrl = baseUrl;
    this.authToken = options.authToken;
    this.apiKey = options.apiKey;

    this.client = axios.create({
      baseURL: baseUrl,
      headers: this._getHeaders()
    });
  }

  _getHeaders() {
    const headers = { 'Content-Type': 'application/json' };

    if (this.authToken) {
      headers['Authorization'] = `Bearer ${this.authToken}`;
    } else if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }

    return headers;
  }

  async createJob(spec) {
    const { data } = await this.client.post('/api/v1/jobs', { spec });
    return data;
  }

  async getJob(jobId) {
    const { data } = await this.client.get(`/api/v1/jobs/${jobId}`);
    return data;
  }
}

// 使用 JWT Token
const pipelineJWT = new MediaPipeline('http://localhost:8081', {
  authToken: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'
});

// 使用 API Key
const pipelineAPIKey = new MediaPipeline('http://localhost:8081', {
  apiKey: 'sk_1a2b3c4d5e6f7g8h9i0j...'
});

// 创建任务
(async () => {
  const jobSpec = {
    inputs: [{ id: 'video', source: 'input.mp4' }],
    operations: [
      {
        op: 'scale',
        input: 'video',
        output: 'scaled',
        params: { width: 1280, height: 720 }
      }
    ],
    outputs: [{ id: 'scaled', destination: 'output.mp4' }]
  };

  try {
    const result = await pipelineJWT.createJob(jobSpec);
    console.log(`任务已创建: ${result.job_id}`);
  } catch (error) {
    if (error.response?.status === 401) {
      console.error('认证失败：token 无效或已过期');
    } else if (error.response?.status === 403) {
      console.error('权限不足：无法执行此操作');
    } else {
      console.error('请求失败:', error.message);
    }
  }
})();
```

#### Go Client（带认证）

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type Client struct {
    BaseURL   string
    HTTP      *http.Client
    AuthToken string
    APIKey    string
}

func NewClient(baseURL string, authToken string, apiKey string) *Client {
    return &Client{
        BaseURL:   baseURL,
        HTTP:      &http.Client{Timeout: 30 * time.Second},
        AuthToken: authToken,
        APIKey:    apiKey,
    }
}

func (c *Client) addAuth(req *http.Request) {
    if c.AuthToken != "" {
        req.Header.Set("Authorization", "Bearer "+c.AuthToken)
    } else if c.APIKey != "" {
        req.Header.Set("X-API-Key", c.APIKey)
    }
}

func (c *Client) CreateJob(spec map[string]interface{}) (map[string]interface{}, error) {
    body, _ := json.Marshal(map[string]interface{}{"spec": spec})

    req, err := http.NewRequest(
        "POST",
        c.BaseURL+"/api/v1/jobs",
        bytes.NewReader(body),
    )
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")
    c.addAuth(req)

    resp, err := c.HTTP.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusUnauthorized {
        return nil, fmt.Errorf("认证失败: token 无效或已过期")
    } else if resp.StatusCode == http.StatusForbidden {
        return nil, fmt.Errorf("权限不足: 无法执行此操作")
    }

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result, nil
}

func main() {
    // 使用 JWT Token
    clientJWT := NewClient(
        "http://localhost:8081",
        "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "",
    )

    // 使用 API Key
    clientAPIKey := NewClient(
        "http://localhost:8081",
        "",
        "sk_1a2b3c4d5e6f7g8h9i0j...",
    )

    spec := map[string]interface{}{
        "inputs": []map[string]string{
            {"id": "video", "source": "input.mp4"},
        },
        "operations": []map[string]interface{}{
            {
                "op":     "trim",
                "input":  "video",
                "output": "trimmed",
                "params": map[string]string{
                    "start":    "00:00:10",
                    "duration": "00:01:00",
                },
            },
        },
        "outputs": []map[string]string{
            {"id": "trimmed", "destination": "output.mp4"},
        },
    }

    result, err := clientJWT.CreateJob(spec)
    if err != nil {
        fmt.Printf("创建任务失败: %v\n", err)
        return
    }

    fmt.Printf("任务已创建: %s\n", result["job_id"])
}
```

### 5. 常见认证错误处理

```bash
# 1. 缺少认证信息
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{"spec": {...}}'

# 响应: 401 Unauthorized
# {
#   "error": "Unauthorized: No valid authentication provided"
# }

# 2. Token 无效或已过期
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "Authorization: Bearer invalid-token" \
  -H "Content-Type: application/json" \
  -d '{"spec": {...}}'

# 响应: 401 Unauthorized
# {
#   "error": "Unauthorized: Invalid or expired token"
# }

# 3. API Key 已被撤销
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "X-API-Key: sk_revoked_key" \
  -H "Content-Type: application/json" \
  -d '{"spec": {...}}'

# 响应: 401 Unauthorized
# {
#   "error": "Unauthorized: Invalid or revoked API key"
# }

# 4. 权限不足
curl http://localhost:8081/api/v1/admin/stats \
  -H "Authorization: Bearer user-token"

# 响应: 403 Forbidden
# {
#   "error": "Forbidden: Insufficient permissions"
# }
```

### 6. 最佳实践

#### 安全建议

1. **JWT Token**:
   - 使用足够长的密钥（至少 32 字节）
   - 设置合理的过期时间（如 1-24 小时）
   - 不要在 URL 中传递 token
   - 使用 HTTPS 传输

2. **API Key**:
   - 妥善保管 API Key，不要提交到代码仓库
   - 为不同环境使用不同的 API Key
   - 定期轮换 API Key
   - 及时撤销不再使用的 Key

3. **角色权限**:
   - 遵循最小权限原则
   - 定期审查用户权限
   - 为敏感操作添加额外验证

#### 环境变量管理

```bash
# .env 文件（不要提交到 git）
JWT_SECRET=your-very-long-secret-key-here
API_KEY=sk_1a2b3c4d5e6f7g8h9i0j...

# 使用环境变量
export JWT_SECRET=$(cat .env | grep JWT_SECRET | cut -d '=' -f2)
export API_KEY=$(cat .env | grep API_KEY | cut -d '=' -f2)

# 在脚本中使用
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"spec": {...}}'
```

## 云存储 S3

Media Pipeline 支持 Amazon S3 作为输入和输出存储。使用 `s3://` URI 格式访问 S3 对象。

### 配置 AWS 凭证

S3 存储使用 AWS SDK 的默认凭证链，支持以下方式：

#### 1. 环境变量

```bash
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_REGION=us-east-1
```

#### 2. AWS 配置文件

```bash
# ~/.aws/credentials
[default]
aws_access_key_id = your-access-key
aws_secret_access_key = your-secret-key

# ~/.aws/config
[default]
region = us-east-1
```

#### 3. IAM 角色（推荐用于 EC2/ECS）

在 AWS EC2 或 ECS 上运行时，可以使用 IAM 角色自动获取凭证，无需手动配置。

### S3 URI 格式

```
s3://bucket-name/path/to/file.mp4
```

- `bucket-name`: S3 存储桶名称
- `path/to/file.mp4`: 对象键（Key）

### 示例 1：从 S3 读取，输出到本地

```bash
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {
          "id": "video",
          "source": "s3://my-bucket/videos/input.mp4"
        }
      ],
      "operations": [
        {
          "op": "trim",
          "input": "video",
          "output": "trimmed",
          "params": {
            "start": "00:00:10",
            "duration": "00:01:00"
          }
        }
      ],
      "outputs": [
        {
          "id": "trimmed",
          "destination": "file:///output/trimmed.mp4"
        }
      ]
    }
  }'
```

### 示例 2：从本地读取，输出到 S3

```bash
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {
          "id": "video",
          "source": "file:///uploads/source.mp4"
        }
      ],
      "operations": [
        {
          "op": "scale",
          "input": "video",
          "output": "scaled",
          "params": {
            "width": 1920,
            "height": 1080
          }
        }
      ],
      "outputs": [
        {
          "id": "scaled",
          "destination": "s3://my-bucket/processed/output-1080p.mp4",
          "codec": {
            "video": {
              "codec": "libx264",
              "preset": "medium",
              "crf": 23
            }
          }
        }
      ]
    }
  }'
```

### 示例 3：S3 到 S3（完全云端处理）

```bash
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {
          "id": "raw_video",
          "source": "s3://source-bucket/raw/video.mp4"
        }
      ],
      "operations": [
        {
          "op": "trim",
          "input": "raw_video",
          "output": "trimmed",
          "params": {
            "start": "00:05:00",
            "duration": "00:10:00"
          }
        },
        {
          "op": "scale",
          "input": "trimmed",
          "output": "scaled",
          "params": {
            "width": 1280,
            "height": 720
          }
        }
      ],
      "outputs": [
        {
          "id": "scaled",
          "destination": "s3://processed-bucket/videos/final.mp4",
          "codec": {
            "video": {
              "codec": "libx264",
              "preset": "fast",
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
  }'
```

### 示例 4：生成多个分辨率输出到 S3

```bash
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {
          "id": "source",
          "source": "s3://videos/source/original-4k.mp4"
        }
      ],
      "operations": [
        {
          "op": "scale",
          "input": "source",
          "output": "hd",
          "params": {"width": 1920, "height": 1080}
        },
        {
          "op": "scale",
          "input": "source",
          "output": "sd",
          "params": {"width": 1280, "height": 720}
        },
        {
          "op": "scale",
          "input": "source",
          "output": "mobile",
          "params": {"width": 640, "height": 360}
        }
      ],
      "outputs": [
        {
          "id": "hd",
          "destination": "s3://videos/transcoded/video-1080p.mp4"
        },
        {
          "id": "sd",
          "destination": "s3://videos/transcoded/video-720p.mp4"
        },
        {
          "id": "mobile",
          "destination": "s3://videos/transcoded/video-360p.mp4"
        }
      ]
    }
  }'
```

### Go 代码示例：使用 S3 存储

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "your-project/pkg/storage"
)

func main() {
    ctx := context.Background()

    // 创建 S3 存储客户端
    s3Storage, err := storage.NewS3Storage(ctx)
    if err != nil {
        panic(fmt.Sprintf("Failed to create S3 storage: %v", err))
    }

    // 检查文件是否存在
    exists, err := s3Storage.Exists(ctx, "s3://my-bucket/videos/input.mp4")
    if err != nil {
        panic(err)
    }
    fmt.Printf("File exists: %v\n", exists)

    // 下载文件
    reader, err := s3Storage.Get(ctx, "s3://my-bucket/videos/input.mp4")
    if err != nil {
        panic(err)
    }
    defer reader.Close()

    // 上传文件
    data := strings.NewReader("Hello S3!")
    err = s3Storage.Put(ctx, "s3://my-bucket/uploads/test.txt", data)
    if err != nil {
        panic(err)
    }
    fmt.Println("File uploaded successfully")

    // 删除文件
    err = s3Storage.Delete(ctx, "s3://my-bucket/uploads/test.txt")
    if err != nil {
        panic(err)
    }
    fmt.Println("File deleted successfully")
}
```

### S3 权限要求

确保你的 AWS 凭证具有以下权限：

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::your-bucket-name",
        "arn:aws:s3:::your-bucket-name/*"
      ]
    }
  ]
}
```

### 最佳实践

1. **使用 IAM 角色**: 在 AWS 环境中运行时，优先使用 IAM 角色而不是硬编码凭证
2. **区域优化**: 将 Media Pipeline 部署在与 S3 存储桶相同的区域以降低延迟和成本
3. **存储桶策略**: 配置适当的存储桶策略和 CORS 规则
4. **版本控制**: 为重要的输出文件启用 S3 版本控制
5. **生命周期策略**: 配置自动归档或删除旧文件以节省成本
6. **传输加速**: 对于跨区域传输，考虑启用 S3 Transfer Acceleration

### 故障排查

```bash
# 验证 AWS 凭证
aws sts get-caller-identity

# 列出存储桶内容
aws s3 ls s3://your-bucket/

# 测试上传
echo "test" | aws s3 cp - s3://your-bucket/test.txt

# 测试下载
aws s3 cp s3://your-bucket/test.txt -

# 检查存储桶权限
aws s3api get-bucket-policy --bucket your-bucket
```

## 视频处理

### 1. 裁剪视频片段

```bash
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {"id": "video", "source": "input.mp4"}
      ],
      "operations": [
        {
          "op": "trim",
          "input": "video",
          "output": "segment",
          "params": {
            "start": "00:05:30",
            "duration": "00:02:00"
          }
        }
      ],
      "outputs": [
        {"id": "segment", "destination": "segment.mp4"}
      ]
    }
  }'
```

### 2. 调整视频分辨率

```bash
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {"id": "video", "source": "4k-video.mp4"}
      ],
      "operations": [
        {
          "op": "scale",
          "input": "video",
          "output": "hd",
          "params": {
            "width": 1920,
            "height": 1080,
            "algorithm": "lanczos"
          }
        }
      ],
      "outputs": [
        {
          "id": "hd",
          "destination": "hd-video.mp4",
          "codec": {
            "video": {
              "codec": "libx264",
              "preset": "medium",
              "crf": 23
            }
          }
        }
      ]
    }
  }'
```

### 3. 裁剪 + 缩放组合

```bash
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {"id": "video", "source": "long-video.mp4"}
      ],
      "operations": [
        {
          "op": "trim",
          "input": "video",
          "output": "trimmed",
          "params": {
            "start": "00:10:00",
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
          "destination": "highlight.mp4",
          "codec": {
            "video": {
              "codec": "libx264",
              "preset": "fast",
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
  }'
```

### 4. 生成多种分辨率

```bash
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {"id": "source", "source": "original.mp4"}
      ],
      "operations": [
        {
          "op": "scale",
          "input": "source",
          "output": "hd",
          "params": {"width": 1920, "height": 1080}
        },
        {
          "op": "scale",
          "input": "source",
          "output": "sd",
          "params": {"width": 1280, "height": 720}
        },
        {
          "op": "scale",
          "input": "source",
          "output": "mobile",
          "params": {"width": 640, "height": 360}
        }
      ],
      "outputs": [
        {"id": "hd", "destination": "output-1080p.mp4"},
        {"id": "sd", "destination": "output-720p.mp4"},
        {"id": "mobile", "destination": "output-360p.mp4"}
      ]
    }
  }'
```

## 音频处理

### 1. 提取音频

```bash
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {"id": "video", "source": "video.mp4"}
      ],
      "operations": [],
      "outputs": [
        {
          "id": "video",
          "destination": "audio.mp3",
          "codec": {
            "audio": {
              "codec": "libmp3lame",
              "bitrate": "192k"
            }
          }
        }
      ]
    }
  }'
```

### 2. 裁剪音频片段

```bash
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {"id": "audio", "source": "podcast.mp3"}
      ],
      "operations": [
        {
          "op": "trim",
          "input": "audio",
          "output": "clip",
          "params": {
            "start": "00:15:30",
            "duration": "00:03:00"
          }
        }
      ],
      "outputs": [
        {"id": "clip", "destination": "highlight.mp3"}
      ]
    }
  }'
```

## 批量处理

### 使用脚本批量创建任务

```bash
#!/bin/bash
# batch-process.sh

FILES=(
  "video1.mp4"
  "video2.mp4"
  "video3.mp4"
)

for file in "${FILES[@]}"; do
  echo "Processing: $file"

  curl -X POST http://localhost:8081/api/v1/jobs \
    -H "Content-Type: application/json" \
    -d "{
      \"spec\": {
        \"inputs\": [{\"id\": \"input\", \"source\": \"$file\"}],
        \"operations\": [
          {
            \"op\": \"scale\",
            \"input\": \"input\",
            \"output\": \"scaled\",
            \"params\": {\"width\": 1280, \"height\": 720}
          }
        ],
        \"outputs\": [{\"id\": \"scaled\", \"destination\": \"processed-$file\"}]
      }
    }"

  echo ""
  sleep 1
done
```

### 监控批量任务

```bash
#!/bin/bash
# monitor-jobs.sh

# 获取所有处理中的任务
JOBS=$(curl -s "http://localhost:8081/api/v1/jobs?status=processing" | jq -r '.[].job_id')

for job_id in $JOBS; do
  STATUS=$(curl -s "http://localhost:8081/api/v1/jobs/$job_id" | jq -r '.status, .progress.overall_percent')
  echo "Job $job_id: $STATUS"
done
```

## 高级用法

### 1. 使用 Python SDK（示例）

```python
import requests
import time

class MediaPipeline:
    def __init__(self, base_url="http://localhost:8081", auth_token=None, api_key=None):
        self.base_url = base_url
        self.auth_token = auth_token
        self.api_key = api_key

    def _get_headers(self):
        """获取认证 headers"""
        headers = {"Content-Type": "application/json"}
        if self.auth_token:
            headers["Authorization"] = f"Bearer {self.auth_token}"
        elif self.api_key:
            headers["X-API-Key"] = self.api_key
        return headers

    def create_job(self, spec):
        """创建处理任务"""
        response = requests.post(
            f"{self.base_url}/api/v1/jobs",
            json={"spec": spec},
            headers=self._get_headers()
        )
        response.raise_for_status()
        return response.json()

    def get_job(self, job_id):
        """获取任务状态"""
        response = requests.get(
            f"{self.base_url}/api/v1/jobs/{job_id}",
            headers=self._get_headers()
        )
        response.raise_for_status()
        return response.json()

    def wait_for_completion(self, job_id, timeout=300):
        """等待任务完成"""
        start_time = time.time()

        while time.time() - start_time < timeout:
            job = self.get_job(job_id)
            status = job['status']

            if status == 'completed':
                return job
            elif status in ['failed', 'cancelled']:
                raise Exception(f"Job {status}: {job.get('error', {}).get('message')}")

            # 显示进度
            if 'progress' in job:
                percent = job['progress'].get('overall_percent', 0)
                print(f"Progress: {percent:.1f}%")

            time.sleep(2)

        raise TimeoutError(f"Job {job_id} did not complete within {timeout}s")

# 使用示例（带认证）
pipeline = MediaPipeline(api_key="sk_your_api_key_here")

# 创建任务
job_spec = {
    "inputs": [{"id": "video", "source": "input.mp4"}],
    "operations": [
        {
            "op": "trim",
            "input": "video",
            "output": "trimmed",
            "params": {"start": "00:00:10", "duration": "00:01:00"}
        }
    ],
    "outputs": [{"id": "trimmed", "destination": "output.mp4"}]
}

result = pipeline.create_job(job_spec)
job_id = result['job_id']
print(f"Created job: {job_id}")

# 等待完成
try:
    final_job = pipeline.wait_for_completion(job_id)
    print(f"Job completed: {final_job}")
except Exception as e:
    print(f"Job failed: {e}")
```

### 2. 使用 Node.js SDK（示例）

```javascript
const axios = require('axios');

class MediaPipeline {
  constructor(baseUrl = 'http://localhost:8081', options = {}) {
    this.baseUrl = baseUrl;
    this.authToken = options.authToken;
    this.apiKey = options.apiKey;
    this.client = axios.create({
      baseURL: baseUrl,
      headers: this._getHeaders()
    });
  }

  _getHeaders() {
    const headers = { 'Content-Type': 'application/json' };
    if (this.authToken) {
      headers['Authorization'] = `Bearer ${this.authToken}`;
    } else if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }
    return headers;
  }

  async createJob(spec) {
    const { data } = await this.client.post('/api/v1/jobs', { spec });
    return data;
  }

  async getJob(jobId) {
    const { data } = await this.client.get(`/api/v1/jobs/${jobId}`);
    return data;
  }

  async waitForCompletion(jobId, timeout = 300000) {
    const startTime = Date.now();

    while (Date.now() - startTime < timeout) {
      const job = await this.getJob(jobId);

      if (job.status === 'completed') {
        return job;
      } else if (['failed', 'cancelled'].includes(job.status)) {
        throw new Error(`Job ${job.status}: ${job.error?.message}`);
      }

      // 显示进度
      if (job.progress) {
        console.log(`Progress: ${job.progress.overall_percent.toFixed(1)}%`);
      }

      await new Promise(resolve => setTimeout(resolve, 2000));
    }

    throw new Error(`Job ${jobId} did not complete within timeout`);
  }
}

// 使用示例（带认证）
(async () => {
  const pipeline = new MediaPipeline('http://localhost:8081', {
    apiKey: 'sk_your_api_key_here'
  });

  const jobSpec = {
    inputs: [{ id: 'video', source: 'input.mp4' }],
    operations: [
      {
        op: 'scale',
        input: 'video',
        output: 'scaled',
        params: { width: 1280, height: 720 }
      }
    ],
    outputs: [{ id: 'scaled', destination: 'output.mp4' }]
  };

  try {
    const result = await pipeline.createJob(jobSpec);
    console.log(`Created job: ${result.job_id}`);

    const finalJob = await pipeline.waitForCompletion(result.job_id);
    console.log('Job completed:', finalJob);
  } catch (error) {
    console.error('Job failed:', error.message);
  }
})();
```

### 3. 使用 Go Client（示例）

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type Client struct {
    BaseURL   string
    HTTP      *http.Client
    AuthToken string
    APIKey    string
}

func NewClient(baseURL, authToken, apiKey string) *Client {
    return &Client{
        BaseURL:   baseURL,
        HTTP:      &http.Client{Timeout: 30 * time.Second},
        AuthToken: authToken,
        APIKey:    apiKey,
    }
}

func (c *Client) addAuth(req *http.Request) {
    if c.AuthToken != "" {
        req.Header.Set("Authorization", "Bearer "+c.AuthToken)
    } else if c.APIKey != "" {
        req.Header.Set("X-API-Key", c.APIKey)
    }
}

func (c *Client) CreateJob(spec map[string]interface{}) (map[string]interface{}, error) {
    body, _ := json.Marshal(map[string]interface{}{"spec": spec})

    req, err := http.NewRequest(
        "POST",
        c.BaseURL+"/api/v1/jobs",
        bytes.NewReader(body),
    )
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")
    c.addAuth(req)

    resp, err := c.HTTP.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result, nil
}

func (c *Client) GetJob(jobID string) (map[string]interface{}, error) {
    req, err := http.NewRequest("GET", c.BaseURL+"/api/v1/jobs/"+jobID, nil)
    if err != nil {
        return nil, err
    }

    c.addAuth(req)

    resp, err := c.HTTP.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result, nil
}

func main() {
    // 使用 API Key 认证
    client := NewClient("http://localhost:8081", "", "sk_your_api_key_here")

    // 创建任务
    spec := map[string]interface{}{
        "inputs": []map[string]string{
            {"id": "video", "source": "input.mp4"},
        },
        "operations": []map[string]interface{}{
            {
                "op":     "trim",
                "input":  "video",
                "output": "trimmed",
                "params": map[string]string{
                    "start":    "00:00:10",
                    "duration": "00:01:00",
                },
            },
        },
        "outputs": []map[string]string{
            {"id": "trimmed", "destination": "output.mp4"},
        },
    }

    result, err := client.CreateJob(spec)
    if err != nil {
        panic(err)
    }

    jobID := result["job_id"].(string)
    fmt.Printf("Created job: %s\n", jobID)

    // 轮询状态
    for {
        job, _ := client.GetJob(jobID)
        status := job["status"].(string)

        if status == "completed" {
            fmt.Println("Job completed!")
            break
        } else if status == "failed" || status == "cancelled" {
            fmt.Printf("Job %s\n", status)
            break
        }

        if progress, ok := job["progress"].(map[string]interface{}); ok {
            percent := progress["overall_percent"].(float64)
            fmt.Printf("Progress: %.1f%%\n", percent)
        }

        time.Sleep(2 * time.Second)
    }
}
```

## 错误处理

### 处理常见错误

```bash
# 1. 无效的 JobSpec
curl -X POST http://localhost:8081/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{"spec": null}'

# 响应: 400 Bad Request
# {
#   "error": "missing_spec",
#   "message": "Job specification is required",
#   "code": 400
# }

# 2. 任务不存在
curl http://localhost:8081/api/v1/jobs/nonexistent

# 响应: 404 Not Found
# {
#   "error": "job_not_found",
#   "message": "Job nonexistent not found",
#   "code": 404
# }

# 3. 无法取消已完成的任务
curl -X DELETE http://localhost:8081/api/v1/jobs/completed-job-id

# 响应: 400 Bad Request
# {
#   "error": "job_terminal",
#   "message": "Job is already in terminal state",
#   "code": 400
# }
```

### 重试失败的任务

```python
def create_job_with_retry(pipeline, spec, max_retries=3):
    """创建任务并自动重试"""
    for attempt in range(max_retries):
        try:
            result = pipeline.create_job(spec)
            job_id = result['job_id']

            # 等待完成
            final_job = pipeline.wait_for_completion(job_id)
            return final_job

        except Exception as e:
            print(f"Attempt {attempt + 1} failed: {e}")

            if attempt < max_retries - 1:
                time.sleep(5 * (attempt + 1))  # 指数退避
                continue
            else:
                raise

    raise Exception("Max retries exceeded")
```

## 调试技巧

### 查看详细日志

```bash
# 查看 API 日志
docker-compose logs -f api

# 查看最近 100 行
docker-compose logs --tail=100 api

# 保存日志到文件
docker-compose logs api > api.log
```

### 健康检查

```bash
# 检查服务是否正常
curl -v http://localhost:8081/health

# 检查 Docker 容器状态
docker-compose ps

# 检查资源使用
docker stats media-pipeline-api
```

### 性能测试

```bash
# 使用 Apache Bench 进行负载测试
ab -n 100 -c 10 \
  -T 'application/json' \
  -p job-spec.json \
  http://localhost:8081/api/v1/jobs
```

## 更多资源

- [部署指南](DEPLOYMENT.md) - 完整的部署文档
- [API 文档](README.md#api-endpoints) - API 端点详细说明
- [故障排查](DEPLOYMENT.md#troubleshooting) - 常见问题解决

---

**提示**: 所有示例都假设服务运行在 `localhost:8081`。如果使用不同的端口或主机，请相应调整。
