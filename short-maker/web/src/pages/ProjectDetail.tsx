import { useEffect, useState, useRef, useCallback } from "react";
import { useParams } from "react-router-dom";
import {
  getProject,
  runPhase,
  subscribeToEvents,
  type ProjectDetail as ProjectDetailType,
  type SSEEvent,
} from "../api";
import PipelineProgress from "../components/PipelineProgress";
import BlueprintView from "../components/BlueprintView";
import StoryboardView from "../components/StoryboardView";
import ShotGallery from "../components/ShotGallery";

const PHASE_ORDER = [
  "story_understanding",
  "character_asset",
  "storyboard",
  "image_generation",
  "video_generation",
];

export default function ProjectDetail() {
  const { id } = useParams<{ id: string }>();
  const [detail, setDetail] = useState<ProjectDetailType | null>(null);
  const [loading, setLoading] = useState(true);
  const [completedPhases, setCompletedPhases] = useState<Set<string>>(
    new Set()
  );
  const [pipelineStatus, setPipelineStatus] = useState("unknown");
  const [currentPhase, setCurrentPhase] = useState("");
  const [nextPhase, setNextPhase] = useState("");
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState("");
  const esRef = useRef<EventSource | null>(null);

  const refreshProject = useCallback(async () => {
    if (!id) return;
    try {
      const d = await getProject(id);
      setDetail(d);
      setPipelineStatus(d.pipeline_status);
      setCurrentPhase(d.current_phase);
      setNextPhase(d.next_phase || "");

      if (d.pipeline_status === "completed") {
        setCompletedPhases(new Set(PHASE_ORDER));
      } else if (d.current_phase) {
        const idx = PHASE_ORDER.indexOf(d.current_phase);
        if (idx >= 0) {
          setCompletedPhases(new Set(PHASE_ORDER.slice(0, idx + 1)));
        }
      }

      if (d.pipeline_status === "running") {
        setIsRunning(true);
      } else {
        setIsRunning(false);
      }
    } catch (err: any) {
      console.error(err);
    }
  }, [id]);

  useEffect(() => {
    refreshProject().finally(() => setLoading(false));
  }, [refreshProject]);

  const handleRunPhase = async (phase: string, episode?: number) => {
    if (!id) return;
    setError("");
    setIsRunning(true);

    try {
      await runPhase(id, { phase, episode });

      // Subscribe to SSE for this run
      if (esRef.current) {
        esRef.current.close();
      }

      const es = subscribeToEvents(id, (event: SSEEvent) => {
        if (event.type === "phase_complete" && event.phase) {
          setCompletedPhases((prev) => new Set([...prev, event.phase!]));
          setCurrentPhase(event.phase);
          setIsRunning(false);
          es.close();
          esRef.current = null;
          refreshProject();
        }
        if (event.type === "done") {
          setPipelineStatus("completed");
          setCompletedPhases(new Set(PHASE_ORDER));
          setIsRunning(false);
          es.close();
          esRef.current = null;
          refreshProject();
        }
        if (event.type === "error") {
          setIsRunning(false);
          setPipelineStatus("paused");
          setError(event.message || "Pipeline 执行失败");
          es.close();
          esRef.current = null;
          refreshProject();
        }
      });
      esRef.current = es;
    } catch (err: any) {
      setIsRunning(false);
      setError(err.message || "启动失败");
      refreshProject();
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

  return (
    <div>
      <div className="flex items-center gap-3 mb-6">
        <h2 className="text-xl font-bold">{detail.project.name}</h2>
        <span
          className={`text-xs px-2 py-0.5 rounded text-white ${statusColor[pipelineStatus] || "bg-gray-500"}`}
        >
          {statusLabel[pipelineStatus] || pipelineStatus}
        </span>
        <span className="text-sm text-gray-500">
          {detail.project.style} · {detail.project.episode_count} 集
        </span>
      </div>

      <PipelineProgress
        currentPhase={currentPhase}
        status={pipelineStatus}
        completedPhases={completedPhases}
        nextPhase={nextPhase}
        episodeCount={detail.project.episode_count}
        onRunPhase={handleRunPhase}
        isRunning={isRunning}
      />

      {error && (
        <div className="text-red-400 text-sm mb-4 bg-red-900/20 border border-red-800 rounded-lg p-3">
          {error}
        </div>
      )}

      {detail.blueprint && (
        <div className="mb-6">
          <h3 className="text-lg font-bold mb-3">剧本蓝图</h3>
          <BlueprintView blueprint={detail.blueprint} />
        </div>
      )}

      {detail.storyboard && detail.storyboard.length > 0 && (
        <div className="mb-6">
          <h3 className="text-lg font-bold mb-3">分镜</h3>
          <StoryboardView storyboard={detail.storyboard} />
        </div>
      )}

      {detail.images && detail.images.length > 0 && (
        <div className="mb-6">
          <h3 className="text-lg font-bold mb-3">图片/视频</h3>
          <ShotGallery
            images={detail.images}
            videos={detail.videos || []}
          />
        </div>
      )}

      {pipelineStatus === "paused" &&
        !detail.blueprint &&
        (!detail.storyboard || detail.storyboard.length === 0) &&
        (!detail.images || detail.images.length === 0) && (
          <div className="text-center py-12 text-gray-500">
            点击上方"运行"按钮开始第一个阶段
          </div>
        )}

      {pipelineStatus === "completed" &&
        (!detail.images || detail.images.length === 0) && (
          <div className="text-center py-12 text-gray-500">
            Pipeline 已完成
          </div>
        )}
    </div>
  );
}
