# ArcReel 开源项目对比分析

> ArcReel（v0.6.2）是目前发现的**最完整的开源 AI 视频生成工作台**，Python + React 全栈，基于 Claude Agent SDK 做多 Agent 编排。与 short-maker 的 Go 线性 Pipeline 形成鲜明对比。详细代码见 [ArcReel/](../ArcReel/)。

## 项目对比概览

| 维度 | short-maker | ArcReel |
|------|------------|---------|
| 语言 | Go | Python (FastAPI) + React 19 |
| 阶段 | CLI MVP，后端 5 阶段 Pipeline 完成，无前端 | 成熟产品 v0.6.2，前后端完整 |
| AI 编排 | 自研 Orchestrator（确定性顺序执行） | Claude Agent SDK（多 Agent 编排） |
| Provider | 仅 OpenAI + Mock Adapter | 4 家预置（Gemini/Ark/Grok/OpenAI）+ 自定义 |
| 任务模型 | 同步串行 | 异步队列 + 租约 + 心跳 |
| 实时通信 | 无（Plan 5 设计了 SSE 但未实现） | SSE（assistant 流式输出、任务状态、项目变更） |
| 存储 | SQLite | SQLite (开发) / PostgreSQL (生产) + Alembic 迁移 |
| 部署 | 无 | Docker 多阶段构建 + Compose |
| 测试 | 21 个测试文件，全 Mock | 80+ 测试文件，80% 覆盖率要求 |

### 架构模式差异

```
short-maker（线性 Pipeline）:
  Script → StoryAgent → CharacterAgent → StoryboardAgent → ImageGenAgent → VideoGenAgent
  确定性流程，每步输入输出明确，PipelineState 贯穿全程

ArcReel（Agent 驱动）:
  Orchestrator Skill → 分派 Subagent → 各 Subagent 独立执行 → 汇总结果
  灵活但不确定，Agent 自行决策调用哪些工具
```

---

## short-maker 做得好的地方（ArcReel 没有的）

### 1. ImportanceScore 差异化资源分配

short-maker 的三因子乘法评分系统是独特设计，ArcReel 对所有镜头同等对待：

```
Score = EpisodeRole × RhythmPosition × ContentType

EpisodeRole:    hook=1.5, climax=2.0, transition=0.8, resolution=1.2
RhythmPosition: open_hook=1.5, emotion_peak=2.0, mid_narration=0.8, closing_hook=1.2
ContentType:    first_appear=1.5, dialogue=1.0, empty=0.5

→ Grade: S(>=2.0) A(>=1.5) B(>=1.0) C(<1.0)
→ S 级：质量阈值 85，最多重试 3 次
→ C 级：质量阈值 55，不重试
```

这让高潮戏的首次出场镜头获得最多资源，过渡镜头快速通过，符合创作直觉和成本控制需求。

### 2. Strategy Engine 镜头策略匹配

10 种预定义镜头策略 + tag 匹配打分，让分镜生成有专业电影语法支撑。ArcReel 的分镜完全依赖 LLM 自由发挥，缺乏结构化约束。

### 3. Domain 层纯净性

Go 的 interface 契约 + 零外部依赖的 domain 层，比 ArcReel 的 Python 代码边界更清晰。

---

## 从 ArcReel 借鉴的具体点

### 借鉴 1：多 Provider 抽象 — Registry + Factory + 运行时配置

**问题**：short-maker 的 `ModelAdapter` 接口设计方向对了，但只有 Mock 实现，缺乏 Provider 管理基础设施。

**ArcReel 的方案**：
- `ImageBackend` / `VideoBackend` / `TextBackend` 是 Protocol 接口（duck-typed）
- `lib/config/registry.py` 维护 PROVIDER_REGISTRY，预置 4 家供应商
- `lib/custom_provider/` 支持 OpenAI 兼容 / Google 兼容的自定义 Provider
- WebUI `/settings` 运行时配置 API Key、模型选择、速率限制
- 每个 Provider 独立并发池 + RPM 限流

**short-maker 应借鉴的**：
1. Provider 配置管理：不硬编码 API Key，支持运行时配置和切换
2. 并发限流：按 Provider 维度限制 RPM，防止被封
3. 自定义 Provider：OpenAI 兼容接口让用户接入任意模型服务

