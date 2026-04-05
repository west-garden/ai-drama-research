import { useState } from "react";

const PHASES = [
  { key: "story_understanding", label: "剧本理解" },
  { key: "character_asset", label: "角色资产" },
  { key: "storyboard", label: "分镜" },
  { key: "image_generation", label: "图片生成" },
  { key: "video_generation", label: "视频生成" },
];

interface Props {
  currentPhase: string;
  status: string;
  completedPhases: Set<string>;
  nextPhase: string;
  episodeCount: number;
  onRunPhase: (phase: string, episode?: number) => void;
  isRunning: boolean;
}

export default function PipelineProgress({
  currentPhase,
  status,
  completedPhases,
  nextPhase,
  episodeCount,
  onRunPhase,
  isRunning,
}: Props) {
  const [selectedEpisode, setSelectedEpisode] = useState(0); // 0 = all

  const needsEpisodeSelector =
    nextPhase === "image_generation" || nextPhase === "video_generation";

  const phaseLabel =
    PHASES.find((p) => p.key === nextPhase)?.label || nextPhase;

  return (
    <div className="mb-6">
      <div className="flex gap-1 mb-4">
        {PHASES.map((phase) => {
          const isCompleted = completedPhases.has(phase.key);
          const isCurrent = phase.key === currentPhase && status === "running";
          const isFailed = phase.key === currentPhase && status === "failed";
          const isNext = phase.key === nextPhase && !isRunning;

          let bg = "bg-gray-800 border-gray-700";
          let icon = "○";
          let iconColor = "text-gray-600";

          if (isCompleted) {
            bg = "bg-green-900/30 border-green-700";
            icon = "✓";
            iconColor = "text-green-400";
          } else if (isCurrent) {
            bg = "bg-blue-900/30 border-blue-600";
            icon = "●";
            iconColor = "text-blue-400 animate-pulse";
          } else if (isFailed) {
            bg = "bg-red-900/30 border-red-700";
            icon = "✗";
            iconColor = "text-red-400";
          } else if (isNext) {
            bg = "bg-yellow-900/30 border-yellow-600";
            icon = "▶";
            iconColor = "text-yellow-400";
          }

          return (
            <div
              key={phase.key}
              className={`flex-1 text-center p-2 border rounded-lg ${bg}`}
            >
              <div className={`text-sm ${iconColor}`}>{icon}</div>
              <div className="text-xs text-gray-400 mt-1">{phase.label}</div>
            </div>
          );
        })}
      </div>

      {nextPhase && !isRunning && (
        <div className="flex items-center gap-3">
          {needsEpisodeSelector && (
            <select
              value={selectedEpisode}
              onChange={(e) => setSelectedEpisode(Number(e.target.value))}
              className="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white"
            >
              <option value={0}>全部集数</option>
              {Array.from({ length: episodeCount }, (_, i) => (
                <option key={i + 1} value={i + 1}>
                  第 {i + 1} 集
                </option>
              ))}
            </select>
          )}
          <button
            onClick={() =>
              onRunPhase(
                nextPhase,
                needsEpisodeSelector && selectedEpisode > 0
                  ? selectedEpisode
                  : undefined
              )
            }
            className="bg-yellow-600 hover:bg-yellow-700 text-white px-4 py-2 rounded-lg text-sm font-medium"
          >
            运行: {phaseLabel}
          </button>
        </div>
      )}

      {isRunning && (
        <div className="text-sm text-blue-400 animate-pulse">
          正在运行: {phaseLabel}...
        </div>
      )}
    </div>
  );
}
