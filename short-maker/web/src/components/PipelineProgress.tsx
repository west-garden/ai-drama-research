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
}

export default function PipelineProgress({
  currentPhase,
  status,
  completedPhases,
}: Props) {
  return (
    <div className="flex gap-1 mb-6">
      {PHASES.map((phase) => {
        const isCompleted = completedPhases.has(phase.key);
        const isCurrent = phase.key === currentPhase && status === "running";
        const isFailed = phase.key === currentPhase && status === "failed";

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
  );
}
