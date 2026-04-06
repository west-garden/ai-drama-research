import { useCallback, useMemo } from "react";
import {
  ReactFlow,
  Background,
  type Node,
  type Edge,
  type NodeTypes,
  type NodeProps,
  Handle,
  Position,
  useNodesState,
  useEdgesState,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import type { WorkflowNode, WorkflowEdge } from "../api";

// --- Layout constants ---
const COL_WIDTH = 180;
const ROW_HEIGHT = 64;
const START_X = 40;
const START_Y = 40;
const EP_START_COL = 3; // columns 3,4,5 for storyboard/image/video

// --- Status colors ---
const STATUS_STYLES: Record<
  string,
  { bg: string; border: string; text: string; ring?: string }
> = {
  pending: {
    bg: "bg-gray-800",
    border: "border-gray-600",
    text: "text-gray-400",
  },
  running: {
    bg: "bg-blue-900/50",
    border: "border-blue-500",
    text: "text-blue-300",
    ring: "ring-2 ring-blue-500/50 animate-pulse",
  },
  completed: {
    bg: "bg-green-900/40",
    border: "border-green-600",
    text: "text-green-300",
  },
  failed: {
    bg: "bg-red-900/40",
    border: "border-red-600",
    text: "text-red-300",
  },
  skipped: {
    bg: "bg-yellow-900/40",
    border: "border-yellow-600",
    text: "text-yellow-300",
  },
};

const STATUS_ICONS: Record<string, string> = {
  pending: "○",
  running: "●",
  completed: "✓",
  failed: "✗",
  skipped: "⊘",
};

// --- Custom Node: ProjectNode ---
function ProjectNode({ data }: NodeProps) {
  const d = data as { label: string; style: string; episodes: number; language: string };
  return (
    <div className="bg-gray-900 border-2 border-gray-500 rounded-xl px-4 py-3 min-w-[140px]">
      <Handle type="source" position={Position.Right} className="!bg-gray-500" />
      <div className="text-sm font-bold text-white">{d.label}</div>
      <div className="text-xs text-gray-400 mt-1">
        {d.style} · {d.episodes}集 · {d.language === "zh" ? "中文" : "EN"}
      </div>
    </div>
  );
}

// --- Custom Node: PhaseNode (project-level) ---
function PhaseNode({ data }: NodeProps) {
  const d = data as {
    label: string;
    status: string;
    canRun: boolean;
    selected: boolean;
    error?: string;
    onRun: () => void;
    onSelect: () => void;
  };
  const s = STATUS_STYLES[d.status] || STATUS_STYLES.pending;
  const selectedClass = d.selected ? "ring-2 ring-white/70 shadow-lg shadow-white/10" : "";

  return (
    <div
      className={`${s.bg} border-2 ${s.border} ${s.ring || ""} ${selectedClass} rounded-xl px-4 py-3 min-w-[140px] cursor-pointer transition-all hover:brightness-125`}
      onClick={d.onSelect}
    >
      <Handle type="target" position={Position.Left} className="!bg-gray-500" />
      <Handle type="source" position={Position.Right} className="!bg-gray-500" />
      <div className="flex items-center gap-2">
        <span className={`text-sm ${s.text}`}>{STATUS_ICONS[d.status]}</span>
        <span className="text-sm font-medium text-white">{d.label}</span>
      </div>
      {d.error && (
        <div className="text-xs text-red-400 mt-1 truncate max-w-[160px]" title={d.error}>
          {d.error}
        </div>
      )}
      {d.canRun && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            d.onRun();
          }}
          className="mt-2 w-full bg-blue-600 hover:bg-blue-700 text-white text-xs py-1 rounded-lg"
        >
          运行
        </button>
      )}
    </div>
  );
}

// --- Custom Node: EpisodePhaseNode (per-episode) ---
function EpisodePhaseNode({ data }: NodeProps) {
  const d = data as {
    label: string;
    status: string;
    canRun: boolean;
    selected: boolean;
    error?: string;
    onRun: () => void;
    onSelect: () => void;
  };
  const s = STATUS_STYLES[d.status] || STATUS_STYLES.pending;
  const selectedClass = d.selected ? "ring-2 ring-white/70 shadow-lg shadow-white/10" : "";

  return (
    <div
      className={`${s.bg} border ${s.border} ${s.ring || ""} ${selectedClass} rounded-lg px-3 py-2 min-w-[120px] cursor-pointer transition-all hover:brightness-125`}
      onClick={d.onSelect}
    >
      <Handle type="target" position={Position.Left} className="!bg-gray-500 !w-2 !h-2" />
      <Handle type="source" position={Position.Right} className="!bg-gray-500 !w-2 !h-2" />
      <div className="flex items-center gap-1.5">
        <span className={`text-xs ${s.text}`}>{STATUS_ICONS[d.status]}</span>
        <span className="text-xs font-medium text-white truncate">{d.label}</span>
      </div>
      {d.error && (
        <div className="text-[10px] text-red-400 mt-0.5 truncate max-w-[120px]" title={d.error}>
          {d.error}
        </div>
      )}
      {d.canRun && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            d.onRun();
          }}
          className="mt-1 w-full bg-blue-600 hover:bg-blue-700 text-white text-[10px] py-0.5 rounded"
        >
          运行
        </button>
      )}
    </div>
  );
}

const nodeTypes: NodeTypes = {
  projectNode: ProjectNode,
  phaseNode: PhaseNode,
  episodePhaseNode: EpisodePhaseNode,
};

