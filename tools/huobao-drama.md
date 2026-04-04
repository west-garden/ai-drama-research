# 火宝短剧 (Huobao Drama)

> 基于 AI Agent 的一站式短剧生成平台，从小说到成片全自动化

- **官网:** https://github.com/chatfire-AI/huobao-drama
- **类型:** 端到端平台（开源）
- **Stars:** 9,552 | **Forks:** 1,801
- **技术栈:** TypeScript / Hono (后端) / Nuxt 3 + Vue 3 (前端) / SQLite + Drizzle ORM / Mastra Agent / FFmpeg
- **调研日期:** 2026-04-03

## 核心能力

将小说/剧本文本作为输入，通过 5 个专用 AI Agent 协作，全自动生成短剧视频。覆盖剧本改写、角色场景提取、分镜拆解、图片生成、视频生成、配音合成、FFmpeg 合成拼接的完整链路。

核心特色是 **Skill-as-prompt** 架构——每个 Agent 的行为规范写成可编辑的 SKILL.md 文件，前端 UI 可直接修改，实现 prompt 与代码解耦。另有"火宝预设"一键配置功能，通过 `api.chatfire.site` 统一网关快速接入 AI 服务。

## 架构

### 整体架构

```
[浏览器 Nuxt 3 SPA] → devProxy → [Hono API Server :3301]
                                        │
                                   /api/v1/*
                                        │
                    ┌───────────────────┼───────────────────┐
                    │                   │                   │
              [SQLite DB]        [Mastra Agents]     [媒体服务]
              Drizzle ORM        5 个专用 Agent      Image/Video/TTS
                                 SKILL.md prompt     FFmpeg 合成
                                        │
                                 [多供应商适配器]
                                 8 家 AI 供应商
                                        │
                              [uploads/ 本地存储]
```

### 前后端分离

- **后端:** Hono HTTP 框架，17 个路由模块，50+ API 端点
- **前端:** Nuxt 3 SPA，4 个页面（项目列表、设置、剧本详情、分镜工作台），2 个布局

### 多供应商适配器

4 个统一接口（TextAdapter / ImageAdapter / VideoAdapter / AudioAdapter）+ Map 注册表：

| 供应商 | 文本 | 图像 | 视频 | 语音 |
|--------|:----:|:----:|:----:|:----:|
| chatfire (统一网关) | ✓ | ✓ | ✓ | ✓ |
| ali (阿里) | ✓ | ✓ | ✓ | |
| openai | ✓ | ✓ | | ✓ |
| gemini | ✓ | ✓ | | |
| volcengine (火山) | ✓ | | ✓ | ✓ |
| minimax | | | ✓ | |
| vidu | | | ✓ | |
| openrouter | ✓ | | | |

默认预设（火宝预设）：文本 `gemini-3-pro-preview`，图像 `gemini-3-pro-image-preview`，视频 `doubao-seedance-1-5-pro-251215`，语音 `speech-2.8-hd`。

### Agent 系统

基于 Mastra 框架，5 个专用 Agent，每个 Agent 加载对应的 SKILL.md 作为系统提示词：

| Agent | 职责 | 核心工具 |
|-------|------|---------|
| script_rewriter | 小说 → 格式化剧本 | read_episode_script, rewrite_to_screenplay, save_script |
| extractor | 角色/场景提取+去重 | read_script_for_extraction, save_dedup_characters, save_dedup_scenes |
| grid_prompt_generator | 参考图 prompt 生成 | 3 模式：首帧/首尾帧/多参考 |
| voice_assigner | 角色配音匹配 | 根据性别、年龄、性格匹配 TTS 声线 |
| storyboard_breaker | 分镜拆解 (17 字段/镜头) | read_storyboard_context, save_storyboards, update_storyboard |

## 工作流

