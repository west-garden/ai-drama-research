package agent

import (
	"context"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
)

const sampleStoryResponse = `{
	"world_view": "西游记神话世界，充满妖魔鬼怪和仙法奇术",
	"characters": [
		{
			"name": "孙悟空",
			"description": "齐天大圣，被压五行山五百年后被唐僧解救",
			"traits": ["好斗", "忠诚", "机智"]
		},
		{
			"name": "唐僧",
			"description": "取经人，慈悲为怀的高僧",
			"traits": ["慈悲", "坚定", "善良"]
		}
	],
	"episodes": [
		{
			"number": 1,
			"role": "hook",
			"emotion_arc": "从平静到惊奇",
			"synopsis": "唐僧途经五行山，解救孙悟空",
			"scenes": [
				{
					"narrative_beat": "开场引入",
					"emotion_arc": "平静→好奇",
					"setting": "五行山脚下",
					"pacing": "slow",
					"character_count": 1
				},
				{
					"narrative_beat": "关键相遇",
					"emotion_arc": "惊奇→激动",
					"setting": "五行山顶",
					"pacing": "fast",
					"character_count": 2
				}
			]
		},
		{
			"number": 2,
			"role": "hook",
			"emotion_arc": "从混乱到和解",
			"synopsis": "孙悟空获自由后大闹，唐僧用紧箍咒制服",
			"scenes": [
				{
					"narrative_beat": "冲突爆发",
					"emotion_arc": "混乱→愤怒",
					"setting": "山林间",
					"pacing": "fast",
					"character_count": 2
				}
			]
		}
	],
	"relationships": [
		{
			"character_a": "孙悟空",
			"character_b": "唐僧",
			"type": "师徒"
		}
	]
}`

func TestStoryAgent_Run(t *testing.T) {
	mockLLM := llm.NewMockClient()
	mockLLM.SetDefaultResponse(sampleStoryResponse)

	agent := NewStoryAgent(mockLLM, "test-model")
	project := domain.NewProject("西游记测试", domain.StyleManga, 2)
	state := NewPipelineState(project, "第一集：初遇\n孙悟空从五行山下被唐僧解救。\n第二集：收服\n唐僧用紧箍咒制服孙悟空。")

	result, err := agent.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("StoryAgent.Run: %v", err)
	}

	bp := result.Blueprint
	if bp == nil {
		t.Fatal("expected Blueprint to be set")
	}
	if bp.ProjectID != project.ID {
		t.Errorf("expected projectID '%s', got '%s'", project.ID, bp.ProjectID)
	}
	if bp.WorldView != "西游记神话世界，充满妖魔鬼怪和仙法奇术" {
		t.Errorf("unexpected world_view: %s", bp.WorldView)
	}
	if len(bp.Characters) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(bp.Characters))
	}
	if bp.Characters[0].Name != "孙悟空" {
		t.Errorf("expected first character '孙悟空', got '%s'", bp.Characters[0].Name)
	}
	if bp.Characters[0].ID == "" {
		t.Error("expected character to have an ID")
	}
	if len(bp.Episodes) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(bp.Episodes))
	}
	if bp.Episodes[0].Role != domain.EpisodeRoleHook {
		t.Errorf("expected episode 1 role 'hook', got '%s'", bp.Episodes[0].Role)
	}
	if len(bp.Episodes[0].Scenes) != 2 {
		t.Errorf("expected 2 scenes in episode 1, got %d", len(bp.Episodes[0].Scenes))
	}
	if len(bp.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(bp.Relationships))
	}
}

func TestStoryAgent_Phase(t *testing.T) {
	agent := NewStoryAgent(llm.NewMockClient(), "model")
	if agent.Phase() != PhaseStoryUnderstanding {
		t.Errorf("expected phase story_understanding, got %s", agent.Phase())
	}
}

func TestStoryAgent_LLMCalledWithCorrectPrompt(t *testing.T) {
	mockLLM := llm.NewMockClient()
	mockLLM.SetDefaultResponse(sampleStoryResponse)

	agent := NewStoryAgent(mockLLM, "gpt-4o")
	project := domain.NewProject("测试剧", domain.StyleManga, 5)
	state := NewPipelineState(project, "测试剧本内容")

	agent.Run(context.Background(), state)

	calls := mockLLM.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 LLM call, got %d", len(calls))
	}
	if calls[0].Model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got '%s'", calls[0].Model)
	}
	if len(calls[0].Messages) < 2 {
		t.Fatal("expected at least 2 messages (system + user)")
	}
	if calls[0].Messages[0].Role != "system" {
		t.Errorf("expected first message role 'system', got '%s'", calls[0].Messages[0].Role)
	}
}
