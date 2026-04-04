import { useState } from "react";
import type { GeneratedShot } from "../api";

const gradeColors: Record<string, string> = {
  S: "bg-amber-500 text-black",
  A: "bg-blue-500 text-white",
  B: "bg-green-500 text-black",
  C: "bg-gray-500 text-white",
};

interface Props {
  images: GeneratedShot[];
  videos: GeneratedShot[];
}

export default function ShotGallery({ images, videos }: Props) {
  const [selectedShot, setSelectedShot] = useState<GeneratedShot | null>(null);
  const [showVideo, setShowVideo] = useState(false);

  const episodes = new Map<number, GeneratedShot[]>();
  for (const img of images) {
    if (!episodes.has(img.episode_number)) {
      episodes.set(img.episode_number, []);
    }
    episodes.get(img.episode_number)!.push(img);
  }

  const videoMap = new Map<string, GeneratedShot>();
  for (const vid of videos) {
    videoMap.set(`${vid.episode_number}-${vid.shot_number}`, vid);
  }

  return (
    <div>
      {Array.from(episodes.entries())
        .sort(([a], [b]) => a - b)
        .map(([epNum, shots]) => (
          <div key={epNum} className="mb-6">
            <h3 className="font-bold mb-3 text-gray-300">第 {epNum} 集</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
              {shots
                .sort((a, b) => a.shot_number - b.shot_number)
                .map((shot) => (
                  <div
                    key={shot.shot_number}
                    onClick={() => {
                      setSelectedShot(shot);
                      setShowVideo(false);
                    }}
                    className="border border-gray-800 rounded-lg overflow-hidden cursor-pointer hover:border-gray-600 transition-colors"
                  >
                    <div className="aspect-video bg-gray-900 flex items-center justify-center">
                      <img
                        src={shot.image_path}
                        alt={`Shot ${shot.shot_number}`}
                        className="w-full h-full object-cover"
                        onError={(e) => {
                          (e.target as HTMLImageElement).style.display = "none";
                        }}
                      />
                    </div>
                    <div className="p-2 text-xs">
                      <div className="flex justify-between items-center">
                        <span>Shot {shot.shot_number}</span>
                        <span
                          className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${gradeColors[shot.grade] || "bg-gray-700"}`}
                        >
                          {shot.grade}
                        </span>
                      </div>
                      <div className="text-gray-500 mt-0.5">
                        {shot.image_score}分
                      </div>
                    </div>
                  </div>
                ))}
            </div>
          </div>
        ))}

      {selectedShot && (
        <div
          className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-8"
          onClick={() => setSelectedShot(null)}
        >
          <div
            className="max-w-3xl w-full bg-gray-900 rounded-xl overflow-hidden"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="aspect-video bg-black flex items-center justify-center">
              {showVideo ? (
                <video
                  src={
                    videoMap.get(
                      `${selectedShot.episode_number}-${selectedShot.shot_number}`
                    )?.video_path
                  }
                  controls
                  autoPlay
                  className="w-full h-full"
                />
              ) : (
                <img
                  src={selectedShot.image_path}
                  alt={`Shot ${selectedShot.shot_number}`}
                  className="w-full h-full object-contain"
                />
              )}
            </div>
            <div className="p-4 flex justify-between items-center">
              <div>
                <span className="font-bold">
                  EP{selectedShot.episode_number} Shot{" "}
                  {selectedShot.shot_number}
                </span>
                <span
                  className={`ml-2 px-2 py-0.5 rounded text-xs font-bold ${gradeColors[selectedShot.grade] || ""}`}
                >
                  {selectedShot.grade}
                </span>
                <span className="ml-2 text-gray-500 text-sm">
                  图片 {selectedShot.image_score}分
                </span>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => setShowVideo(false)}
                  className={`px-3 py-1 rounded text-sm ${!showVideo ? "bg-blue-600" : "bg-gray-700"}`}
                >
                  图片
                </button>
                <button
                  onClick={() => setShowVideo(true)}
                  className={`px-3 py-1 rounded text-sm ${showVideo ? "bg-blue-600" : "bg-gray-700"}`}
                >
                  视频
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
