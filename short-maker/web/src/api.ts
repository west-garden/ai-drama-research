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

export interface SceneTag {
  narrative_beat: string;
  emotion_arc: string;
  setting: string;
  pacing: string;
  character_count: number;
}

export interface Relationship {
  character_a: string;
  character_b: string;
  type: string;
}

export interface CharacterProfile {
  id: string;
  name: string;
  description: string;
  traits: string[];
}

export interface EpisodeBlueprint {
  number: number;
  role: string;
  emotion_arc: string;
  scenes: SceneTag[];
  synopsis: string;
}

export interface StoryBlueprint {
  project_id: string;
  world_view: string;
  characters: CharacterProfile[];
  episodes: EpisodeBlueprint[];
  relationships: Relationship[];
}

export interface ShotSpec {
  episode_number: number;
  shot_number: number;
  frame_type: string;
  composition: string;
  camera_move: string;
  emotion: string;
  prompt: string;
  character_refs: string[];
  scene_ref: string;
  rhythm_position: string;
  content_type: string;
}

export interface Asset {
  id: string;
  name: string;
  type: string;
  scope: string;
  project_id?: string;
  file_path: string;
  tags: string[];
  metadata: Record<string, string>;
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
  next_phase: string;
  blueprint?: StoryBlueprint;
  assets?: Asset[];
  storyboard?: ShotSpec[];
  images?: GeneratedShot[];
  videos?: GeneratedShot[];
  errors?: string[];
}

export interface SSEEvent {
  type: "phase_start" | "phase_complete" | "done" | "error" | "paused";
  phase?: string;
  message?: string;
}

export interface RunPhaseRequest {
  phase?: string;
  episode?: number;
}

export interface RunPhaseResponse {
  status: string;
  phase: string;
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

export async function runPhase(
  projectId: string,
  req: RunPhaseRequest
): Promise<RunPhaseResponse> {
  const res = await fetch(`${API_BASE}/projects/${projectId}/run-phase`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
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
