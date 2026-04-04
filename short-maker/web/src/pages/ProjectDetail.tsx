import { useEffect, useState, useRef } from "react";
import { useParams } from "react-router-dom";
import { getProject, subscribeToEvents, type ProjectDetail as ProjectDetailType, type SSEEvent } from "../api";
import PipelineProgress from "../components/PipelineProgress";
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
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!id) return;

    getProject(id)
      .then((d) => {
        setDetail(d);
        setPipelineStatus(d.pipeline_status);
        setCurrentPhase(d.current_phase);

        if (d.pipeline_status === "completed") {
          setCompletedPhases(new Set(PHASE_ORDER));
        } else if (d.current_phase) {
          const idx = PHASE_ORDER.indexOf(d.current_phase);
          if (idx >= 0) {
            setCompletedPhases(new Set(PHASE_ORDER.slice(0, idx + 1)));
          }
        }
      })
      .catch(console.error)
      .finally(() => setLoading(false));

    const es = subscribeToEvents(id, (event: SSEEvent) => {
      if (event.type === "phase_complete" && event.phase) {
        setCompletedPhases((prev) => new Set([...prev, event.phase!]));
        setCurrentPhase(event.phase);
      }
      if (event.type === "done") {
        setPipelineStatus("completed");
        setCompletedPhases(new Set(PHASE_ORDER));
        getProject(id).then(setDetail);
        es.close();
      }
      if (event.type === "error") {
        setPipelineStatus("failed");
        es.close();
      }
    });
    esRef.current = es;

    return () => {
      es.close();
    };
  }, [id]);

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
    unknown: "未知",
  };

  const statusColor: Record<string, string> = {
    running: "bg-blue-500",
    completed: "bg-green-500",
    failed: "bg-red-500",
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
      />

      {detail.images && detail.images.length > 0 && (
        <ShotGallery
          images={detail.images}
          videos={detail.videos || []}
        />
      )}

      {pipelineStatus === "running" && (!detail.images || detail.images.length === 0) && (
        <div className="text-center py-12 text-gray-500">
          Pipeline 运行中，请等待...
        </div>
      )}

      {pipelineStatus === "failed" && (
        <div className="text-center py-12 text-red-400">
          Pipeline 执行失败
        </div>
      )}
    </div>
  );
}
