# Media Pipeline 使用示例

实用的 API 使用示例和常见场景。

## 目录

- [基础示例](#基础示例)
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
    def __init__(self, base_url="http://localhost:8081"):
        self.base_url = base_url

    def create_job(self, spec):
        """创建处理任务"""
        response = requests.post(
            f"{self.base_url}/api/v1/jobs",
            json={"spec": spec}
        )
        response.raise_for_status()
        return response.json()

    def get_job(self, job_id):
        """获取任务状态"""
        response = requests.get(
            f"{self.base_url}/api/v1/jobs/{job_id}"
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

# 使用示例
pipeline = MediaPipeline()

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
  constructor(baseUrl = 'http://localhost:8081') {
    this.baseUrl = baseUrl;
    this.client = axios.create({ baseURL: baseUrl });
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

// 使用示例
(async () => {
  const pipeline = new MediaPipeline();

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
    BaseURL string
    HTTP    *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        BaseURL: baseURL,
        HTTP:    &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *Client) CreateJob(spec map[string]interface{}) (map[string]interface{}, error) {
    body, _ := json.Marshal(map[string]interface{}{"spec": spec})

    resp, err := c.HTTP.Post(
        c.BaseURL+"/api/v1/jobs",
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result, nil
}

func (c *Client) GetJob(jobID string) (map[string]interface{}, error) {
    resp, err := c.HTTP.Get(c.BaseURL + "/api/v1/jobs/" + jobID)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result, nil
}

func main() {
    client := NewClient("http://localhost:8081")

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