```
                        ┌──────────────────┐
                        │  小说/剧本文本    │
                        └────────┬─────────┘
                                 │
                    ┌────────────▼────────────┐
                    │  1. 剧本改写             │
                    │  (script_rewriter)       │
                    │  小说 → 格式化剧本       │
                    │  场景头+舞台指示+对白     │
                    └────────────┬─────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │  2. 角色/场景提取         │
                    │  (extractor)             │
                    │  提取角色(300-500字外貌)  │
                    │  提取场景 + 跨集去重      │
                    └────────────┬─────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                   │
   ┌──────────▼──────────┐  ┌───▼────────────┐  ┌──▼───────────────┐
   │ 3a. 角色图生成       │  │ 3b. 场景图生成  │  │ 3c. 配音分配      │
   │ (grid_prompt_gen +   │  │ (grid_prompt   │  │ (voice_assigner) │
   │  图片生成服务)       │  │  + 生成服务)    │  │ 匹配 TTS 声线    │
   └──────────┬───────────┘  └───┬────────────┘  └──────┬───────────┘
              │                  │                      │
              └──────────┬───────┘                      │
                         │                              │
              ┌──────────▼─────────────┐                │
              │  4. 分镜拆解            │                │
              │  (storyboard_breaker)   │                │
              │  剧本 → 逐镜头拆解      │                │
              │  每镜头 17 个字段        │                │
              └──────────┬─────────────┘                │
                         │                              │
         ┌───────────────┼──────────────────┐           │
         │               │                  │           │
  ┌──────▼──────┐  ┌─────▼──────┐  ┌───────▼───────────▼──┐
  │ 5a. Grid    │  │ 5b. I2V    │  │ 5c. TTS 语音合成     │
  │ 参考图生成  │  │ 图生视频   │  │ (逐镜头对白音频)     │
  │ + 切分      │  │ (逐镜头)   │  │                      │
  └──────┬──────┘  └─────┬──────┘  └───────┬──────────────┘
         │               │                 │
         └───────────────┼─────────────────┘
                         │
              ┌──────────▼─────────────┐
              │  6. FFmpeg 合成         │
              │  (逐镜头)              │
              │  视频 + 音频 + 字幕     │
              └──────────┬─────────────┘
                         │
              ┌──────────▼─────────────┐
              │  7. FFmpeg 拼接         │
              │  所有镜头 → 完整单集     │
              └────────────────────────┘
```

### 各阶段细节

**Stage 1 — 剧本改写：** 将叙事散文转为格式化剧本，场景头格式为 `## S01 | 内景 · 地点 | 时间段`，对白格式为 `角色名：（状态）台词`，每场景目标 30-60 秒。

**Stage 2 — 角色/场景提取：** 从剧本中提取角色（含 300-500 字外貌描述、性格、角色定位）和场景（位置、时间、氛围、英文图片 prompt），自动与项目级已有数据去重。

**Stage 3 — 素材准备：** grid_prompt_generator 生成英文 prompt，支持 3 种模式（first_frame/first_last/multi_ref）；voice_assigner 根据角色特征匹配 TTS 声线。

**Stage 4 — 分镜拆解：** 每镜头 17 个字段：title, shot_type, angle, movement, location, time, character_ids, action, dialogue, description, result, atmosphere, image_prompt, video_prompt, bgm_prompt, sound_effect, duration, scene_id。视频 prompt 使用 3 秒分段 + XML 标签格式（`<location>`, `<role>`, `<voice>`, `<n>`）。

**Stage 5 — 媒体生成：** Grid 参考图通过 sharp 切分为单帧；I2V 视频生成采用异步模式（提交任务 → 轮询状态）；TTS 逐镜头生成对白音频。

**Stage 6/7 — FFmpeg 合成拼接：** fluent-ffmpeg + ffmpeg-static，逐镜头合成（视频+音频+字幕），最终拼接为完整单集。

## 功能清单

| 功能 | 支持情况 | 备注 |
|------|---------|------|
| 剧本生成 | ✅ 完整 | Agent 驱动，小说 → 格式化剧本，场景头+舞台指示+对白 |
| 角色设计 | ✅ 完整 | AI 提取角色，300-500 字外貌描述，支持跨集去重 |
| 分镜生成 | ✅ 完整 | 单步 Agent 生成，17 字段/镜头，含视频 prompt 和 BGM prompt |
| 图片/画面生成 | ✅ 完整 | 8 供应商，Grid 参考图 3 种模式（首帧/首尾帧/多参考） |
| 视频生成 | ✅ 完整 | 图生视频 (I2V)，异步轮询 + webhook 回调 |
| 语音合成 | ✅ 基础 | TTS 逐镜头生成，无声音克隆/情感控制 |
| 口型同步 | ❌ | 不支持 |
| 字幕生成 | ✅ 基础 | FFmpeg 字幕覆盖 |
| 配乐 | ⚠️ 有 prompt | 分镜中有 bgm_prompt 字段，但无 AI 配乐生成 |
| 多集连续性 | ✅ 完整 | Drama → Episode 两层结构，角色/场景跨集共享去重 |
| 批量生成 | ⚠️ 有限 | 支持批量 TTS 和批量合成，但无持久化队列保障 |

## AI 模型集成

| 类别 | 供应商 | 默认预设模型 |
|------|--------|-------------|
| LLM | chatfire, ali, openai, gemini, volcengine, openrouter | gemini-3-pro-preview |
| 图片 | chatfire, ali, openai, gemini | gemini-3-pro-image-preview |
| 视频 | chatfire, ali, volcengine, minimax, vidu | doubao-seedance-1-5-pro-251215 |
| 语音 | chatfire, openai, volcengine | speech-2.8-hd |

