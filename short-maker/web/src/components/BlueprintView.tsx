import type { StoryBlueprint } from "../api";

interface Props {
  blueprint: StoryBlueprint;
}

export default function BlueprintView({ blueprint }: Props) {
  return (
    <div className="space-y-6">
      {blueprint.world_view && (
        <div>
          <h3 className="text-sm font-bold text-gray-400 mb-2">世界观</h3>
          <p className="text-gray-300 whitespace-pre-wrap text-sm leading-relaxed bg-gray-900 rounded-lg p-4 border border-gray-800">
            {blueprint.world_view}
          </p>
        </div>
      )}

      {blueprint.characters && blueprint.characters.length > 0 && (
        <div>
          <h3 className="text-sm font-bold text-gray-400 mb-2">角色</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {blueprint.characters.map((ch) => (
              <div
                key={ch.id}
                className="bg-gray-900 border border-gray-800 rounded-lg p-3"
              >
                <div className="font-bold text-white text-sm mb-1">
                  {ch.name}
                </div>
                <div className="text-gray-400 text-xs mb-2">
                  {ch.description}
                </div>
                {ch.traits && ch.traits.length > 0 && (
                  <div className="flex flex-wrap gap-1">
                    {ch.traits.map((t, i) => (
                      <span
                        key={i}
                        className="text-[10px] bg-gray-800 text-gray-400 px-1.5 py-0.5 rounded"
                      >
                        {t}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {blueprint.episodes && blueprint.episodes.length > 0 && (
        <div>
          <h3 className="text-sm font-bold text-gray-400 mb-2">剧集</h3>
          <div className="space-y-2">
            {blueprint.episodes.map((ep) => (
              <div
                key={ep.number}
                className="bg-gray-900 border border-gray-800 rounded-lg p-3 flex items-start gap-3"
              >
                <div className="text-xs font-mono text-gray-500 shrink-0 w-10">
                  EP{ep.number}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-[10px] bg-blue-900/50 text-blue-400 px-1.5 py-0.5 rounded">
                      {ep.role}
                    </span>
                    {ep.emotion_arc && (
                      <span className="text-[10px] text-gray-500">
                        {ep.emotion_arc}
                      </span>
                    )}
                  </div>
                  {ep.synopsis && (
                    <div className="text-xs text-gray-400">{ep.synopsis}</div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {blueprint.relationships && blueprint.relationships.length > 0 && (
        <div>
          <h3 className="text-sm font-bold text-gray-400 mb-2">人物关系</h3>
          <div className="flex flex-wrap gap-2">
            {blueprint.relationships.map((rel, i) => (
              <div
                key={i}
                className="text-xs bg-gray-900 border border-gray-800 rounded-lg px-3 py-2"
              >
                <span className="text-white">{rel.character_a}</span>
                <span className="text-gray-600 mx-1">—</span>
                <span className="text-gray-400">{rel.type}</span>
                <span className="text-gray-600 mx-1">—</span>
                <span className="text-white">{rel.character_b}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
