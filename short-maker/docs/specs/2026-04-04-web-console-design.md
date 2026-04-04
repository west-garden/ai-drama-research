# Web 控制台设计

> 项目代号：Short Maker
> 日期：2026-04-04
> 状态：设计评审
> 依赖：Plan 1-4（完整 5 阶段 pipeline + mock 生成）

## 目标

为 Short Maker pipeline 提供 Web 控制台，支持上传剧本、启动 pipeline、实时查看进度、浏览生成的图片/视频产物。内部工具，功能实用优先。

## 决策摘要

| 决策项 | 选择 | 理由 |
|-------|------|------|
| 用户定位 | 自用/团队内部 | 不需要精致 UI，功能实用优先 |
| MVP 范围 | 跑通 + 查看 | 不做审核/编辑流程 |
| 前端技术 | React + Vite + TailwindCSS | 灵活、生态好，内部工具不需要组件库 |
| Go HTTP 框架 | net/http + chi | 轻量路由，和现有项目风格一致（依赖少） |
| 进度推送 | SSE (Server-Sent Events) | 单向推送，浏览器原生支持，实现简单，完全匹配"看进度"场景 |
| 状态存储 | SQLite 持久化 | 开发测试频繁重启，内存 map 会丢数据 |

## 架构

### 系统分层

```
React (Vite)          Go API (chi)           Pipeline (existing)
┌─────────────┐      ┌──────────────┐       ┌──────────────────┐
│ 上传剧本     │─POST→│ /api/projects│──────→│ Orchestrator.Run │
│ 看进度       │←SSE──│ /api/.../sse │←hook──│ (checkpoint hook)│
│ 浏览产物     │─GET──│ /api/shots   │       │ 5 agents         │
│             │─GET──│ /output/...  │       │ (mock backends)  │
└─────────────┘      └──────────────┘       └──────────────────┘
```

### 依赖方向

```
cmd/shortmaker/main.go
    ↓
internal/api (Server, handlers)
    ↓            ↓
internal/agent   internal/store
    ↓
internal/router, quality, domain
```

`api` 包依赖 `agent`（调 Orchestrator）和 `store`（读写 SQLite）。不引入反向依赖。

## Go API 设计

### 新增包：`internal/api/`

#### Server 结构

```go
// internal/api/server.go
type Server struct {
    router    chi.Router
    agents    map[agent.Phase]agent.Agent
    store     store.Store
    outputDir string
    runs      map[string]*PipelineRun  // projectID → 活跃 run（内存中维护 SSE channel）
    mu        sync.RWMutex
}

type PipelineRun struct {
    ProjectID string
    Status    string         // "running", "completed", "failed"
    Phase     agent.Phase    // 当前正在执行的 phase
    Events    chan SSEEvent  // checkpoint hook 写入，SSE handler 读取
    Error     string
}

type SSEEvent struct {
    Type    string `json:"type"`    // "phase_start", "phase_complete", "done", "error"
    Phase   string `json:"phase,omitempty"`
    Message string `json:"message,omitempty"`
}
```

#### 路由表

| 方法 | 路径 | 功能 |
|------|------|------|
| `POST` | `/api/projects` | 创建项目（multipart form：剧本文件 + style + episodes） |
| `GET` | `/api/projects` | 列出所有项目（简要信息 + 状态） |
| `GET` | `/api/projects/{id}` | 项目详情（blueprint、storyboard、images、videos） |
| `GET` | `/api/projects/{id}/events` | SSE 进度推送 |
| `GET` | `/output/*` | 静态文件服务（生成的 PNG/MP4） |

#### POST /api/projects 流程

1. 解析 multipart form，拿到剧本内容、style、episodes
2. 创建 `domain.Project` 和 `agent.PipelineState`
3. 写入 SQLite（project + pipeline_run 记录）
4. 创建 `PipelineRun`（含 Events channel），启动后台 goroutine 跑 `orchestrator.Run()`
5. Orchestrator 的 checkpoint hook 每个 phase 完成时：
   - 往 Events channel 写 SSEEvent
   - 更新 SQLite 中的 pipeline_run 状态和 pipeline_result JSON 快照
6. 返回 `201 Created` + project JSON

#### GET /api/projects/{id}/events 流程

1. 查找对应的 PipelineRun
2. 设置 SSE 响应头（`Content-Type: text/event-stream`，`Cache-Control: no-cache`）
3. 从 Events channel 读取事件，写入响应流（`data: {...}\n\n`）
4. Pipeline 完成或客户端断开时关闭连接
5. 如果 pipeline 已完成（无活跃 run），立即发送历史状态 + done

### SSE 事件格式

```json
{"type": "phase_start", "phase": "story_understanding"}
{"type": "phase_complete", "phase": "story_understanding"}
{"type": "phase_start", "phase": "character_asset"}
{"type": "phase_complete", "phase": "character_asset"}
{"type": "done"}
```

