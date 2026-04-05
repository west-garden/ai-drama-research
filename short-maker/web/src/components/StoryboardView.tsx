import type { ShotSpec } from "../api";

interface Props {
  storyboard: ShotSpec[];
}

export default function StoryboardView({ storyboard }: Props) {
  // Group by episode
  const episodes = new Map<number, ShotSpec[]>();
  for (const shot of storyboard) {
    if (!episodes.has(shot.episode_number)) {
      episodes.set(shot.episode_number, []);
    }
    episodes.get(shot.episode_number)!.push(shot);
  }

  return (
    <div className="space-y-6">
      {Array.from(episodes.entries())
        .sort(([a], [b]) => a - b)
        .map(([epNum, shots]) => (
          <div key={epNum}>
            <h3 className="font-bold mb-3 text-gray-300 text-sm">
              第 {epNum} 集
              <span className="text-gray-600 font-normal ml-2">
                {shots.length} 镜
              </span>
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
              {shots
                .sort((a, b) => a.shot_number - b.shot_number)
                .map((shot) => (
                  <div
                    key={`${shot.episode_number}-${shot.shot_number}`}
                    className="bg-gray-900 border border-gray-800 rounded-lg p-3 text-xs"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <span className="font-mono text-gray-500">
                        #{shot.shot_number}
                      </span>
                      <div className="flex gap-1">
                        {shot.frame_type && (
                          <span className="bg-purple-900/50 text-purple-400 px-1.5 py-0.5 rounded text-[10px]">
                            {shot.frame_type}
                          </span>
                        )}
                        {shot.camera_move && (
                          <span className="bg-cyan-900/50 text-cyan-400 px-1.5 py-0.5 rounded text-[10px]">
                            {shot.camera_move}
                          </span>
                        )}
                      </div>
                    </div>
                    {shot.prompt && (
                      <div className="text-gray-400 mb-2 line-clamp-3">
                        {shot.prompt}
                      </div>
                    )}
                    <div className="flex items-center gap-2 text-[10px] text-gray-600">
                      {shot.emotion && <span>{shot.emotion}</span>}
                      {shot.rhythm_position && (
                        <span className="bg-gray-800 px-1 py-0.5 rounded">
                          {shot.rhythm_position}
                        </span>
                      )}
                    </div>
                  </div>
                ))}
            </div>
          </div>
        ))}
    </div>
  );
}