**不需要照搬的**：Python Protocol 的 duck-typed 设计，Go 的 interface 已经更严格。

### 借鉴 2：异步任务队列 — 并发生成的核心基础设施

**问题**：short-maker 是同步串行执行，一集 20-50 个镜头依次处理太慢。

**ArcReel 的方案**：
- `GenerationQueue`（7.4KB）：SQLAlchemy ORM 持久化的任务队列
- `GenerationWorker`（21KB）：后台 Worker，图片/视频分独立通道
- 租约 + 心跳 + TTL 机制：任务被领取后有 TTL，Worker 定时发心跳续租，超时自动回收
- `enqueue_and_wait()`：调用方阻塞等待结果，队列异步执行
- 按 Provider 分隔并发度：Gemini 可能允许 5 并发，Grok 可能只允许 2 并发

**short-maker 应借鉴的**：
1. 引入并发任务池，ImageGenAgent / VideoGenAgent 不应逐个串行
2. 按 Provider 维度控制并发度（不同 API 的 rate limit 不同）
3. 任务持久化到 SQLite，进程崩溃后可恢复未完成任务

**Go 实现思路**：
```
internal/queue/
  queue.go       # 任务队列接口 + SQLite 实现
  worker.go      # Worker pool，per-provider 并发控制（semaphore）
  lease.go       # 租约管理：claim → heartbeat → complete/expire
```

用 `sync.Semaphore`（或 channel）控制 per-provider 并发度，比 Python asyncio 更直接。

### 借鉴 3：版本管理 + 回滚

**问题**：short-maker 生成结果直接写文件，无历史记录。用户可能觉得上一版更好。

**ArcReel 的方案**：
- `VersionManager`（12KB）：每次生成保留历史版本
- API 支持查看版本列表、回滚到指定版本
- 前端可对比不同版本的生成结果

**short-maker 应借鉴的**：
1. 在 `AssetStore` 增加版本维度：`(shot_id, version) → asset`
2. 生成新版本不删除旧版本，追加存储
3. CLI 支持 `--version` 查看 / 回滚
4. Web Console 展示版本对比

### 借鉴 4：成本追踪

**问题**：图片/视频生成烧钱，short-maker 完全不追踪成本。

**ArcReel 的方案**：
- `CostCalculator`（17KB）：多 Provider 成本计算
- `UsageTracker`（4.6KB）：API 调用量统计
- 前端展示每个项目/每集的累计花费

**short-maker 应借鉴的**：
1. 每次 API 调用记录 provider、model、token 数/图片数、单价
2. 按项目/集/镜头维度汇总成本
3. 结合 ImportanceScore：S 级镜头的高重试次数会消耗更多，用户应能看到

**Go 实现思路**：
```go
type UsageRecord struct {
    ShotID    string
    Provider  string
    Model     string
    Type      string  // "image" | "video" | "text"
    InputCost float64
    CreatedAt time.Time
}
```

### 借鉴 5：SSE 实时进度推送

**问题**：short-maker Plan 5 设计了 SSE 但未实现，Web Console 需要实时展示生成进度。

**ArcReel 的方案**：
- 三种 SSE 通道：assistant 流式输出、任务状态变更、项目文件变更
- `ProjectEventService` 监控文件系统变更并推送事件
- 前端用 `EventSource` 订阅，实时更新 UI

**short-maker 应借鉴的**：
1. Orchestrator 的 `CheckpointFunc` 回调已经预留了进度上报的接口
2. 实现 SSE endpoint，将 CheckpointFunc 的回调事件推送到前端
3. 任务队列的状态变更（pending → running → done/failed）也应推送

### 借鉴 6：角色视觉参考管理

**问题**：short-maker 的 CharacterAgent 只生成文字描述，缺少视觉参考图的管理和复用。

**ArcReel 的方案**：
- 角色设计图（Character Sheet）：为每个角色生成视觉参考图
- 道具设计图（Clue Sheet）：追踪关键道具的视觉一致性
- `project.json` 作为单一数据源，存储角色/道具定义和资产引用
- 生成每个镜头时注入对应角色的参考图

