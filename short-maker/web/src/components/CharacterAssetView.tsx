import type { Asset } from "../api";

interface Props {
  assets: Asset[];
}

export default function CharacterAssetView({ assets }: Props) {
  if (!assets || assets.length === 0) return null;

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {assets.map((asset) => (
          <div
            key={asset.id}
            className="bg-gray-900 border border-gray-800 rounded-lg p-4"
          >
            <div className="flex items-center gap-2 mb-2">
              <span className="font-bold text-white text-sm">
                {asset.metadata?.character_id || asset.name}
              </span>
              <span className="text-[10px] bg-indigo-900/50 text-indigo-400 px-1.5 py-0.5 rounded">
                {asset.type}
              </span>
            </div>

            {asset.metadata?.visual_prompt && (
              <div className="mb-3">
                <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-1">
                  Visual Prompt
                </div>
                <div className="text-xs text-gray-300 bg-gray-800/50 rounded p-2 whitespace-pre-wrap leading-relaxed">
                  {asset.metadata.visual_prompt}
                </div>
              </div>
            )}

            <div className="grid grid-cols-1 gap-2 mb-2">
              {asset.metadata?.face && (
                <div>
                  <span className="text-[10px] text-gray-500">面部: </span>
                  <span className="text-xs text-gray-400">
                    {asset.metadata.face}
                  </span>
                </div>
              )}
              {asset.metadata?.body && (
                <div>
                  <span className="text-[10px] text-gray-500">体型: </span>
                  <span className="text-xs text-gray-400">
                    {asset.metadata.body}
                  </span>
                </div>
              )}
              {asset.metadata?.clothing && (
                <div>
                  <span className="text-[10px] text-gray-500">服装: </span>
                  <span className="text-xs text-gray-400">
                    {asset.metadata.clothing}
                  </span>
                </div>
              )}
            </div>

            {asset.tags && asset.tags.length > 0 && (
              <div className="flex flex-wrap gap-1 mt-2">
                {asset.tags.map((tag, i) => (
                  <span
                    key={i}
                    className="text-[10px] bg-gray-800 text-gray-400 px-1.5 py-0.5 rounded"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