错误情况：

```json
{"type": "error", "message": "LLM call failed: connection refused"}
```

### CLI 变更

新增 `serve` 子命令：

```
shortmaker serve [flags]

Flags:
  --port        int     HTTP 端口 (default 8080)
  --output      string  输出目录 (default "./output")
  --db          string  SQLite 数据库路径 (default "./shortmaker.db")
  --strategies  string  策略 JSON 文件路径
  --model       string  LLM 模型名 (default "gpt-4o-mini")
  --mock        bool    使用 mock agents (default true)
```

### 中间件

- CORS：开发时前端 Vite（端口 5173）和 Go API（端口 8080）跨域，需要 CORS 中间件允许 `localhost:5173`
- 生产环境 Go 直接 serve `web/dist/` 静态文件，不需要 CORS

## SQLite 存储扩展

### 新增接口

`PipelineRunStore` 加入现有 `Store` 组合接口（现有 `Store` = `ProjectStore + AssetStore + BlueprintStore`，变为 `+ PipelineRunStore`）。

```go
// internal/store/store.go — 新增
type PipelineRunStore interface {
    SavePipelineRun(run *PipelineRunRecord) error
    UpdatePipelineRun(projectID string, status string, phase string, errMsg string) error
    GetPipelineRun(projectID string) (*PipelineRunRecord, error)
    SavePipelineResult(projectID string, resultJSON []byte) error
    GetPipelineResult(projectID string) ([]byte, error)
}

type PipelineRunRecord struct {
    ProjectID    string    `json:"project_id"`
    Status       string    `json:"status"`       // running, completed, failed
    CurrentPhase string    `json:"current_phase"`
    Error        string    `json:"error"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

### 新增表

```sql
CREATE TABLE IF NOT EXISTS pipeline_runs (
    project_id   TEXT PRIMARY KEY,
    status       TEXT NOT NULL DEFAULT 'running',
    current_phase TEXT NOT NULL DEFAULT '',
    error        TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pipeline_results (
    project_id   TEXT PRIMARY KEY,
    result_json  TEXT NOT NULL,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### 重启恢复策略

服务重启时：
1. 从 SQLite 读取所有 pipeline_runs
2. 状态为 `completed` 或 `failed` 的直接展示
3. 状态为 `running` 的标记为 `failed`（error = "server restarted"）
4. 生成的文件在 output/ 目录中不受影响

## React 前端设计

### 项目结构

```
web/
├── package.json
├── vite.config.ts
├── tsconfig.json
├── index.html
├── tailwind.config.js
├── postcss.config.js
└── src/
    ├── main.tsx
    ├── App.tsx               # React Router 路由配置
    ├── api.ts                # fetch 封装 + EventSource hook
    ├── pages/
    │   ├── ProjectList.tsx   # 项目列表页
    │   ├── NewProject.tsx    # 新建项目页
    │   └── ProjectDetail.tsx # 项目详情页
    └── components/
        ├── Layout.tsx            # 页面 shell（顶栏 + 容器）
        ├── PipelineProgress.tsx  # 5 阶段进度条
        └── ShotGallery.tsx       # 图片/视频网格
```

### 3 个页面

#### 项目列表 `/`

- 卡片网格，每个卡片显示：项目名、风格、状态标签（running 蓝色 / completed 绿色 / failed 红色）、创建时间
- 右上角"新建项目"按钮
- 点击卡片跳转详情页

#### 新建项目 `/new`

- 表单：拖拽/点击上传剧本文件（.txt）、风格下拉（manga/3d/live_action）、集数输入
- 提交后 POST 到 `/api/projects`，跳转到详情页

#### 项目详情 `/projects/{id}`

- **顶部**：项目名 + 状态标签 + 风格 + 创建时间
- **进度条**：5 个阶段横排（剧本理解、角色资产、分镜、图片生成、视频生成），完成的绿色带勾、进行中蓝色有动画、未开始灰色
- **产物画廊**：按集分组，每集一行，每个 shot 显示：
  - 缩略图（`<img src="/output/...">`)
  - Grade 色标（S 金色、A 蓝色、B 绿色、C 灰色）
  - 质量分数
  - 点击弹出大图，可切换图片/视频

### API 交互

```typescript
// api.ts
const API_BASE = "/api";

export async function createProject(form: FormData) {
  const res = await fetch(`${API_BASE}/projects`, { method: "POST", body: form });
  return res.json();
}

export async function listProjects() {
  const res = await fetch(`${API_BASE}/projects`);
  return res.json();
}

export async function getProject(id: string) {
  const res = await fetch(`${API_BASE}/projects/${id}`);
  return res.json();
}

