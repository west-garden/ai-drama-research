# Plan 4 设计：Image/Video Generation Agent + Model Router + Quality Check

> 项目代号：Short Maker
> 日期：2026-04-04
> 状态：设计评审
> 依赖：Plan 1 (domain types, Agent interface, Orchestrator, ImportanceScore), Plan 2 (StoryAgent, CharacterAgent), Plan 3 (StoryboardAgent, Strategy Engine)

## 目标

实现流水线的 Phase 4（图片生成）和 Phase 5（视频生成），以及支撑它们的基础设施：ModelRouter（模型路由）和 QualityChecker（质检）。本轮全部使用 mock 实现，建立正确的接口和闭环流程，为后续真实模型接入做好准备。

## 决策摘要

| 决策项 | 选择 | 理由 |
|-------|------|------|
| 模型 API | 全部 mock | 先跑通架构和闭环，真实 API 下一轮接入 |
| Quality Check | 接口 + mock（始终通过） | 让"生成→评估→重试"流程跑通 |
| 产物存储 | 目录结构 `output/<project>/<ep>/<shot>` | 简单直观，mock 阶段够用 |
| 并行生成 | 串行 | mock 阶段性能不是问题，串行便于调试 |
| 架构 | 分层路由（独立 router + quality 包） | 符合 spec 架构方向，新模型 = 实现接口 |

## 架构

### 依赖方向

```
cmd/shortmaker/main.go
    ↓
agent (ImageGenAgent, VideoGenAgent)
    ↓          ↓
router      quality
    ↓          ↓
domain      domain
```

严格单向依赖。`router` 和 `quality` 互不依赖，都只依赖 `domain`。

### 新增包

#### `internal/router/` — 模型路由

负责根据镜头重要性等级选择合适的生成模型，并统一调用接口。

#### `internal/quality/` — 质量检查

负责评估生成结果的质量，返回结构化评分报告。

## 接口设计

### ModelAdapter

```go
// internal/router/adapter.go
package router

type ModelType string
const (
    ModelTypeImage ModelType = "image"
    ModelTypeVideo ModelType = "video"
)

type Capabilities struct {
    Type           ModelType // image 或 video
    Styles         []string  // 支持的风格：manga, 3d, live_action
    MaxResolution  string    // "1024x1024"
    SupportsFusion bool      // 是否支持角色参考图融合
}

type GenerateRequest struct {
    Prompt        string            // 生成提示词
    Style         string            // 风格
    CharacterRefs []string          // 角色参考图路径（融图用）
    CameraMove    string            // 运镜指令（视频用）
    SourceImage   string            // 源图片路径（视频生成用）
    OutputPath    string            // 输出文件路径
    Metadata      map[string]string // 扩展参数
}

type GenerateResponse struct {
    FilePath   string  // 生成文件的实际路径
    ModelUsed  string  // 实际使用的模型名
    Cost       float64 // 本次调用成本（美元）
    DurationMs int64   // 耗时毫秒
}

type ModelAdapter interface {
    Name() string
    Capabilities() Capabilities
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
    HealthCheck(ctx context.Context) error
}
```

### ModelRouter

```go
// internal/router/router.go
type ModelRouter struct {
    adapters []ModelAdapter
}

func NewModelRouter(adapters ...ModelAdapter) *ModelRouter

// Route 选择适配器：
// 1. 按 ModelType 过滤
// 2. 按 Style 过滤（adapter.Capabilities().Styles 包含请求的 style）
// 3. MVP 阶段返回第一个匹配的（后续根据 Grade 做优先级排序）
func (r *ModelRouter) Route(grade domain.Grade, style string, modelType ModelType) (ModelAdapter, error)

// Generate 组合 Route + adapter.Generate
func (r *ModelRouter) Generate(ctx context.Context, grade domain.Grade, style string, modelType ModelType, req GenerateRequest) (*GenerateResponse, error)
```

### QualityChecker

```go
// internal/quality/checker.go
package quality

type Dimension struct {
    Name   string  // 维度名
    Weight float64 // 权重
    Score  int     // 0-100
    Notes  string  // 扣分说明
}

type QualityReport struct {
    ShotNumber  int         // 对应镜头号
    Dimensions  []Dimension // 5 个维度的评分
    TotalScore  int         // 加权总分 (0-100)
    Passed      bool        // 是否通过
    Suggestions []string    // 改进建议
}

type Checker interface {
    Check(ctx context.Context, filePath string, shotSpec *domain.ShotSpec, characterAssets []*domain.Asset) (*QualityReport, error)
}
```