## AI 能力边界

- **能自动完成的：** 小说→剧本→分镜→媒体生成的完整链路；角色/场景的 AI 提取与去重；参考图 prompt 生成与 Grid 切分；逐镜头视频+音频+字幕合成拼接
- **需要人工介入的：** Agent 行为调优（通过编辑 SKILL.md）；角色形象确认；分镜结果审核；AI 服务配置（或使用一键预设）
- **完全做不到的：** 口型同步；复杂视频特效/转场；AI 配乐生成；声音克隆/情感控制；实时预览

## 数据模型

### 核心表

```
dramas (项目)
  ├── episodes (剧集)
  │     ├── storyboards (分镜，17 字段)
  │     ├── episode_characters (多对多)
  │     └── episode_scenes (多对多)
  ├── characters (角色，含外貌/性格/配音/头像)
  └── scenes (场景，含 prompt/图片)

ai_service_configs (AI 服务配置，Episode 创建时锁定)
agent_configs (Agent LLM 配置)
ai_voices (TTS 声线目录)
images (生成的图片)
videos (生成的视频，含异步状态追踪)
```

### 设计特点

- **Episode 级配置锁定：** 创建剧集时冻结引用的 AI 服务配置（image_config_id, video_config_id, audio_config_id），保证同一集生产一致性
- **角色/场景跨集共享：** 通过 drama_id 关联到项目级，通过中间表关联到具体 episode，支持去重复用
- **异步视频追踪：** videos 表有 status + task_id 字段，支持提交-轮询-回调模式

## API 端点

17 个路由模块，约 50+ 端点：

| 模块 | 路由前缀 | 核心端点 |
|------|---------|---------|
| Dramas | `/api/v1/dramas` | CRUD |
| Episodes | `/api/v1/episodes` | CRUD + 状态查询 |
| Storyboards | `/api/v1/storyboards` | CRUD + 单条/批量 TTS |
| Characters | `/api/v1/characters` | CRUD + 图片生成 + 语音生成 |
| Scenes | `/api/v1/scenes` | CRUD + 图片生成 |
| Agent | `/api/v1/agent` | `POST /:type/chat` (SSE 流式) |
| Images | `/api/v1/images` | 列表 + 生成 |
| Grid | `/api/v1/grid` | prompt 生成 + 图片生成 + 状态 + 切分 |
| Videos | `/api/v1/videos` | 列表 + 生成 + 状态轮询 |
| Compose | `/api/v1/compose` | 单镜头合成 + 批量合成 + 状态 |
| Merge | `/api/v1/merge` | 全集拼接 + 状态 |
| AI Configs | `/api/v1/ai-configs` | CRUD + 测试 + 火宝预设 |
| Agent Configs | `/api/v1/agent-configs` | CRUD |
| Skills | `/api/v1/skills` | CRUD (SKILL.md 内容) |
| Voices | `/api/v1/voices` | 列表 + 同步 |
| Webhooks | `/api/v1/webhooks` | 视频生成回调 |
| Upload | `/api/v1/upload` | 文件上传 |

## 架构评价

### 整体架构设计：7/10

前后端分离 + SQLite 零依赖，架构整洁但缺少生产环境必要的基础设施（任务队列、重试、监控）。

### 各模块评价

#### Skill-as-prompt 系统：9/10 — 全项目最亮眼的设计

SKILL.md 既是文档也是运行时 prompt，前端 UI 可编辑。非技术用户可调整 Agent 行为，prompt 与代码版本一致。这种设计在开源 AI 项目中少见。

#### 多供应商适配器：8/10 — 干净的抽象

4 个统一接口 + Map 注册表，新增供应商只需实现接口。比 switch-case 分发更易维护。

#### 分镜拆解：6.5/10 — 单步生成的局限

一次性生成 17 个字段，对 LLM 压力大。没有中间校验步骤，出错只能整体重来。Video prompt 的 3 秒分段 + XML 标签格式是有意思的尝试，但没有验证机制。

#### 媒体生成管线：5/10 — 缺少可靠性保障

无持久化队列，无重试机制，无心跳/看门狗，服务重启丢失进行中任务。批量操作无并发控制。对于视频生成这种耗时任务，这是严重短板。

#### FFmpeg 合成：7/10 — 够用但扩展性差

fluent-ffmpeg 做基础的视频+音频+字幕合成没问题，但无法做转场效果、图层合成、文字特效。相比 Remotion 等可编程方案，扩展成本高。