export function subscribeToEvents(id: string, onEvent: (e: SSEEvent) => void) {
  const es = new EventSource(`${API_BASE}/projects/${id}/events`);
  es.onmessage = (e) => onEvent(JSON.parse(e.data));
  return es;  // 调用方负责 es.close()
}
```

### 开发代理

`vite.config.ts` 配 proxy：

```typescript
export default defineConfig({
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/output": "http://localhost:8080",
    }
  }
});
```

### 依赖

- `react`, `react-dom`, `react-router-dom` — 核心
- `tailwindcss`, `postcss`, `autoprefixer` — 样式
- `typescript`, `@types/react`, `@types/react-dom` — 类型

不引入状态管理库（Redux 等），3 个页面用 `useState` + `useEffect` 足够。

## API 响应结构

### GET /api/projects

```json
[
  {
    "id": "proj_abc123",
    "name": "西游记",
    "style": "manga",
    "episode_count": 10,
    "status": "completed",
    "current_phase": "video_generation",
    "created_at": "2026-04-04T15:30:00Z"
  }
]
```

### GET /api/projects/{id}

```json
{
  "project": {
    "id": "proj_abc123",
    "name": "西游记",
    "style": "manga",
    "status": "completed"
  },
  "pipeline_status": "completed",
  "current_phase": "video_generation",
  "blueprint": {
    "world_view": "古代仙侠世界",
    "characters": [{"name": "孙悟空", "description": "齐天大圣"}],
    "episodes": [{"number": 1, "role": "hook"}]
  },
  "storyboard": [
    {"shot_number": 1, "episode_number": 1, "frame_type": "wide", "prompt": "五行山全景"}
  ],
  "images": [
    {"shot_number": 1, "episode_number": 1, "image_path": "/output/proj_abc123/ep01/shot001.png", "grade": "S", "image_score": 90}
  ],
  "videos": [
    {"shot_number": 1, "episode_number": 1, "video_path": "/output/proj_abc123/ep01/shot001.mp4", "grade": "S", "video_score": 90}
  ]
}
```

图片/视频路径使用 `/output/` 前缀，前端直接用 `<img>` 和 `<video>` 标签加载。

## 新增 Go 依赖

- `github.com/go-chi/chi/v5` — 路由
- `github.com/go-chi/cors` — CORS 中间件

## 测试策略

### Go API 测试

| 测试 | 验证内容 |
|------|---------|
| TestCreateProject | POST 创建项目，返回 201 + project ID |
| TestCreateProject_InvalidForm | 缺少剧本文件时返回 400 |
| TestListProjects | 返回所有项目的简要列表 |
| TestGetProject | 返回完整 PipelineState JSON |
| TestGetProject_NotFound | 不存在的 ID 返回 404 |
| TestSSE_ReceivesEvents | 创建项目后 SSE 连接收到 phase 事件 |
| TestPipelineRunPersistence | 重启后能读到历史 pipeline 结果 |

用 `net/http/httptest` 做测试，mock agents 跑 pipeline。

### SQLite store 扩展测试

| 测试 | 验证内容 |
|------|---------|
| TestSavePipelineRun | 写入 run 记录 |
| TestUpdatePipelinePhase | 更新当前 phase |
| TestSavePipelineResult | 保存 PipelineState JSON 快照 |
| TestGetPipelineResult | 读取并反序列化结果 |

### React 端

MVP 不写自动化测试，手动验证。

### 端到端验证

手动用 `shortmaker serve` 启动，浏览器打开，上传 `testdata/sample-script.txt`，观察 pipeline 跑完并看到产物。

## 文件清单

| 操作 | 文件 |
|-----|------|
| 创建 | `internal/api/server.go` |
| 创建 | `internal/api/server_test.go` |
| 创建 | `internal/api/handler_project.go` |
| 创建 | `internal/api/handler_sse.go` |
| 创建 | `internal/api/middleware.go` |
| 修改 | `internal/store/store.go` — 新增 PipelineRunStore 接口 |
| 修改 | `internal/store/sqlite.go` — 新增表和方法 |
| 创建 | `internal/store/pipeline_test.go` |
| 修改 | `cmd/shortmaker/main.go` — 新增 serve 子命令 |
| 创建 | `web/package.json` |
| 创建 | `web/vite.config.ts` |
| 创建 | `web/tsconfig.json` |
| 创建 | `web/tailwind.config.js` |
| 创建 | `web/postcss.config.js` |
| 创建 | `web/index.html` |
| 创建 | `web/src/main.tsx` |
| 创建 | `web/src/App.tsx` |
| 创建 | `web/src/api.ts` |
| 创建 | `web/src/pages/ProjectList.tsx` |
| 创建 | `web/src/pages/NewProject.tsx` |
| 创建 | `web/src/pages/ProjectDetail.tsx` |
| 创建 | `web/src/components/Layout.tsx` |
| 创建 | `web/src/components/PipelineProgress.tsx` |
| 创建 | `web/src/components/ShotGallery.tsx` |