5 个评估维度（来自 spec Section 九）：

| 维度 | 权重 | 说明 |
|-----|------|------|
| character_consistency | 30% | 与角色资产的视觉匹配度 |
| image_quality | 25% | 清晰度、伪影、畸变 |
| storyboard_fidelity | 20% | 景别/构图/运镜是否符合分镜 |
| style_consistency | 15% | 与项目风格模板的匹配度 |
| narrative_accuracy | 10% | 画面是否准确表达剧本意图 |

通过阈值由 `ImportanceScore.Grade().QualityThreshold()` 决定（已在 Plan 1 中实现）：S≥85, A≥75, B≥65, C≥55。

## Mock 实现

### MockImageAdapter

- `Name()`: "mock-image"
- `Capabilities()`: Type=image, Styles=["manga","3d","live_action"], SupportsFusion=false
- `Generate()`: 在 `req.OutputPath` 创建一个最小 PNG 文件（1x1 像素），返回路径
- `HealthCheck()`: 始终返回 nil

### MockVideoAdapter

- `Name()`: "mock-video"
- `Capabilities()`: Type=video, Styles=["manga","3d","live_action"]
- `Generate()`: 在 `req.OutputPath` 创建一个空文件作为占位，返回路径
- `HealthCheck()`: 始终返回 nil

### MockChecker

- `Check()`: 始终返回 QualityReport{TotalScore: 90, Passed: true}，5 个维度各 90 分

## Agent 设计

### ImageGenAgent

**Phase**: `image_generation`

**依赖注入**：
- `*router.ModelRouter` — 模型路由
- `quality.Checker` — 质检
- `outputDir string` — 输出根目录

**输入**（从 PipelineState 读取）：
- `Storyboard` — ShotSpec 列表
- `Blueprint` — 获取 EpisodeRole（用于 ImportanceScore）
- `Assets` — 角色参考资产

**输出**（写入 PipelineState）：
- `Images` — GeneratedShot 列表

**核心流程**：
```
for each ShotSpec in Storyboard:
    1. 从 Blueprint 中找到该 ShotSpec 对应的 EpisodeRole
    2. 计算 ImportanceScore(episodeRole, rhythmPosition, contentType)
    3. 收集角色参考资产（ShotSpec.CharacterRefs → Assets 查找）
    4. 构造输出路径：outputDir/<project_id>/ep<N>/shot<M>.png
    5. 确保目录存在（os.MkdirAll）
    6. 构造 GenerateRequest
    7. Router.Generate(grade, style, "image", req)
    8. QualityChecker.Check(生成文件, shotSpec, 角色资产)
    9. if !passed && retryCount < grade.MaxRetries():
         retryCount++
         goto step 7
    10. 记录 GeneratedShot{ShotNumber, EpisodeNum, ImagePath, Grade, ImageScore}
```

### VideoGenAgent

**Phase**: `video_generation`

**依赖注入**：同 ImageGenAgent

**输入**：
- `Images` — 已生成的图片列表
- `Storyboard` — 运镜指令（CameraMove）

**输出**：
- `Videos` — GeneratedShot 列表（复用 Images 的数据，填充 VideoPath 和 VideoScore）

**核心流程**：
```
for each GeneratedShot in Images:
    1. 找到对应的 ShotSpec（按 ShotNumber 匹配）
    2. 使用已有的 Grade（和 ImageGen 阶段一致）
    3. 构造输出路径：outputDir/<project_id>/ep<N>/shot<M>.mp4
    4. 构造 GenerateRequest（SourceImage = ImagePath, CameraMove = shotSpec.CameraMove）
    5. Router.Generate(grade, style, "video", req)
    6. QualityChecker.Check(生成文件, shotSpec, 角色资产)
    7. 重试逻辑同上
    8. 复制 GeneratedShot 并填充 VideoPath + VideoScore，追加到 Videos 列表
```

## 数据结构扩展

### PipelineState 新增字段

```go
// agent/agent.go — PipelineState
type GeneratedShot struct {
    ShotNumber  int          `json:"shot_number"`
    EpisodeNum  int          `json:"episode_number"`
    ImagePath   string       `json:"image_path"`
    VideoPath   string       `json:"video_path"`
    Grade       domain.Grade `json:"grade"`
    ImageScore  int          `json:"image_score"`
    VideoScore  int          `json:"video_score"`
}

// PipelineState 新增：
Images  []*GeneratedShot `json:"images,omitempty"`
Videos  []*GeneratedShot `json:"videos,omitempty"`
```

