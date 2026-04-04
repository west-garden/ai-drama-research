# 统一配置 + 多 Provider 设计

> 项目代号：Short Maker
> 日期：2026-04-04
> 状态：设计评审
> 依赖：Plan 1-5（完整 pipeline + Web 控制台）

## 目标

用统一的 YAML 配置文件替代环境变量和 CLI flag，支持多种 LLM/图片/视频 provider 切换。MVP 对接 Gemini（图片 + 视频）和即梦/火山引擎（图片 + 视频）。

## 决策摘要

| 决策项 | 选择 | 理由 |
|-------|------|------|
| 配置格式 | YAML | Go 生态常用，支持注释，结构清晰 |
| 配置存储 | 单文件 config.yaml | 内部工具，不需要 DB 存配置 |
| 图片 provider | Gemini (Imagen) + 即梦 + mock | 用户选择 |
| 视频 provider | Gemini (Veo) + 即梦 + mock | 用户选择 |
| LLM provider | OpenAI 兼容 | 现有实现，通过 base_url 切换不同供应商 |
| Gemini SDK | google.golang.org/genai | 官方 Go SDK，统一处理 Imagen 和 Veo |
| 即梦 SDK | volc-sdk-golang/service/visual | 官方 Go SDK，自动处理 AK/SK 签名 |
| 异步轮询 | adapter 内部封装 | 上层 agent 不感知异步，Generate() 阻塞返回 |

## 配置文件结构

```yaml
# config.yaml
llm:
  provider: openai           # openai（兼容所有 OpenAI 格式 API）
  api_key: sk-xxx
  base_url: https://api.openai.com/v1
  model: gpt-4o-mini

image:
  provider: gemini           # gemini | jimeng | mock
  gemini:
    api_key: AIza...
    model: imagen-4.0-generate-001
  jimeng:
    access_key: AKxxx
    secret_key: SKxxx
    req_key: jimeng_t2i_v40

video:
  provider: jimeng           # gemini | jimeng | mock
  gemini:
    api_key: AIza...
    model: veo-3.1-generate-preview
  jimeng:
    access_key: AKxxx
    secret_key: SKxxx
    req_key: jimeng_vgfm_i2v_l20

output_dir: ./output
db_path: ./shortmaker.db
strategies_path: ""
```

### 设计要点

- `image` 和 `video` 独立选 provider，可以图片用 Gemini、视频用即梦
- 每个 provider 的配置写在对应 section 下，只有被 `provider` 字段选中的才会被加载和校验
- Gemini 的 `api_key` 在 image 和 video 下可以写同一个值
- 即梦的 `access_key`/`secret_key` 同理
- `mock` provider 不需要任何配置字段
- 不存在 config 文件时全部用 mock（向后兼容）

### 默认值

| 字段 | 默认值 |
|------|--------|
| `llm.provider` | `"openai"` |
| `llm.base_url` | `"https://api.openai.com/v1"` |
| `llm.model` | `"gpt-4o-mini"` |
| `image.provider` | `"mock"` |
| `video.provider` | `"mock"` |
| `output_dir` | `"./output"` |
| `db_path` | `"./shortmaker.db"` |

## Go 类型定义

### 新增包：`internal/config/`

```go
// internal/config/config.go
type Config struct {
    LLM            LLMConfig   `yaml:"llm"`
    Image          ImageConfig `yaml:"image"`
    Video          VideoConfig `yaml:"video"`
    OutputDir      string      `yaml:"output_dir"`
    DBPath         string      `yaml:"db_path"`
    StrategiesPath string      `yaml:"strategies_path"`
}

type LLMConfig struct {
    Provider string `yaml:"provider"`
    APIKey   string `yaml:"api_key"`
    BaseURL  string `yaml:"base_url"`
    Model    string `yaml:"model"`
}

type ImageConfig struct {
    Provider string       `yaml:"provider"`
    Gemini   GeminiConfig `yaml:"gemini"`
    Jimeng   JimengConfig `yaml:"jimeng"`
}

type VideoConfig struct {
    Provider string       `yaml:"provider"`
    Gemini   GeminiConfig `yaml:"gemini"`
    Jimeng   JimengConfig `yaml:"jimeng"`
}

type GeminiConfig struct {
    APIKey string `yaml:"api_key"`
    Model  string `yaml:"model"`
}

type JimengConfig struct {
    AccessKey string `yaml:"access_key"`
    SecretKey string `yaml:"secret_key"`
    ReqKey    string `yaml:"req_key"`
}
```

