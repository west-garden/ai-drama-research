# 工具全景对比表

## 漫剧/短剧端到端工具

| 工具 | 类型 | 剧本 | 画面 | 语音 | 口型 | 字幕 | API | 评价 |
|------|------|------|------|------|------|------|-----|------|
| [巨日禄](https://video.jurilu.com/) | 漫剧+真人 | ❌ | ✅ 融图+多模型 | ❌ | ❌ | ? | ❌ | 行业头部，融图技术成熟，案例验证充分 |
| [FreeToon](https://www.freetoon.com/) | 漫剧 | ❌ | ✅ 多模型+资产继承 | ✅ 腾讯云AI配音 | ? | ✅ 30+语种 | ❌ | 内容一线验证，全集思维，40%爆款率，先锋测试中 |
| [火山剧创/DramArt](https://dramart.volcengine.com/) | 漫剧+真人 | ❌ | ✅ Seedance 2.0 | ? | ? | ? | ? | 多Agent协同架构，内置200+爆款策略，字节生态闭环，邀测中 |
| [小云雀短剧Agent](https://xyq.jianying.com/) | 漫剧+真人 | ✅ AI生成 | ✅ Seedance 2.0 | ✅ 原生音画一体 | ✅ 原生支持 | ? | ❌ | 字节C端，免费，故事理解+全局角色管理，《万兽独尊》4天破亿，Seedance 2.0画质顶流 |
| [waoowaoo](https://github.com/saturndec/waoowaoo) | 端到端开源 | ✅ 4阶段分镜 | ✅ 7+供应商 | ✅ TTS+克隆 | ✅ 3供应商 | ✅ Remotion | ✅ 100+ API | 工程能力强（计费/队列/租约），分镜生成代码粗糙，Next.js单体+BullMQ，部署偏重 |
| [火宝短剧](https://github.com/chatfire-AI/huobao-drama) | 端到端开源 | ✅ Agent编排 | ✅ 8供应商 | ✅ TTS | ❌ | ✅ FFmpeg | ✅ 50+ API | 轻量务实（SQLite零依赖），Skill-as-prompt可编辑，缺队列/计费/口型，适合个人使用 |
| [Jellyfish](https://github.com/Forget-C/Jellyfish) | 端到端开源 | ✅ 12 Agent链 | ✅ 多供应商 | ? | ✅ 提及 | ✅ 提示词级 | ✅ FastAPI | Agent编排最深（12个LangGraph Agent+可视化编辑器），资产管理最精细（角色/场景/道具/服装四类+双层资产库），项目太新（2026.03），核心流程未稳定 |

## 通用 AI 视频创作平台

| 工具 | 类型 | 剧本 | 画面 | 语音 | 口型 | 字幕 | API | 评价 |
|------|------|------|------|------|------|------|-----|------|
| [LibTV](https://www.liblib.tv/) | 通用视频创作 | ✅ 脚本节点 | ✅ 30+模型+三视图 | ? | ? | ? | ✅ MIT开源Skill | 首个Creator+Agent双入口，30+模型集成，价格比Runway/Pika低76-92%，非漫剧垂直但架构设计对自建系统有重要参考价值 |

## 视频生成底层工具

| 工具 | 画质 | 时长 | 一致性 | 可控性 | API | 价格 | 评价 |
|------|------|------|--------|--------|-----|------|------|
| | | | | | | | |

## 图片/漫画生成工具

| 工具 | 风格 | 角色一致性 | 分镜控制 | API | 价格 | 评价 |
|------|------|-----------|---------|-----|------|------|
| | | | | | | |

## 语音/配音工具

| 工具 | TTS质量 | 克隆 | 多语言 | 情感控制 | API | 价格 | 评价 |
|------|---------|------|--------|---------|-----|------|------|
| | | | | | | | |

---

## 开源端到端平台深度对比：火宝短剧 vs waoowaoo vs Jellyfish

> 三个项目定位相同——从文本到短剧视频的全自动化流水线——但设计哲学截然不同。

### 基础信息

| | 火宝短剧 | waoowaoo | Jellyfish |
|---|---------|----------|-----------|
| GitHub | [chatfire-AI/huobao-drama](https://github.com/chatfire-AI/huobao-drama) | [saturndec/waoowaoo](https://github.com/saturndec/waoowaoo) | [Forget-C/Jellyfish](https://github.com/Forget-C/Jellyfish) |
| Stars | 9,552 | 10,778 | 2,250 |
| 创建时间 | 较早 | 较早 | 2026-03-06 |
| 架构模式 | 前后端分离 (Hono + Nuxt 3) | Next.js 单体 + Worker 进程 | 前后端分离 (FastAPI + React 18) |
| 语言 | TypeScript 100% | TypeScript 100% | TypeScript 57% + Python 41% |
| 数据库 | SQLite (零外部依赖) | MySQL + Redis | MySQL (可用 SQLite) |
| 任务队列 | 无（提交+轮询） | BullMQ + Redis | 未知 |
| 视频合成 | FFmpeg 命令行 | Remotion (React 可编程) | 内置时间线编辑器 |
| AI 编排 | Mastra Agent 框架 | 自建工作流引擎 (run-runtime) | LangChain/LangGraph DAG |
| 对象存储 | 本地文件系统 | MinIO | RustFS (S3 兼容) |
| 部署复杂度 | 极低 (`npm run dev` 即可) | 高 (MySQL + Redis + MinIO + Worker 进程) | 中等 (Docker Compose 一键启动) |
| 开源协议 | 未注明 | 无开源协议 | Apache-2.0 |

### 流水线对比

```
火宝短剧                     waoowaoo                       Jellyfish
────────────────────         ────────────────────           ────────────────────
1. 剧本改写 (单步 Agent)     1. 剧本分析                     1. 剧本精简 Agent
   script_rewriter              story-to-script 工作流          script_simplifier
                                角色/场景/道具提取
                                → 分集 → 编剧转换            2. 剧本分段 Agent
                                                                script_divider
2. 角色/场景提取              2. 素材准备
   (单步 Agent)                  角色形象生成                 3. 元素提取 Agent
   extractor + 去重              场景图 / 配音分配               element_extractor
                                                                ├→ 角色画像分析
3. 素材生成 (并行)            3. 分镜生成 (4 阶段)              ├→ 场景分析
   角色图 + 场景图               Phase 1: 基础规划               ├→ 服装分析
   + 配音分配                    Phase 2a: 摄影 ─┐ 并行         └→ 道具分析
                                 Phase 2b: 表演 ─┘
4. 分镜拆解 (单步)               Phase 3: 细节                4. 实体合并 + 一致性检查
   storyboard_breaker                                            entity_merger
   17 字段/镜头               4. 媒体生成 (BullMQ)              consistency_checker
                                 图片(50)/视频(50)
5. 媒体生成                      语音(20)/文本(50)            5. 分镜提示词 Agent
   参考图 → I2V + TTS                                            shot_frame_prompt
                              5. 视频编辑 (Remotion)
6. FFmpeg 合成                   时间轴→转场→导出             6. AI 媒体生成
7. FFmpeg 拼接 → 成片            ↓ 含口型同步                 7. 时间线剪辑 → 导出
```

### 各自的设计优势

#### 火宝：轻量务实，用户体验优先

**1. 上手门槛最低** — SQLite + 无 Redis + 无 MinIO，零外部依赖，`npm run dev` 即可运行。

**2. Skill-as-prompt** — 把每个 Agent 的行为规范写成 SKILL.md，既是文档也是运行时 prompt，前端 UI 可直接编辑。非技术用户可调整 Agent 行为而不碰代码。

**3. 多供应商适配器干净** — 4 个接口（Text/Image/Video/Audio）+ Map 注册表，新增供应商实现接口即可。

**4. 前后端分离架构** — Hono 后端 + Nuxt 前端，职责清晰，未来可独立部署和扩展。

#### waoowaoo：工程深度，生产可靠性优先

**1. 任务可靠性最强** — 心跳 + 看门狗 + 租约三重保障，`withTaskLifecycle` 统一封装心跳、状态转换、计费结算/回滚。

**2. 计费系统最成熟** — 冻结-结算-退款模式（幂等键 + 乐观并发 + Decimal 精度），在开源项目中罕见。

**3. 分镜生成最成熟** — 4 阶段流水线（规划→摄影→表演→细节），每步 prompt 更聚焦，中间结果可审核。

**4. 功能链路最完整** — 口型同步（3 供应商）、声音克隆、情感控制、Remotion 可编程视频编辑。

#### Jellyfish：Agent 深度，语义理解优先

**1. Agent 编排最精细** — 12 个 LangGraph Agent 组成 DAG，每个职责单一。独立的一致性检查 Agent 和实体合并 Agent 在另外两个项目中完全没有。

**2. 可视化工作流编辑** — React Flow 实现的类 Dify 节点式编辑器，Agent 工作流可在 UI 上自定义编排。火宝和 waoowaoo 都没有这个能力。

**3. 资产管理最精细** — 四类独立资产（角色/场景/道具/服装）+ 项目级/全局级双层资产库。火宝和 waoowaoo 的资产只有角色和场景两类。

**4. Python 生态优势** — 唯一用 Python 后端（FastAPI + LangGraph）的项目，对接 AI/ML 生态更直接，LangChain 社区的 Agent 模式可直接复用。

**5. 协议最明确** — Apache-2.0，可商用。waoowaoo 无开源协议，火宝未注明。

### 各自的设计劣势

#### 火宝

**1. 没有任务队列** — 视频生成可能几分钟，服务重启 = 任务丢失，无重试/无看门狗/无并发控制。

**2. 分镜单步生成** — 一次产出 17 个字段，对 LLM 压力大，出错只能整体重来。

**3. 缺口型同步** — 嘴型对不上是短剧中非常明显的观感问题。

**4. Agent 框架偏重** — Mastra 的有状态聊天模式对流水线批处理过度设计。

#### waoowaoo

**1. 核心业务代码质量差** — 基础设施（计费/队列/租约）精致，但核心的分镜生成代码重复严重、prompt 校验系统被绕过。

**2. 单体架构** — 前端/API/Worker 全塞在一个 Next.js 项目，长期扩展受限。

**3. 工作流引擎过度设计** — 2 个工作流撑不起通用 DAG 引擎的复杂度。

**4. 无开源协议** — 法律层面使用有风险。

#### Jellyfish

**1. 项目太新** — 2026-03-06 创建，核心流程未稳定，不适合直接生产使用。

**2. 语音合成缺失** — TTS、配音相关能力公开信息未提及，是端到端链路的明显缺口。

**3. 任务可靠性未知** — 没有看到类似 BullMQ/看门狗的机制，生产可靠性存疑。

**4. 功能完整度偏低** — 相比 waoowaoo 的 15+ AI 供应商和完整计费系统，Jellyfish 在媒体生成层和运营层还不成熟。

### 功能完整度对比

| 能力 | 火宝短剧 | waoowaoo | Jellyfish |
|------|---------|----------|-----------|
| 剧本生成/改写 | ✅ Agent 驱动 | ✅ 工作流驱动 | ❌ 需外部提供 |
| 角色提取 | ✅ 含去重 | ✅ 含多变体 | ✅ 独立 Agent + 画像分析 |
| 场景提取 | ✅ 含去重 | ✅ | ✅ 独立 Agent |
| 道具/服装提取 | ❌ | ❌ | ✅ 各有独立 Agent |
| 一致性检查 | ❌ | ❌ | ✅ 独立 Agent |
| 分镜生成 | ✅ 单步 17 字段 | ✅ 4 阶段流水线 | ✅ Agent 链 |
| 图片生成 | ✅ 8 供应商 | ✅ 7+ 供应商 | ✅ 多供应商 |
| Grid 参考图 | ✅ 3 模式 | ❌ | ❌ |
| 视频生成 (I2V) | ✅ 异步轮询 | ✅ BullMQ Worker | ✅ |
| TTS 语音 | ✅ | ✅ + 声音克隆 + 情感控制 | ? 未确认 |
| 口型同步 | ❌ | ✅ 3 供应商 | ✅ 提及 |
| 视频合成/剪辑 | ✅ FFmpeg | ✅ Remotion | ✅ 内置时间线编辑器 |
| 转场效果 | ❌ | ✅ 基础 | 未知 |
| 多集管理 | ✅ Drama→Episode | ✅ Project→Episode→Clip | ✅ 项目→章节 |
| 资产层级 | 项目级 | 项目级 + 全局级 | 项目级 + 全局级（四类资产） |
| 批量任务 | ⚠️ 无队列保障 | ✅ BullMQ 并行 | 未知 |
| 计费系统 | ❌ | ✅ 冻结-结算-退款 | ❌ |
| AI 供应商数 | 8 家 | 15+ 家 | 多家（具体数量未列出） |
| Prompt 可编辑 | ✅ 前端 UI 编辑 | ❌ 代码内嵌 | ✅ 提示词模板库 |
| 工作流可视化编辑 | ❌ | ❌ | ✅ React Flow 编辑器 |
| 任务可靠性 | ⚠️ 无重试/无看门狗 | ✅ 心跳+看门狗+租约 | 未知 |

### 架构质量对比

| 维度 | 火宝短剧 | waoowaoo | Jellyfish |
|------|---------|----------|-----------|
| 上手难度 | ★★★★★ 极低 | ★★☆☆☆ 偏高 | ★★★☆☆ Docker 中等 |
| 架构整洁度 | ★★★★☆ | ★★★☆☆ | ★★★★☆ |
| 生产可靠性 | ★★☆☆☆ | ★★★★☆ | ★★☆☆☆ 未知 |
| 功能完整度 | ★★★☆☆ | ★★★★★ | ★★★☆☆ |
| 可扩展性 | ★★★☆☆ | ★★★★☆ | ★★★★☆ |
| AI 编排深度 | ★★★☆☆ | ★★★☆☆ | ★★★★★ |
| Prompt 管理 | ★★★★☆ | ★★★☆☆ | ★★★★☆ |
| 资产管理深度 | ★★☆☆☆ | ★★★☆☆ | ★★★★★ |
| 代码质量一致性 | ★★★★☆ 较均匀 | ★★☆☆☆ 基础设施好业务差 | 未深入审计 |
| 项目成熟度 | ★★★☆☆ | ★★★★☆ | ★☆☆☆☆ 极早期 |

### 设计哲学总结

**火宝短剧** = **产品思维**。Skill 可编辑、一键预设、零依赖部署，用户体验打磨得好，但工程深度不够，缺少队列、重试、计费。**适合个人使用和概念验证。**

**waoowaoo** = **工程思维**。分布式执行、计费系统、看门狗都很成熟，但核心业务代码（分镜生成）反而最粗糙，代码重复严重，prompt 校验系统形同虚设。**适合准生产环境和多用户场景。**

**Jellyfish** = **Agent 思维**。12 个 LangGraph Agent 精细分工 + 可视化工作流编辑器 + 四类资产双层管理，Agent 编排深度远超前两者，但项目太新，功能完整度和可靠性尚未验证。**适合研究 Agent 架构设计和资产管理方案。**

三者恰好互补——火宝做用户体验，waoowaoo 做工程可靠性，Jellyfish 做 Agent 编排深度。

### 对自建系统的启发

#### 从火宝学到的

1. **Skill-as-prompt 模式** — 将 Agent 行为规范写成可编辑的 markdown 文件，前端可直接修改，实现 prompt 与代码的解耦
2. **适配器接口 + Map 注册表** — 比 switch-case 更干净的多供应商管理
3. **零依赖部署优先** — SQLite 优先，需要时再升级到 PostgreSQL/MySQL，降低上手门槛
4. **Episode 级配置锁定** — 创建剧集时冻结 AI 服务配置，保证同一集的生产一致性
5. **Grid 参考图的 3 种模式** — 首帧/首尾帧/多参考图，提供不同精度的角色一致性控制

#### 从 waoowaoo 学到的

1. **冻结-结算-退款计费模式** — 任务前冻结预估费用，完成后结算差额，失败回滚
2. **多阶段分镜流水线** — 规划→摄影→表演→细节，每阶段职责明确，比单步生成更可控
3. **心跳 + 看门狗 + 租约** — 长时间 AI 任务的可靠执行保障
4. **用户级并发控制** — 防止单用户垄断 Worker 资源
5. **MediaObject 内容寻址** — SHA256 + storageKey 统一管理媒体文件，支持去重和迁移

#### 从 Jellyfish 学到的

1. **精细的 Agent 分工** — 一致性检查、实体合并、变体分析作为独立 Agent，而非附属步骤。把"质量校验"从流程末端提升为一等公民
2. **可视化工作流编辑** — React Flow 让非技术用户也能调整 Agent 编排，降低 Agent 系统的使用门槛
3. **四类资产 + 双层管理** — 道具和服装作为独立资产类型管理（不是角色的附属属性），项目级 vs 全局级支持跨项目复用
4. **Python + LangGraph 的 Agent 编排** — 直接复用 LangChain 生态，比在 TypeScript 中自建 Agent 框架更高效
5. **Apache-2.0 协议** — 明确的开源协议对商用场景至关重要

#### 三者都应避免的

1. **外部框架过度依赖** — 火宝的 Mastra、waoowaoo 的通用 DAG 引擎、Jellyfish 的 LangGraph，对当前规模都可能偏重
2. **JSON-in-column** — 结构化数据存为 JSON blob 牺牲查询能力（火宝和 waoowaoo 都有此问题）
3. **视频编辑能力薄弱** — 都缺少专业剪辑功能（Ken Burns、图层、遮罩、文字特效）
4. **无 AI 配乐生成** — 都只支持手动 BGM 或提示词级描述，缺少根据情节自动生成配乐的能力
5. **语音合成集成不足** — Jellyfish 缺失 TTS，火宝 TTS 基础，只有 waoowaoo 有声音克隆和情感控制

---

## 视频拼接深度对比：从短片段到完整剧集

> 当前 AI 视频模型单次只能生成 5-15 秒片段，一集 1-2 分钟的剧需要 8-20+ 个片段拼接。这是规模化生产的核心瓶颈，三个开源工具采用了截然不同的方案。

### 三个开源工具的拼接实现

#### 火宝短剧：FFmpeg 两阶段拼接

流程分两步：
1. **Stage 6 — 单镜头合成**：`fluent-ffmpeg` 将每个镜头的视频 + TTS 音频 + 字幕合成为一个完整镜头（`/api/v1/compose`）
2. **Stage 7 — 全集拼接**：将所有合成镜头按顺序拼接成完整剧集（`/api/v1/merge`）

**转场**：无。所有镜头之间是硬切，没有 dissolve/fade/slide 等过渡效果。

**可自动化程度**：✅ 完全自动化，后端 API 驱动，无需人工介入。

**局限**：fluent-ffmpeg 能做基本的视频+音频+字幕合成，但无法做转场效果、图层合成、文字特效。扩展需要手写 FFmpeg `filter_complex`，维护成本高。

#### waoowaoo：Remotion 可编程视频

用 React 生态的 [Remotion](https://remotion.dev) 框架，视频合成逻辑用 React 组件描述：

**核心算法**（`utils/time-utils.ts` `computeClipPositions()`）：
- 片段按顺序排列为 Remotion `<Sequence>` 组件
- 有转场时，相邻片段在时间轴上重叠（重叠区 = 转场时长 / 2）
- 两个 Sequence 在 `<AbsoluteFill>` 中自然叠加，配合 CSS opacity/transform 动画实现转场

**转场效果**：3 种 — dissolve（交叉溶解）、fade（淡入淡出）、slide（滑动，支持四方向）。时长可选 0.3s/0.5s/1.0s/1.5s。

**创建流程**（`useEditorActions.ts` `createProjectFromPanels()`）：
- 每个 AI 生成的面板默认 3 秒 × 30fps = 90 帧
- 所有片段间默认 dissolve 转场，15 帧（0.5s）
- 配音和字幕自动附加到对应片段
- 默认 30fps / 1920×1080

**可自动化程度**：⚠️ 部分。浏览器内预览（`@remotion/player`）已实现，但**服务端渲染路由未实现** — `startRender()` 的 API endpoint 在客户端代码中定义了但服务端没有对应路由。`@remotion/cli` 在 `package.json` 中有声明但未接入。也就是说，目前只能在浏览器里预览，无法自动导出 MP4。

**局限**：Remotion 编辑器组件仅 237 行，没有错误边界、没有 memoization、不支持复杂合成（图层、遮罩、Ken Burns 等）。

#### Jellyfish：内置时间线编辑器

提供多轨视频/音频时间线编辑器，支持素材拖拽和导出。

**可自动化程度**：❌ 需要人工在时间线上拖拽排列，无法通过 API 自动触发拼接。

**局限**：项目极早期（2026-03-06），编辑/导出流程不稳定。

### 拼接方案对比表

| 维度 | 火宝短剧 | waoowaoo | Jellyfish |
|------|---------|----------|-----------|
| 技术方案 | FFmpeg 命令行 | Remotion (React 组件) | 内置时间线编辑器 |
| 转场效果 | ❌ 硬切 | ✅ 3 种（dissolve/fade/slide） | 未知 |
| 转场可配置 | N/A | ✅ 类型+时长 | 未知 |
| 音频处理 | ✅ FFmpeg 混音 | ✅ BGM 淡入淡出 + 配音分轨 | 多轨音频支持 |
| 字幕烧录 | ✅ FFmpeg ASS | ✅ React 组件渲染 | 未知 |
| 完全自动化 | ✅ API 驱动 | ⚠️ 预览可自动、导出未实现 | ❌ 需手动 |
| 服务端渲染 | ✅ FFmpeg 原生 | ❌ Lambda/CLI 未接入 | 未知 |
| 可扩展性 | 低（FFmpeg 命令拼接） | 高（React 组件可编程） | 中 |
| 性能 | ★★★★★ FFmpeg 原生速度 | ★★★☆☆ Headless 浏览器渲染 | 未知 |

### 商业工具的拼接方案参考

| 工具 | 方案 | 可自动化 | 备注 |
|------|------|---------|------|
| 火山剧创 | 导出到剪映/CapCut | ❌ 需人工 | 借助字节编辑器生态 |
| 小云雀 | Agent 自动组装 + 音画一体 | ✅ 全自动 | 具体实现未公开 |
| FreeToon | 腾讯云媒体处理 | ✅ 批量管线 | 依赖腾讯云基础设施 |
| 巨日禄 | 不做拼接，输出单镜头资产 | N/A | 用户自行在外部编辑器组装 |

### 业界自动化拼接方案

除了上述工具的内置方案外，业界还有以下可用于自动化管线的技术路线：

**1. FFmpeg `xfade` 滤镜** — 最成熟的自动化底座

FFmpeg 原生支持 [44 种转场效果](https://ottverse.com/crossfade-between-videos-ffmpeg-xfade-filter/)，可链式串联处理多片段。已有现成自动化脚本：[Python 版](https://gist.github.com/royshil/369e175960718b5a03e40f279b131788)、[Node.js `ffmpeg-transitions`](https://github.com/Rickkorsten/ffmpeg-transitions)、[xfade-easing 扩展](https://github.com/scriptituk/xfade-easing)（CSS 缓动 + GLSL 自定义转场）。注意所有输入需先归一化帧率和 timebase。

**2. [Remotion Lambda](https://www.remotion.dev/docs/lambda)** — 无服务器分布式渲染

将 React 组件描述的合成逻辑部署到 AWS Lambda，主 Lambda 分发 renderer Lambda 并行渲染各段，最终自动拼接上传 S3。可通过 [API Gateway + SQS](https://www.remotion.dev/docs/lambda/sqs) 构建全自动队列。waoowaoo 的 Remotion 方案如果接入 Lambda 即可解决服务端渲染问题。

**3. [MoviePy](https://github.com/Zulko/moviepy)** — Python 生态方案

底层 FFmpeg，提供 `concatenate_videoclips()` + `crossfadein()` 等 Python API。和 Python AI 管线天然集成，但比直接 FFmpeg 慢，大量片段有内存问题。

**4. [ViMax](https://github.com/HKUDS/ViMax) Assembly Agent** — Agent 驱动拼接

港大 HKUDS 实验室的多 Agent 视频框架（2.6k stars），其 Assembly Agent 自动决策拼接顺序、转场类型和时间节奏，底层用 MoviePy/FFmpeg。值得参考其 Agent 驱动的拼接决策逻辑。

**5. [Stable Video Infinity](https://github.com/vita-epfl/Stable-Video-Infinity)** — 模型原生长视频（ICLR 2026 Oral）

通过 Error-Recycling Fine-Tuning 让模型原生生成任意长度视频，支持多场景转场和 250 秒以上连续视频。目前仍是研究项目，但代表了长期方向——拼接问题可能最终被模型能力本身解决。

### 拼接层总结

**现状**：三个开源工具中，只有火宝短剧实现了完全自动化的拼接（但无转场），waoowaoo 有最好的转场效果但缺少服务端渲染，Jellyfish 需要手动操作。商业工具中，小云雀是唯一已知全自动拼接的，但实现闭源。

**差距**：所有开源工具的拼接层都不成熟——要么缺转场，要么缺自动化，要么两者都缺。这是自建系统的差异化机会。

**推荐路径**：MVP 用 FFmpeg `xfade` 命令生成器（性能最好、44 种转场、Go/Python 均可直接调用），进阶阶段引入 Assembly Agent 自动决策转场类型和节奏。