### 输出路径约定

```
<outputDir>/<project_id>/ep<N>/shot<M>.png    — 图片
<outputDir>/<project_id>/ep<N>/shot<M>.mp4    — 视频
```

`outputDir` 通过 CLI `--output` flag 传入，默认 `./output`。

## CLI 变更

### 新增 flags

```go
runCmd.Flags().String("output", "./output", "Output directory for generated files")
```

### buildAgents 变更

```go
// 创建 ModelRouter
imageAdapter := router.NewMockImageAdapter()
videoAdapter := router.NewMockVideoAdapter()
modelRouter := router.NewModelRouter(imageAdapter, videoAdapter)

// 创建 QualityChecker
checker := quality.NewMockChecker()

// 创建 Agents
agents[agent.PhaseImageGeneration] = agent.NewImageGenAgent(modelRouter, checker, outputDir)
agents[agent.PhaseVideoGeneration] = agent.NewVideoGenAgent(modelRouter, checker, outputDir)
```

## 测试策略

### router 包

| 测试 | 验证内容 |
|-----|---------|
| TestMockImageAdapter_Generate | 创建占位 PNG 文件，返回正确路径 |
| TestMockVideoAdapter_Generate | 创建占位 MP4 文件，返回正确路径 |
| TestModelRouter_RouteImage | 正确选择 image 类型 adapter |
| TestModelRouter_RouteVideo | 正确选择 video 类型 adapter |
| TestModelRouter_NoMatch | 无匹配 adapter 时返回 error |
| TestModelRouter_StyleFilter | 按 style 过滤 adapter |

### quality 包

| 测试 | 验证内容 |
|-----|---------|
| TestMockChecker_AlwaysPasses | 返回 Passed=true, TotalScore=90 |
| TestQualityReport_Dimensions | 5 个维度权重和 = 1.0 |
| TestQualityReport_WeightedScore | 加权计算正确 |

### agent 包

| 测试 | 验证内容 |
|-----|---------|
| TestImageGenAgent_Run | 为每个 ShotSpec 生成图片，路径正确，Images 列表完整 |
| TestImageGenAgent_Phase | 返回 PhaseImageGeneration |
| TestImageGenAgent_NilStoryboard | Storyboard 为空时报错 |
| TestImageGenAgent_ImportanceGrade | 不同 ShotSpec 产生正确的 Grade |
| TestVideoGenAgent_Run | 基于 Images 生成视频，路径正确，Videos 列表完整 |
| TestVideoGenAgent_Phase | 返回 PhaseVideoGeneration |
| TestVideoGenAgent_NilImages | Images 为空时报错 |
| TestImageGenAgent_RetryOnFailure | 质检不通过时重试，重试次数和 Grade 一致 |
| TestIntegration_FullPipelineWithGeneration | 5 阶段完整端到端（全 mock） |

### 重试测试

用自定义 mock：
- `FailOnceAdapter`：第一次 Generate 返回 error，第二次成功
- `FailOnceChecker`：第一次 Check 返回 Passed=false，第二次 Passed=true

验证：S 级镜头最多重试 3 次，B 级镜头最多重试 1 次，C 级不重试。

## 文件清单

| 操作 | 文件 |
|-----|------|
| 创建 | `internal/router/adapter.go` |
| 创建 | `internal/router/adapter_test.go` |
| 创建 | `internal/router/router.go` |
| 创建 | `internal/router/router_test.go` |
| 创建 | `internal/router/mock_image.go` |
| 创建 | `internal/router/mock_video.go` |
| 创建 | `internal/quality/checker.go` |
| 创建 | `internal/quality/checker_test.go` |
| 创建 | `internal/quality/mock_checker.go` |
| 创建 | `internal/agent/imagegen.go` |
| 创建 | `internal/agent/imagegen_test.go` |
| 创建 | `internal/agent/videogen.go` |
| 创建 | `internal/agent/videogen_test.go` |
| 修改 | `internal/agent/agent.go` — 新增 GeneratedShot + PipelineState 字段 |
| 修改 | `internal/agent/integration_test.go` — 新增端到端测试 |
| 修改 | `cmd/shortmaker/main.go` — wire 新 agents + --output flag |
