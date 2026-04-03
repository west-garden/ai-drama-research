# AI 短剧/漫剧工具调研

调研目标：了解市面上 AI 短剧/漫剧工具的能力边界，为后续构建 Agent 驱动的自进化 AI 漫剧/短剧系统提供参考。

## 目录结构

```
.
├── tools/                  # 单个工具的详细调研
│   └── _template.md        # 调研模板
├── capabilities/           # 按能力维度的横向对比
│   ├── image-generation.md     # 图片/漫画生成
│   ├── video-generation.md     # 视频生成
│   ├── voice-and-audio.md      # 语音合成/克隆/配音
│   ├── lip-sync.md             # 口型同步
│   ├── story-writing.md        # 剧本/故事生成
│   └── end-to-end.md           # 端到端流水线
├── comparisons/            # 对比分析
│   └── overview.md             # 工具全景对比表
├── insights/               # 调研洞察与思考
│   ├── patterns.md             # 共性模式总结
│   ├── gaps.md                 # 市场空白与机会
│   └── architecture-ideas.md   # 对自建系统的架构启发
└── README.md
```

## 调研维度

每个工具从以下维度评估：

1. **核心能力** — 它解决什么问题
2. **工作流** — 从输入到输出的完整流程
3. **AI 能力边界** — 能做到什么程度，哪里需要人工介入
4. **输出质量** — 实际产出水平
5. **自动化程度** — 多少环节可以自动完成
6. **可编程性** — 是否有 API、是否可集成到自动化流水线
7. **定价** — 成本结构
8. **局限性** — 做不到什么
