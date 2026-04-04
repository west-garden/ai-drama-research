const API_BASE = "/api";

export interface ProjectSummary {
  id: string;
  name: string;
  style: string;
  episode_count: number;
  status: string;
  current_phase: string;
  created_at: string;
}

export interface GeneratedShot {
  shot_number: number;
  episode_number: number;
  image_path: string;
  video_path: string;
  grade: string;
  image_score: number;
  video_score: number;
}

export interface ProjectDetail {
  project: {
    id: string;
    name: string;
    style: string;
    episode_count: number;
    status: string;
  };
  pipeline_status: string;
  current_phase: string;
  blueprint?: any;
  storyboard?: any[];
  images?: GeneratedShot[];
  videos?: GeneratedShot[];
  errors?: string[];
}

export interface SSEEvent {
  type: "phase_start" | "phase_complete" | "done" | "error";
  phase?: string;
  message?: string;
}

export async function createProject(form: FormData): Promise<any> {
  const res = await fetch(`${API_BASE}/projects`, {
    method: "POST",
    body: form,
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function listProjects(): Promise<ProjectSummary[]> {
  const res = await fetch(`${API_BASE}/projects`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function getProject(id: string): Promise<ProjectDetail> {
  const res = await fetch(`${API_BASE}/projects/${id}`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export function subscribeToEvents(
  id: string,
  onEvent: (e: SSEEvent) => void
): EventSource {
  const es = new EventSource(`${API_BASE}/projects/${id}/events`);
  es.onmessage = (e) => onEvent(JSON.parse(e.data));
  return es;
}