#### 数据模型：7.5/10 — 设计合理

Episode 级配置锁定、角色/场景跨集去重都是好设计。Drizzle ORM + SQLite 轻量高效。但部分字段（如 character_ids 存为 JSON 数组）牺牲了关系查询能力。

### 设计风格总评

| 维度 | 评分 | 说明 |
|------|------|------|
| 架构完整度 | 6.5/10 | 链路完整但缺少队列、计费、监控等生产组件 |
| 代码质量 | 7/10 | 质量较均匀，没有明显的模块间落差 |
| 抽象合理性 | 7.5/10 | 适配器和 Skill 系统抽象合理，Agent 框架偏重 |
| 可扩展性 | 6/10 | 前后端分离好，但无队列无分布式限制了规模化 |
| 可靠性设计 | 4/10 | 无重试、无看门狗、无心跳，生产环境不可靠 |
| 上手体验 | 9/10 | 零依赖部署 + 一键预设 + 前端可编辑 Skill |

**一句话总评：** 产品思维驱动的项目——用户体验和 prompt 管理做得好，但工程深度不够，缺少生产环境必要的可靠性保障。适合个人使用和概念验证，不适合直接用于多用户生产环境。

## 可编程性

- **API:** 50+ REST API 端点，Agent 聊天支持 SSE 流式
- **批量操作:** 支持批量 TTS 和批量合成，但无队列保障
- **可集成到自动化流水线:** 可以，API 完整，但需自行处理重试和错误恢复
- **Skill 可编程:** SKILL.md 文件可通过 API 或 UI 修改，实现运行时 prompt 调整

## 定价

| 方案 | 价格 | 包含内容 |
|------|------|---------|
| 开源自部署 | 免费 | 完整功能，自带 API Key |
| 火宝预设 | 免费 | 通过 chatfire 网关使用，需注册 chatfire 账号 |

## 优点

- **零依赖部署**，SQLite + 本地文件系统，`npm run dev` 即可运行
- **Skill-as-prompt** 设计精巧，前端可编辑 Agent 行为，prompt 与代码解耦
- **一键火宝预设**，快速接入 AI 服务，降低配置门槛
- **多供应商适配器干净**，4 个统一接口 + Map 注册表
- **Episode 级配置锁定**，保证同一集生产一致性
- **前后端分离**，架构整洁，职责清晰
- **Grid 参考图 3 模式**，提供不同精度的角色一致性控制
- **代码质量较均匀**，没有明显的模块间质量落差

## 缺点

- **无任务队列**，服务重启丢失任务，无重试/无看门狗/无心跳
- **无计费系统**，多用户场景需从零开始
- **无口型同步**，角色说话嘴型不动是明显的观感问题
- **分镜单步生成**，17 字段一次产出，对 LLM 压力大，出错只能整体重来
- **FFmpeg 合成能力有限**，无转场效果、无图层合成、无文字特效
- **Mastra Agent 框架偏重**，实际需求用简单函数调用即可满足
- **TTS 能力基础**，无声音克隆、无情感控制
- **无 AI 配乐生成**，分镜有 bgm_prompt 字段但未接入生成服务

## 对自建系统的启发

### 值得借鉴

1. **Skill-as-prompt 模式** — 将 Agent 行为规范写成可编辑的 markdown 文件，前端可直接修改，实现 prompt 与代码解耦。这是管理 AI Agent prompt 的优雅方案
2. **适配器接口 + Map 注册表** — 比 switch-case 更干净的多供应商管理，新增供应商只需实现 4 个接口之一
3. **零依赖优先** — SQLite 优先，需要时再升级，降低上手门槛。不是所有项目都需要 MySQL + Redis
4. **Episode 级配置锁定** — 创建剧集时冻结 AI 配置快照，避免生产过程中配置变更导致不一致
5. **Grid 参考图的多模式设计** — 首帧/首尾帧/多参考图 3 种模式，在成本和一致性之间提供灵活选择
6. **Video prompt 的 3 秒分段格式** — 用 XML 标签结构化视频 prompt，比纯文本描述更可控

### 应该避免

1. **无任务队列就上生产** — 视频生成是典型的长耗时异步任务，没有持久化队列和重试机制会导致任务丢失
2. **外部 Agent 框架过早引入** — Mastra 的 Agent 是有状态聊天模式，对流水线批处理场景过重。简单的函数编排 + prompt 加载即可满足需求
3. **分镜单步生成** — 17 个字段一次产出对 LLM 压力过大，应拆分为多阶段（如 waoowaoo 的 4 阶段流水线）
4. **忽略口型同步** — 作为短剧产品，嘴型不动是非常明显的质量短板，应在早期就纳入管线
