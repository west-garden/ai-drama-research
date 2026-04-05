import { useState } from "react";
import type { ShotSpec } from "../api";

interface Props {
  storyboard: ShotSpec[];
}

function PromptText({ text }: { text: string }) {
  const [expanded, setExpanded] = useState(false);
  const needsExpand = text.split("\n").length > 3 || text.length > 200;

  return (
    <div
      className={`text-gray-400 mb-2 ${needsExpand ? "cursor-pointer" : ""} ${!expanded && needsExpand ? "line-clamp-3" : ""}`}
      onClick={() => needsExpand && setExpanded(!expanded)}
      title={needsExpand ? (expanded ? "点击收起" : "点击展开") : undefined}
    >
      {text}
      {needsExpand && !expanded && (
        <span className="text-gray-600 ml-1">...</span>
      )}
    </div>
  );
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
                        {shot.content_type && (
                          <span className="bg-orange-900/50 text-orange-400 px-1.5 py-0.5 rounded text-[10px]">
                            {shot.content_type}
                          </span>
                        )}
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

                    {shot.composition && (
                      <div className="text-gray-500 text-[10px] mb-1">
                        <span className="text-gray-600">构图: </span>
                        {shot.composition}
                      </div>
                    )}

                    {shot.character_refs && shot.character_refs.length > 0 && (
                      <div className="flex flex-wrap gap-1 mb-2">
                        {shot.character_refs.map((ref, i) => (
                          <span
                            key={i}
                            className="text-[10px] bg-indigo-900/40 text-indigo-400 px-1.5 py-0.5 rounded"
                          >
                            {ref}
                          </span>
                        ))}
                      </div>
                    )}

                    {shot.prompt && <PromptText text={shot.prompt} />}

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
