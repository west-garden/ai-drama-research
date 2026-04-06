import { useEffect, useState, useRef, useCallback } from "react";
import { useParams } from "react-router-dom";
import {
  getProject,
  getWorkflow,
  runPhase,
  subscribeToEvents,
  type ProjectDetail as ProjectDetailType,
  type WorkflowResponse,
  type SSEEvent,
} from "../api";
import WorkflowGraph from "../components/WorkflowGraph";
import BlueprintView from "../components/BlueprintView";
import CharacterAssetView from "../components/CharacterAssetView";
import StoryboardView from "../components/StoryboardView";
import ShotGallery from "../components/ShotGallery";

export default function ProjectDetail() {
  const { id } = useParams<{ id: string }>();
  const [detail, setDetail] = useState<ProjectDetailType | null>(null);
  const [workflow, setWorkflow] = useState<WorkflowResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState("");
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const esRef = useRef<EventSource | null>(null);

  const refreshData = useCallback(async () => {
    if (!id) return;
    try {
      const d = await getProject(id);
      setDetail(d);
      setIsRunning(d.pipeline_status === "running");
    } catch (err: any) {
      console.error("Failed to load project:", err);
      return;
    }
    try {
      const w = await getWorkflow(id);
      setWorkflow(w);
    } catch (err: any) {
      console.error("Failed to load workflow:", err);
    }
  }, [id]);

  useEffect(() => {
    refreshData().finally(() => setLoading(false));
  }, [refreshData]);

  const handleRunNode = async (nodeId: string) => {
    if (!id) return;
    setError("");
    setIsRunning(true);

    try {
      await runPhase(id, { phase: nodeId });

      // Subscribe to SSE for this run
      if (esRef.current) {
        esRef.current.close();
      }

      const es = subscribeToEvents(id, (event: SSEEvent) => {
        if (event.type === "phase_complete") {
          setIsRunning(false);
          es.close();
          esRef.current = null;
          refreshData();
        }
        if (event.type === "phase_start") {
          // Refresh workflow to show running state
          if (id) getWorkflow(id).then(setWorkflow);
        }
        if (event.type === "error") {
          setIsRunning(false);
          setError(event.message || "Pipeline 执行失败");
          es.close();
          esRef.current = null;
          refreshData();
        }
      });
      esRef.current = es;
    } catch (err: any) {
      setIsRunning(false);
      setError(err.message || "启动失败");
      refreshData();
    }
  };

  useEffect(() => {
    return () => {
      if (esRef.current) {
        esRef.current.close();
      }
    };
  }, []);

  if (loading) {
    return <div className="text-gray-500">加载中...</div>;
  }

  if (!detail) {
    return <div className="text-red-400">项目未找到</div>;
  }

  const statusLabel: Record<string, string> = {
    running: "运行中",
    completed: "已完成",
    failed: "失败",
    paused: "等待操作",
    unknown: "未知",
  };

  const statusColor: Record<string, string> = {
    running: "bg-blue-500",
    completed: "bg-green-500",
    failed: "bg-red-500",
    paused: "bg-yellow-500",
    unknown: "bg-gray-500",
  };

  const pipelineStatus = detail.pipeline_status;

  // Determine which detail panel to show based on selected node
  const selectedPhase = selectedNode
    ? selectedNode.split(":")[0]
    : null;
  const selectedEpisode = selectedNode?.includes(":ep")
    ? parseInt(selectedNode.split(":ep")[1])
    : null;

  // Filter data for selected episode
  const filteredStoryboard =
    selectedEpisode && detail.storyboard
      ? detail.storyboard.filter((s) => s.episode_number === selectedEpisode)
      : detail.storyboard;

  const filteredImages =
    selectedEpisode && detail.images
      ? detail.images.filter((s) => s.episode_number === selectedEpisode)
      : detail.images;

  const filteredVideos =
    selectedEpisode && detail.videos
      ? detail.videos.filter((s) => s.episode_number === selectedEpisode)
      : detail.videos;

  return (
    <div>
      {/* Header */}
      <div className="flex items-center gap-3 mb-4">
        <h2 className="text-xl font-bold">{detail.project.name}</h2>
        <span
          className={`text-xs px-2 py-0.5 rounded text-white ${statusColor[pipelineStatus] || "bg-gray-500"}`}
        >
          {statusLabel[pipelineStatus] || pipelineStatus}
        </span>
        <span className="text-sm text-gray-500">
          {detail.project.style} · {detail.project.episode_count} 集 ·{" "}
          {detail.project.prompt_language === "zh" ? "中文提示词" : "English prompts"}
        </span>
      </div>

      {/* Error banner */}
      {error && (
        <div className="text-red-400 text-sm mb-4 bg-red-900/20 border border-red-800 rounded-lg p-3">
          {error}
        </div>
      )}

      {/* Workflow Graph — full-bleed, fills viewport */}
      {workflow && (
        <div
          style={{
            height: "calc(100vh - 120px)",
            width: "100vw",
            marginLeft: "calc(-50vw + 50%)",
          }}
        >
          <WorkflowGraph
            workflowNodes={workflow.nodes}
            workflowEdges={workflow.edges}
            projectStyle={detail.project.style}
            episodeCount={detail.project.episode_count}
            promptLanguage={detail.project.prompt_language || "en"}
            selectedNode={selectedNode}
            onRunNode={handleRunNode}
            onSelectNode={setSelectedNode}
          />
        </div>
      )}

      {/* Selected node info bar */}
      {selectedNode && (
        <div className="flex items-center gap-3 mb-4 bg-gray-800 rounded-lg px-4 py-2">
          <span className="text-sm text-gray-400">选中节点:</span>
          <span className="text-sm text-white font-medium">{selectedNode}</span>
          <button
            onClick={() => setSelectedNode(null)}
            className="text-xs text-gray-500 hover:text-gray-300 ml-auto"
          >
            清除选择
          </button>
        </div>
      )}

      {/* Detail Panel */}
      {(selectedPhase === "story_understanding" || !selectedNode) &&
        detail.blueprint && (
          <div className="mb-6">
            <h3 className="text-lg font-bold mb-3">剧本蓝图</h3>
            <BlueprintView blueprint={detail.blueprint} />
          </div>
        )}

      {(selectedPhase === "character_asset" || !selectedNode) &&
        detail.assets &&
        detail.assets.length > 0 && (
          <div className="mb-6">
            <h3 className="text-lg font-bold mb-3">角色资产</h3>
            <CharacterAssetView assets={detail.assets} />
          </div>
        )}

      {(selectedPhase === "storyboard" || !selectedNode) &&
        filteredStoryboard &&
        filteredStoryboard.length > 0 && (
          <div className="mb-6">
            <h3 className="text-lg font-bold mb-3">
              分镜{selectedEpisode ? ` (EP${selectedEpisode})` : ""}
            </h3>
            <StoryboardView storyboard={filteredStoryboard} />
          </div>
        )}

      {(selectedPhase === "image_generation" ||
        selectedPhase === "video_generation" ||
        !selectedNode) &&
        filteredImages &&
        filteredImages.length > 0 && (
          <div className="mb-6">
            <h3 className="text-lg font-bold mb-3">
              图片/视频{selectedEpisode ? ` (EP${selectedEpisode})` : ""}
            </h3>
            <ShotGallery
              images={filteredImages}
              videos={filteredVideos || []}
            />
          </div>
        )}

      {/* Empty state */}
      {!detail.blueprint &&
        (!detail.storyboard || detail.storyboard.length === 0) &&
        (!detail.images || detail.images.length === 0) && (
          <div className="text-center py-12 text-gray-500">
            点击工作流图中的节点开始运行
          </div>
        )}
    </div>
  );
}
