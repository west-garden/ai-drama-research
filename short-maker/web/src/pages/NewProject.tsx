import { useState, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { createProject } from "../api";

export default function NewProject() {
  const navigate = useNavigate();
  const fileRef = useRef<HTMLInputElement>(null);
  const [name, setName] = useState("");
  const [style, setStyle] = useState("manga");
  const [episodes, setEpisodes] = useState(10);
  const [fileName, setFileName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    const file = fileRef.current?.files?.[0];
    if (!file) {
      setError("请上传剧本文件");
      return;
    }

    setSubmitting(true);
    try {
      const form = new FormData();
      form.append("script", file);
      form.append("name", name || file.name.replace(/\.\w+$/, ""));
      form.append("style", style);
      form.append("episodes", String(episodes));

      const project = await createProject(form);
      navigate(`/projects/${project.id}`);
    } catch (err: any) {
      setError(err.message || "创建失败");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="max-w-md mx-auto">
      <h2 className="text-xl font-bold mb-6">新建项目</h2>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm text-gray-400 mb-1">项目名称</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="留空则使用文件名"
            className="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-white"
          />
        </div>

        <div>
          <label className="block text-sm text-gray-400 mb-1">剧本文件</label>
          <div
            onClick={() => fileRef.current?.click()}
            className="border-2 border-dashed border-gray-700 rounded-lg p-6 text-center cursor-pointer hover:border-gray-500 transition-colors"
          >
            {fileName ? (
              <span className="text-white">{fileName}</span>
            ) : (
              <span className="text-gray-500">点击上传 .txt 文件</span>
            )}
          </div>
          <input
            ref={fileRef}
            type="file"
            accept=".txt"
            className="hidden"
            onChange={(e) => setFileName(e.target.files?.[0]?.name || "")}
          />
        </div>

        <div>
          <label className="block text-sm text-gray-400 mb-1">风格</label>
          <select
            value={style}
            onChange={(e) => setStyle(e.target.value)}
            className="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-white"
          >
            <option value="manga">manga（漫画）</option>
            <option value="3d">3D</option>
            <option value="live_action">live_action（真人）</option>
          </select>
        </div>

        <div>
          <label className="block text-sm text-gray-400 mb-1">集数</label>
          <input
            type="number"
            value={episodes}
            onChange={(e) => setEpisodes(Number(e.target.value))}
            min={1}
            max={100}
            className="w-full bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-white"
          />
        </div>

        {error && <div className="text-red-400 text-sm">{error}</div>}

        <button
          type="submit"
          disabled={submitting}
          className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 text-white py-3 rounded-lg font-medium"
        >
          {submitting ? "创建中..." : "开始生成"}
        </button>
      </form>
    </div>
  );
}
