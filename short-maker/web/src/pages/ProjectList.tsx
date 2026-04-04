import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { listProjects } from "../api";
import type { ProjectSummary } from "../api";

const statusColors: Record<string, string> = {
  completed: "bg-green-500 text-black",
  processing: "bg-blue-500 text-white",
  running: "bg-blue-500 text-white",
  failed: "bg-red-500 text-white",
  created: "bg-gray-500 text-white",
};

export default function ProjectList() {
  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    listProjects()
      .then(setProjects)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return <div className="text-gray-500">加载中...</div>;
  }

  if (projects.length === 0) {
    return (
      <div className="text-center py-20">
        <p className="text-gray-500 mb-4">还没有项目</p>
        <Link
          to="/new"
          className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg"
        >
          创建第一个项目
        </Link>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {projects.map((p) => (
        <Link
          key={p.id}
          to={`/projects/${p.id}`}
          className="block border border-gray-800 rounded-lg p-4 hover:border-gray-600 transition-colors"
        >
          <div className="flex justify-between items-center mb-2">
            <span className="font-bold">{p.name}</span>
            <span
              className={`text-xs px-2 py-0.5 rounded ${statusColors[p.status] || statusColors.created}`}
            >
              {p.status}
            </span>
          </div>
          <div className="text-sm text-gray-500">
            {p.style} · {p.episode_count} 集
          </div>
          <div className="text-xs text-gray-600 mt-1">
            {new Date(p.created_at).toLocaleString()}
          </div>
        </Link>
      ))}
    </div>
  );
}