### Load 函数

```go
func Load(path string) (*Config, error)
```

- 读取 YAML 文件 → `yaml.Unmarshal` → 填默认值 → 校验必填字段
- 校验规则：
  - `image.provider` = `"gemini"` 时，`image.gemini.api_key` 必填
  - `image.provider` = `"jimeng"` 时，`image.jimeng.access_key` 和 `secret_key` 必填
  - `video.provider` 同理
  - `llm.provider` != `""` 且 != `"mock"` 时，`llm.api_key` 必填
- 文件不存在时返回全 mock 默认配置（不报错）

## Adapter 实现

### 现有接口（不改）

```go
// internal/router/adapter.go
type ModelAdapter interface {
    Name() string
    Capabilities() Capabilities
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
    HealthCheck(ctx context.Context) error
}
```

### Gemini Adapter

文件：`internal/router/gemini.go`

**GeminiImageAdapter**
- 构造：`NewGeminiImageAdapter(apiKey, model string) (*GeminiImageAdapter, error)`
- 内部创建 `genai.Client`（通过 `google.golang.org/genai`）
- `Generate()`：调用 `client.Models.GenerateImages(ctx, model, prompt, config)`
- 同步返回 base64 图片数据，解码后写入 `req.OutputPath`
- `Capabilities()`：`ModelType = "image"`，`Styles = ["manga", "3d", "live_action"]`

**GeminiVideoAdapter**
- 构造：`NewGeminiVideoAdapter(apiKey, model string) (*GeminiVideoAdapter, error)`
- `Generate()`：
  1. 如果 `req.SourceImage` 非空，读取图片作为输入帧（图生视频）
  2. 调用 `client.Models.GenerateVideos()` 提交任务
  3. 内部轮询：每 10 秒 poll 一次 `client.Operations.GetVideosOperation()`，最长 5 分钟
  4. 完成后通过 `client.Files.Download()` 下载视频写入 `req.OutputPath`
- 支持 context 取消（轮询循环中检查 `ctx.Done()`）
- `Capabilities()`：`ModelType = "video"`，`Styles = ["manga", "3d", "live_action"]`

### 即梦 Adapter

文件：`internal/router/jimeng.go`

**JimengImageAdapter**
- 构造：`NewJimengImageAdapter(ak, sk, reqKey string) *JimengImageAdapter`
- 内部创建 `visual.NewInstance()`，设置 AK/SK
- `Generate()`：调用 `CVProcess`，参数包含 `req_key`、`prompt`、`width`、`height`
- 同步返回 `image_urls`，HTTP GET 下载第一张图片写入 `req.OutputPath`
- `Capabilities()`：`ModelType = "image"`，`Styles = ["manga", "3d", "live_action"]`

**JimengVideoAdapter**
- 构造：`NewJimengVideoAdapter(ak, sk, reqKey string) *JimengVideoAdapter`
- `Generate()`：
  1. 如果 `req.SourceImage` 非空，读取图片用于图生视频（传 `image_urls` 参数）
  2. 调用 `CVSync2AsyncSubmitTask` 提交任务，获取 `task_id`
  3. 内部轮询：每 10 秒调用 `CVSync2AsyncGetResult`，最长 5 分钟
  4. 完成后下载 `video_urls[0]` 写入 `req.OutputPath`
- 支持 context 取消
- `Capabilities()`：`ModelType = "video"`，`Styles = ["manga", "3d", "live_action"]`

### 轮询策略（视频 adapter 共用）

- 轮询间隔：10 秒
- 最大等待：5 分钟（30 次轮询）
- 超时返回 error
- 支持 context 取消（优雅退出）
- 日志：每次 poll 打印状态

## CLI 变更

### 新 flag

| 命令 | flag | 默认值 |
|------|------|--------|
| `run` | `--config` | `./config.yaml` |
| `serve` | `--config` | `./config.yaml` |
| `serve` | `--port` | `8080`（保留） |