// --- Edge styling ---
function getEdgeStyle(
  sourceStatus: string,
  targetStatus: string
): { stroke: string; strokeDasharray?: string; animated?: boolean } {
  if (sourceStatus === "completed" && targetStatus === "completed") {
    return { stroke: "#22c55e" }; // green solid
  }
  if (sourceStatus === "running" || targetStatus === "running") {
    return { stroke: "#3b82f6", strokeDasharray: "5 5", animated: true }; // blue animated
  }
  return { stroke: "#4b5563", strokeDasharray: "4 4" }; // gray dashed
}

// --- Main component ---
interface Props {
  workflowNodes: WorkflowNode[];
  workflowEdges: WorkflowEdge[];
  projectStyle: string;
  episodeCount: number;
  promptLanguage: string;
  selectedNode: string | null;
  onRunNode: (nodeId: string) => void;
  onSelectNode: (nodeId: string | null) => void;
}

export default function WorkflowGraph({
  workflowNodes,
  workflowEdges,
  projectStyle,
  episodeCount,
  promptLanguage,
  selectedNode,
  onRunNode,
  onSelectNode,
}: Props) {
  // Build node status lookup
  const nodeStatusMap = useMemo(() => {
    const map: Record<string, WorkflowNode> = {};
    for (const n of workflowNodes) {
      map[n.id] = n;
    }
    return map;
  }, [workflowNodes]);

  // Compute phase column positions
  const phaseCol: Record<string, number> = {
    story_understanding: 1,
    character_asset: 2,
    storyboard: EP_START_COL,
    image_generation: EP_START_COL + 1,
    video_generation: EP_START_COL + 2,
  };

  // Build ReactFlow nodes
  const rfNodes: Node[] = useMemo(() => {
    const nodes: Node[] = [];

    // Project config node (col 0)
    nodes.push({
      id: "project_config",
      type: "projectNode",
      position: { x: START_X, y: START_Y + (episodeCount * ROW_HEIGHT) / 2 - ROW_HEIGHT / 2 },
      data: {
        label: "项目配置",
        style: projectStyle,
        episodes: episodeCount,
        language: promptLanguage,
      },
      draggable: false,
    });

    // Project-level phase nodes (col 1, 2)
    for (const wn of workflowNodes) {
      if (wn.episode === 0) {
        const col = phaseCol[wn.phase] ?? 1;
        nodes.push({
          id: wn.id,
          type: "phaseNode",
          position: {
            x: START_X + col * COL_WIDTH,
            y: START_Y + (episodeCount * ROW_HEIGHT) / 2 - ROW_HEIGHT / 2,
          },
          data: {
            label: wn.label,
            status: wn.status,
            canRun: wn.can_run,
            selected: selectedNode === wn.id,
            error: wn.error,
            onRun: () => onRunNode(wn.id),
            onSelect: () => onSelectNode(wn.id),
          },
          draggable: false,
        });
      }
    }

    // Episode-level nodes
    for (const wn of workflowNodes) {
      if (wn.episode > 0) {
        const col = phaseCol[wn.phase] ?? EP_START_COL;
        const row = wn.episode - 1;
        nodes.push({
          id: wn.id,
          type: "episodePhaseNode",
          position: {
            x: START_X + col * COL_WIDTH,
            y: START_Y + row * ROW_HEIGHT,
          },
          data: {
            label: wn.label,
            status: wn.status,
            canRun: wn.can_run,
            selected: selectedNode === wn.id,
            error: wn.error,
            onRun: () => onRunNode(wn.id),
            onSelect: () => onSelectNode(wn.id),
          },
          draggable: false,
        });
      }
    }

    return nodes;
  }, [workflowNodes, episodeCount, projectStyle, promptLanguage, selectedNode, onRunNode, onSelectNode]);

  // Build ReactFlow edges
  const rfEdges: Edge[] = useMemo(() => {
    return workflowEdges.map((e, i) => {
      const srcNode = nodeStatusMap[e.source];
      const tgtNode = nodeStatusMap[e.target];
      const srcStatus = srcNode?.status || "pending";
      const tgtStatus = tgtNode?.status || "pending";
      // For project_config source, treat as completed always
      const effectiveSrcStatus = e.source === "project_config" ? "completed" : srcStatus;
      const style = getEdgeStyle(effectiveSrcStatus, tgtStatus);

      return {
        id: `e-${i}`,
        source: e.source,
        target: e.target,
        style: { stroke: style.stroke, strokeDasharray: style.strokeDasharray },
        animated: style.animated || false,
      };
    });
  }, [workflowEdges, nodeStatusMap]);

  const [nodes, , onNodesChange] = useNodesState(rfNodes);
  const [edges, , onEdgesChange] = useEdgesState(rfEdges);

  // Keep nodes/edges in sync with props
  // Since useNodesState/useEdgesState initializes once, we use the raw rfNodes/rfEdges
  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      if (node.id !== "project_config") {
        onSelectNode(node.id);
      }
    },
    [onSelectNode]
  );

  return (
    <div className="border-y border-gray-700 bg-gray-950 overflow-hidden h-full">
      <ReactFlow
        nodes={rfNodes}
        edges={rfEdges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={onNodeClick}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        minZoom={0.3}
        maxZoom={1.5}
        proOptions={{ hideAttribution: true }}
      >
        <Background color="#374151" gap={20} />
      </ReactFlow>
    </div>
  );
}