**short-maker 应借鉴的**：
1. CharacterAgent 输出应包含视觉参考图（不仅是文字描述）
2. ImageGenAgent 生成时注入角色参考图，提升一致性
3. 参考 architecture-ideas.md #15 的三级方案：全局管理 + 资产继承 + 融图注入

### 借鉴 7：项目归档 / 导出

**问题**：short-maker 输出只是散落的文件，无法整体打包或导入编辑器。

**ArcReel 的方案**：
- `ProjectArchiveService`（52KB）：项目导出为 ZIP
- `JianyingDraftService`（11KB）：导出为剪映/CapCut 草稿
- 项目导入：ZIP 解包恢复完整项目

**short-maker 应借鉴的**：
1. 项目打包导出（ZIP），包含脚本 + 分镜 + 图片 + 视频 + 元数据
2. 剪映草稿导出（pyjianyingdraft 有 Go 移植可能性，或通过 subprocess 调用）
3. 这是用户实际使用的最后一环——生成完还需要在编辑器里精修

---

## 不需要借鉴的

### Claude Agent SDK 编排 — 过度设计

ArcReel 用 Claude Agent SDK 做了一个**对话式 Copilot**——用户在聊天框里用自然语言指挥工作流。`session_manager.py` 有 79KB 都是 SDK 的胶水代码，`agent_runtime/` 整个目录有 11 个文件专门服务这套架构。

**这是对短剧制作场景的过度设计。** 短剧制作的用户操作是确定性的：

- 选模型、调参数 → 表单 + 下拉框
- 重新生成某个镜头 → 点按钮
- 替换角色参考图 → 上传 + 选择
- 查看生成进度 → 进度条 + 状态列表
- 对比版本 → 图片并排展示

这些操作都是明确的 CRUD，直接映射到 REST API + 按钮交互就够了。绕一圈让 LLM "理解意图"再调工具，增加了成本和延迟，换来的只是"看起来更智能"。用户真正需要的是：**点一下按钮就重新生成这个镜头**，而不是打字告诉 AI "帮我重新生成第 3 集第 5 个镜头"。

引入 Claude Agent SDK 的具体代价：
- **强绑定 Anthropic**：换模型要重写整个 Agent Runtime
- **成本高**：每次操作都走 Claude API（Orchestrator 分派 + Subagent 执行 = 多次 API 调用），一个本来免费的按钮点击变成几毛钱的 API 调用
- **延迟大**：Agent 推理比直接函数调用慢 10-100 倍，用户点"重新生成"后还要等 Agent 思考该怎么做
- **不确定性**：Agent 可能做出意外决策，确定性操作变成概率性操作
- **调试困难**："为什么 Agent 选了这个方案"很难追踪
- **79KB 的 session_manager.py**：这个文件的体量说明了维护这套架构的工程成本

**结论**：短剧制作是**流水线生产**，不是**对话式创作**。确定性 Pipeline + 确定性 UI 操作是正确的架构。LLM 应该用在真正需要"理解"的环节（剧本分析、分镜生成、提示词构建），而不是用来包装按钮点击。

### Python 技术栈

Go 的选择是正确的：单二进制部署、goroutine 天然并发、无 GIL、编译时类型检查。ArcReel 用 Python 的 asyncio + SQLAlchemy async 做并发，Go 用 goroutine + channel 更自然。

### FastAPI + SQLAlchemy

short-maker 用 Go chi + SQLite 就够了，无需引入 ORM 框架（Go 生态里也不推荐重 ORM）。

---

## 优先级排序

按对 short-maker 当前阶段的价值排序：

| 优先级 | 借鉴点 | 理由 |
|--------|--------|------|
| **P0** | 异步任务队列 | 串行生成是当前最大性能瓶颈，影响用户体验 |
| **P0** | 多 Provider 抽象 | Mock → 真实 Provider 的必经之路，也是接下来的开发重点 |
| **P1** | 成本追踪 | 接入真实 Provider 后立刻需要，否则烧钱失控 |
| **P1** | SSE 实时进度 | Web Console（Plan 5）的必备基础设施 |
| **P2** | 角色视觉参考 | 角色一致性是画质的关键，但当前 Mock 阶段不急 |
| **P2** | 版本管理 | 用户体验提升，但 MVP 阶段可简化处理 |
| **P3** | 项目归档/导出 | 产品完善阶段再做 |