### 移除的 flag

`--mock`、`--model`、`--db`、`--output`、`--strategies` 全部移除，由 config.yaml 管理。

### 命令示例

```bash
# 用配置文件跑 pipeline
shortmaker run --config config.yaml script.txt

# 启动 Web 控制台
shortmaker serve --config config.yaml --port 8080

# 不指定 config，全部 mock（开发模式）
shortmaker run script.txt
shortmaker serve
```

### buildAgents 改造

```go
// 旧签名
func buildAgents(useMock bool, llmModel, dbPath, strategyPath, outputDir string) (...)

// 新签名
func buildAgents(cfg *config.Config) (map[agent.Phase]agent.Agent, func(), error)
```

内部根据 `cfg.Image.Provider` / `cfg.Video.Provider` 构造对应 adapter，组装 `ModelRouter`。

## 新增 Go 依赖

| 依赖 | 用途 |
|------|------|
| `gopkg.in/yaml.v3` | YAML 解析 |
| `google.golang.org/genai` | Gemini SDK（Imagen + Veo） |
| `github.com/volcengine/volc-sdk-golang` | 火山引擎 SDK（即梦） |

## 测试策略

### 自动化测试

| 测试 | 验证内容 |
|------|---------|
| `TestLoadConfig` | 正确解析完整 YAML |
| `TestLoadConfig_Defaults` | 缺省字段填默认值 |
| `TestLoadConfig_Validation` | provider=gemini 但缺 api_key 时报错 |
| `TestLoadConfig_FileNotFound` | 文件不存在时返回全 mock 默认配置 |
| `TestGeminiImageAdapter_Name` | Name() 和 Capabilities() 正确 |
| `TestGeminiVideoAdapter_Name` | Name() 和 Capabilities() 正确 |
| `TestJimengImageAdapter_Name` | Name() 和 Capabilities() 正确 |
| `TestJimengVideoAdapter_Name` | Name() 和 Capabilities() 正确 |
| `TestBuildAgents_MockFallback` | 无 config 文件时全部用 mock |
| `TestBuildAgents_FromConfig` | 从 config 构建出正确的 agent map |

### 手动验证

- Gemini 图片生成：上传剧本 → output/ 下有真实 PNG
- 即梦视频生成：output/ 下有真实 MP4
- 混合模式：image=gemini + video=jimeng，完整 pipeline 跑通

## 文件清单

| 操作 | 文件 |
|-----|------|
| 创建 | `internal/config/config.go` |
| 创建 | `internal/config/config_test.go` |
| 创建 | `internal/router/gemini.go` |
| 创建 | `internal/router/jimeng.go` |
| 创建 | `config.example.yaml` |
| 修改 | `cmd/shortmaker/main.go` — 读 config，新签名 buildAgents |
| 修改 | `internal/api/server.go` — NewServer 接受 config（传 outputDir 等） |
| 修改 | `go.mod` — 新增 yaml.v3 + genai + volc-sdk 依赖 |

## API 参考

### Gemini

- 认证：API Key（`x-goog-api-key` header，SDK 自动处理）
- 图片：`client.Models.GenerateImages()` — 同步返回 base64
- 视频：`client.Models.GenerateVideos()` — 异步，返回 operation → 轮询 `GetVideosOperation()` → 下载
- 图生视频：在请求中传入 `image` 参数（base64 图片数据）
- Go SDK：`google.golang.org/genai`

### 即梦 (火山引擎)

- 认证：AK/SK（HMAC-SHA256 签名，SDK 自动处理）
- 图片：`CVProcess` — 同步返回 `image_urls`
- 视频：`CVSync2AsyncSubmitTask` 提交 → `CVSync2AsyncGetResult` 轮询 → 返回 `video_urls`
- 图生视频：请求中传入 `image_urls` 参数
- Go SDK：`github.com/volcengine/volc-sdk-golang/service/visual`
- `req_key` 常用值：`jimeng_t2i_v40`（图片）、`jimeng_vgfm_i2v_l20`（图生视频）、`jimeng_vgfm_t2v_l20`（文生视频）
