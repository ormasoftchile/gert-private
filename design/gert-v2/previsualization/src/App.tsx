import { useMemo, useState } from "react";
import {
  ReactFlow,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  Panel,
  type Edge,
  type Node,
} from "@xyflow/react";
import type { NodeTypes } from "@xyflow/react";
import { RunbookNode } from "./components/RunbookNode";
import { layoutNodes } from "./flows/layout";
import { scenarios } from "./flows/scenarios";
import type { RunbookNodeData } from "./types";

const nodeTypes: NodeTypes = {
  runbook: RunbookNode,
};

type VisualizationMode =
  | "flow"
  | "flow-sankey"
  | "sequence"
  | "stereotype-lanes"
  | "governance-risk"
  | "radial-constellation"
  | "critical-path"
  | "failure-paths"
  | "sankey-emphasis"
  | "concern-matrix";

const modeLabel: Record<VisualizationMode, string> = {
  flow: "Operational Flow",
  "flow-sankey": "Operational Sankey",
  sequence: "Sequence Pipeline",
  "stereotype-lanes": "Stereotype Lanes",
  "governance-risk": "Governance & Risk Lens",
  "radial-constellation": "Radial Constellation",
  "critical-path": "Critical Path Focus",
  "failure-paths": "Failure & Escalation Lens",
  "sankey-emphasis": "Sankey Emphasis",
  "concern-matrix": "Concern Matrix",
};

const modeSummary: Record<VisualizationMode, string> = {
  flow: "Best for execution semantics with explicit branching and joins.",
  "flow-sankey": "Operational topology with weighted transition bands so dominant fan-outs stand out.",
  sequence: "Best for seeing end-to-end progression and step order quickly.",
  "stereotype-lanes": "Best for responsibility split by step stereotype across the runbook.",
  "governance-risk": "Highlights control, decision, compensation, and terminal nodes; de-emphasizes pure operational details.",
  "radial-constellation": "Places control pivots near the center and spreads related steps by graph distance for a systems-level map.",
  "critical-path": "Highlights one likely happy-path progression and fades alternates for fast narrative review.",
  "failure-paths": "Emphasizes reject/escalate/fail/timeout/rollback routes and downplays non-exception paths.",
  "sankey-emphasis": "Uses weighted connectors to show dominant fan-out and stronger transition corridors.",
  "concern-matrix": "Maps steps into concern columns to compare operational, human, control, and outcome coverage.",
};

function lanePositioned(nodes: Node<RunbookNodeData>[]): Node<RunbookNodeData>[] {
  const laneOrder: RunbookNodeData["stereotype"][] = [
    "trigger",
    "decision",
    "human",
    "control",
    "automation",
    "compensation",
    "terminal",
  ];

  const laneIndex = new Map<RunbookNodeData["stereotype"], number>();
  laneOrder.forEach((lane, index) => laneIndex.set(lane, index));

  const sorted = [...nodes].sort((a, b) => a.position.y - b.position.y);
  return sorted.map((node, i) => {
    const lane = laneIndex.get(node.data.stereotype) ?? laneOrder.length;
    return {
      ...node,
      position: {
        x: 80 + lane * 305,
        y: 90 + i * 145,
      },
    };
  });
}

function radialPositioned(
  nodes: Node<RunbookNodeData>[],
  edges: Edge[],
): Node<RunbookNodeData>[] {
  if (nodes.length === 0) {
    return nodes;
  }

  const anchor =
    nodes.find((node) => node.data.stereotype === "decision") ??
    nodes.find((node) => node.data.stereotype === "control") ??
    nodes[0];

  const adjacency = new Map<string, Set<string>>();
  for (const node of nodes) {
    adjacency.set(node.id, new Set());
  }
  for (const edge of edges) {
    adjacency.get(edge.source)?.add(edge.target);
    adjacency.get(edge.target)?.add(edge.source);
  }

  const depth = new Map<string, number>();
  const queue: string[] = [anchor.id];
  depth.set(anchor.id, 0);

  while (queue.length > 0) {
    const current = queue.shift();
    if (!current) {
      continue;
    }
    const currentDepth = depth.get(current) ?? 0;
    for (const next of adjacency.get(current) ?? []) {
      if (!depth.has(next)) {
        depth.set(next, currentDepth + 1);
        queue.push(next);
      }
    }
  }

  const groups = new Map<number, Node<RunbookNodeData>[]>();
  for (const node of nodes) {
    const d = depth.get(node.id) ?? 4;
    if (!groups.has(d)) {
      groups.set(d, []);
    }
    groups.get(d)?.push(node);
  }

  const centerX = 760;
  const centerY = 460;
  const ring = 235;
  const positioned = new Map<string, { x: number; y: number }>();
  positioned.set(anchor.id, { x: centerX, y: centerY });

  const depths = [...groups.keys()].sort((a, b) => a - b);
  for (const d of depths) {
    if (d === 0) {
      continue;
    }
    const group = groups.get(d) ?? [];
    const count = group.length;
    if (count === 0) {
      continue;
    }
    group.sort((a, b) => a.id.localeCompare(b.id));

    for (let i = 0; i < count; i += 1) {
      const angle = (-Math.PI / 2) + (i / count) * Math.PI * 2;
      const radius = d * ring;
      positioned.set(group[i].id, {
        x: centerX + Math.cos(angle) * radius,
        y: centerY + Math.sin(angle) * radius,
      });
    }
  }

  return nodes.map((node) => ({
    ...node,
    position: positioned.get(node.id) ?? node.position,
  }));
}

function scoreEdgeLabel(label: string | undefined): number {
  const text = (label ?? "").toLowerCase();
  if (text.includes("approve") || text.includes("healthy") || text.includes("success")) {
    return 3;
  }
  if (text.includes("yes") || text.includes("continue") || text.includes("proceed")) {
    return 2;
  }
  if (text.includes("reject") || text.includes("escalat") || text.includes("fail") || text.includes("rollback")) {
    return -2;
  }
  return 1;
}

function likelyCriticalPath(
  nodes: Node<RunbookNodeData>[],
  edges: Edge[],
): { nodeIds: Set<string>; edgeIds: Set<string> } {
  const start =
    nodes.find((node) => node.data.stereotype === "trigger") ??
    nodes.find((node) => node.data.stereotype === "control") ??
    nodes[0];

  if (!start) {
    return { nodeIds: new Set(), edgeIds: new Set() };
  }

  const outgoing = new Map<string, Edge[]>();
  for (const edge of edges) {
    const list = outgoing.get(edge.source) ?? [];
    list.push(edge);
    outgoing.set(edge.source, list);
  }

  const nodeIds = new Set<string>([start.id]);
  const edgeIds = new Set<string>();
  const visited = new Set<string>([start.id]);
  let current = start.id;
  let guard = 0;

  while (guard < nodes.length * 2) {
    guard += 1;
    const candidates = outgoing.get(current) ?? [];
    if (candidates.length === 0) {
      break;
    }

    const sorted = [...candidates].sort((a, b) => scoreEdgeLabel(String(b.label)) - scoreEdgeLabel(String(a.label)));
    const next = sorted.find((edge) => !visited.has(edge.target)) ?? sorted[0];
    if (!next) {
      break;
    }

    edgeIds.add(next.id);
    nodeIds.add(next.target);
    visited.add(next.target);

    if (nodes.find((node) => node.id === next.target)?.data.stereotype === "terminal") {
      break;
    }
    current = next.target;
  }

  return { nodeIds, edgeIds };
}

function isFailureLabel(label: string | undefined): boolean {
  const text = (label ?? "").toLowerCase();
  return (
    text.includes("reject") ||
    text.includes("escalat") ||
    text.includes("fail") ||
    text.includes("rollback") ||
    text.includes("timeout") ||
    text.includes("deny")
  );
}

function depthByFlow(nodes: Node<RunbookNodeData>[], edges: Edge[]): Map<string, number> {
  const incoming = new Map<string, number>();
  const outgoing = new Map<string, string[]>();

  for (const node of nodes) {
    incoming.set(node.id, 0);
    outgoing.set(node.id, []);
  }

  for (const edge of edges) {
    incoming.set(edge.target, (incoming.get(edge.target) ?? 0) + 1);
    const list = outgoing.get(edge.source) ?? [];
    list.push(edge.target);
    outgoing.set(edge.source, list);
  }

  const depth = new Map<string, number>();
  const queue: string[] = [];

  for (const [id, count] of incoming) {
    if (count === 0) {
      depth.set(id, 0);
      queue.push(id);
    }
  }

  if (queue.length === 0 && nodes[0]) {
    depth.set(nodes[0].id, 0);
    queue.push(nodes[0].id);
  }

  while (queue.length > 0) {
    const current = queue.shift();
    if (!current) {
      continue;
    }
    const d = depth.get(current) ?? 0;
    for (const next of outgoing.get(current) ?? []) {
      const candidate = d + 1;
      if (!depth.has(next) || (depth.get(next) ?? 0) < candidate) {
        depth.set(next, candidate);
      }
      incoming.set(next, (incoming.get(next) ?? 1) - 1);
      if ((incoming.get(next) ?? 0) <= 0) {
        queue.push(next);
      }
    }
  }

  for (const node of nodes) {
    if (!depth.has(node.id)) {
      depth.set(node.id, 1);
    }
  }

  return depth;
}

function concernMatrixPositioned(
  nodes: Node<RunbookNodeData>[],
  edges: Edge[],
): Node<RunbookNodeData>[] {
  const concernFor = (s: RunbookNodeData["stereotype"]): number => {
    if (s === "trigger" || s === "automation") {
      return 0; // Observe/Execute
    }
    if (s === "human" || s === "decision") {
      return 1; // Human/Judgment
    }
    if (s === "control" || s === "compensation") {
      return 2; // Orchestration/Risk
    }
    return 3; // Outcome
  };

  const depth = depthByFlow(nodes, edges);
  const rows = new Map<number, number>();
  const positioned = [...nodes].sort((a, b) => (depth.get(a.id) ?? 0) - (depth.get(b.id) ?? 0));

  return positioned.map((node) => {
    const col = concernFor(node.data.stereotype);
    const d = depth.get(node.id) ?? 0;
    const rowKey = d * 10 + col;
    const slot = rows.get(rowKey) ?? 0;
    rows.set(rowKey, slot + 1);

    return {
      ...node,
      className: "matrix-node",
      position: {
        x: 80 + col * 360,
        y: 90 + d * 165 + slot * 52,
      },
    };
  });
}

export function App() {
  const [scenarioId, setScenarioId] = useState(scenarios[0].id);
  const [mode, setMode] = useState<VisualizationMode>("flow");

  const scenario = useMemo(
    () => scenarios.find((item) => item.id === scenarioId) ?? scenarios[0],
    [scenarioId],
  );

  const rendered = useMemo(() => {
    const baseNodes = scenario.nodes.map((node) => ({ ...node }));
    const baseEdges = scenario.edges.map((edge) => ({ ...edge }));

    if (mode === "flow") {
      return { nodes: baseNodes, edges: baseEdges };
    }

    if (mode === "flow-sankey") {
      const outCounts = new Map<string, number>();
      for (const edge of baseEdges) {
        outCounts.set(edge.source, (outCounts.get(edge.source) ?? 0) + 1);
      }

      const nodes = baseNodes.map((node) => ({
        ...node,
        className: "flow-sankey-node",
      }));

      const edges = baseEdges.map((edge) => {
        const sourceFanOut = outCounts.get(edge.source) ?? 1;
        const branchBoost = sourceFanOut > 1 ? 1.3 : 1;
        const baseWidth = 2.2;
        const width = Math.min(8.5, baseWidth + sourceFanOut * branchBoost);
        const failure = isFailureLabel(String(edge.label));

        return {
          ...edge,
          type: "smoothstep" as const,
          animated: false,
          style: {
            stroke: failure ? "#b42318" : "#225f93",
            strokeWidth: width,
            opacity: failure ? 0.9 : 0.72,
          },
          labelStyle: {
            color: failure ? "#7a1414" : "#1f3d58",
            fontWeight: 650,
          },
        };
      });

      return { nodes, edges };
    }

    if (mode === "sequence") {
      const nodes = layoutNodes(baseNodes, baseEdges, "LR");
      const edges = baseEdges.map((edge) => ({ ...edge, type: "step" as const }));
      return { nodes, edges };
    }

    if (mode === "governance-risk") {
      const focusStereotypes = new Set(["decision", "control", "compensation", "terminal"]);
      const nodes = baseNodes.map((node) => ({
        ...node,
        className: focusStereotypes.has(node.data.stereotype) ? "gov-focus" : "gov-muted",
      }));

      const nodeMap = new Map(nodes.map((node) => [node.id, node]));
      const edges = baseEdges.map((edge) => {
        const source = nodeMap.get(edge.source);
        const target = nodeMap.get(edge.target);
        const focus = source?.className === "gov-focus" || target?.className === "gov-focus";
        return {
          ...edge,
          type: "smoothstep" as const,
          animated: false,
          style: {
            stroke: focus ? "#d95f02" : "#98a8b8",
            strokeWidth: focus ? 2.4 : 1.2,
            opacity: focus ? 1 : 0.35,
          },
          labelStyle: {
            color: focus ? "#562e11" : "#73879a",
            fontWeight: focus ? 700 : 500,
          },
        };
      });

      return { nodes, edges };
    }

    if (mode === "radial-constellation") {
      const nodes = radialPositioned(baseNodes, baseEdges);
      const edges = baseEdges.map((edge) => ({
        ...edge,
        type: "smoothstep" as const,
        animated: true,
        style: {
          stroke: "#2f4c68",
          strokeWidth: 1.6,
          opacity: 0.7,
        },
      }));
      return { nodes, edges };
    }

    if (mode === "critical-path") {
      const path = likelyCriticalPath(baseNodes, baseEdges);
      const nodes = baseNodes.map((node) => ({
        ...node,
        className: path.nodeIds.has(node.id) ? "cp-focus" : "cp-muted",
      }));
      const edges = baseEdges.map((edge) => {
        const focused = path.edgeIds.has(edge.id);
        return {
          ...edge,
          type: "step" as const,
          animated: focused,
          style: {
            stroke: focused ? "#0b7a4d" : "#aebccc",
            strokeWidth: focused ? 2.8 : 1.1,
            opacity: focused ? 1 : 0.28,
          },
          labelStyle: {
            color: focused ? "#0b5e41" : "#8193a6",
            fontWeight: focused ? 700 : 500,
          },
        };
      });
      return { nodes, edges };
    }

    if (mode === "failure-paths") {
      const focusNodeIds = new Set<string>();
      for (const edge of baseEdges) {
        if (isFailureLabel(String(edge.label))) {
          focusNodeIds.add(edge.source);
          focusNodeIds.add(edge.target);
        }
      }
      const nodes = baseNodes.map((node) => ({
        ...node,
        className: focusNodeIds.has(node.id) ? "fail-focus" : "fail-muted",
      }));
      const edges = baseEdges.map((edge) => {
        const failure = isFailureLabel(String(edge.label));
        return {
          ...edge,
          type: "smoothstep" as const,
          animated: failure,
          style: {
            stroke: failure ? "#b42318" : "#94a3b8",
            strokeWidth: failure ? 2.8 : 1.1,
            opacity: failure ? 1 : 0.25,
            strokeDasharray: failure ? "6 3" : undefined,
          },
          labelStyle: {
            color: failure ? "#7a1414" : "#8293a4",
            fontWeight: failure ? 700 : 500,
          },
        };
      });
      return { nodes, edges };
    }

    if (mode === "sankey-emphasis") {
      const nodes = layoutNodes(baseNodes, baseEdges, "LR").map((node) => ({
        ...node,
        className: "sankey-node",
      }));

      const outCounts = new Map<string, number>();
      for (const edge of baseEdges) {
        outCounts.set(edge.source, (outCounts.get(edge.source) ?? 0) + 1);
      }

      const edges = baseEdges.map((edge) => {
        const weight = Math.max(1, outCounts.get(edge.source) ?? 1);
        const width = Math.min(7.5, 1.6 + weight * 1.2);
        return {
          ...edge,
          type: "smoothstep" as const,
          animated: true,
          style: {
            stroke: isFailureLabel(String(edge.label)) ? "#b42318" : "#2f6ea7",
            strokeWidth: width,
            opacity: isFailureLabel(String(edge.label)) ? 0.9 : 0.65,
          },
          labelStyle: {
            color: "#1f3d58",
            fontWeight: 650,
          },
        };
      });

      return { nodes, edges };
    }

    if (mode === "concern-matrix") {
      const nodes = concernMatrixPositioned(baseNodes, baseEdges);
      const edges = baseEdges.map((edge) => ({
        ...edge,
        type: "step" as const,
        animated: false,
        style: {
          stroke: "#4f6b84",
          strokeWidth: 1.6,
          opacity: 0.65,
        },
      }));
      return { nodes, edges };
    }

    const nodes = lanePositioned(baseNodes);
    const edges = baseEdges.map((edge) => ({
      ...edge,
      type: "straight" as const,
      animated: false,
    }));
    return { nodes, edges };
  }, [mode, scenario.edges, scenario.nodes]);

  return (
    <main className="app-shell">
      <aside className="app-sidebar">
        <h1>Gert Runbook Previsualization</h1>
        <p>
          Speculative visual stereotypes for v2 runbook archetypes. This is a design sandbox,
          not a parser-backed renderer.
        </p>

        <label htmlFor="scenario" className="field-label">
          Scenario
        </label>
        <select
          id="scenario"
          className="scenario-select"
          value={scenarioId}
          onChange={(event) => setScenarioId(event.target.value)}
        >
          {scenarios.map((item) => (
            <option key={item.id} value={item.id}>
              {item.name}
            </option>
          ))}
        </select>

        <label htmlFor="mode" className="field-label">
          Visualization
        </label>
        <select
          id="mode"
          className="scenario-select"
          value={mode}
          onChange={(event) => setMode(event.target.value as VisualizationMode)}
        >
          <option value="flow">{modeLabel.flow}</option>
          <option value="flow-sankey">{modeLabel["flow-sankey"]}</option>
          <option value="sequence">{modeLabel.sequence}</option>
          <option value="stereotype-lanes">{modeLabel["stereotype-lanes"]}</option>
          <option value="governance-risk">{modeLabel["governance-risk"]}</option>
          <option value="radial-constellation">{modeLabel["radial-constellation"]}</option>
          <option value="critical-path">{modeLabel["critical-path"]}</option>
          <option value="failure-paths">{modeLabel["failure-paths"]}</option>
          <option value="sankey-emphasis">{modeLabel["sankey-emphasis"]}</option>
          <option value="concern-matrix">{modeLabel["concern-matrix"]}</option>
        </select>

        <div className="scenario-meta">
          <div>
            <strong>Archetype:</strong> {scenario.archetype}
          </div>
          <div>{scenario.summary}</div>
          <div className="mode-summary">{modeSummary[mode]}</div>
        </div>

        <h2>Legend</h2>
        <ul className="legend">
          <li><span className="swatch swatch--trigger" /> Trigger</li>
          <li><span className="swatch swatch--human" /> Human Input / Approval</li>
          <li><span className="swatch swatch--decision" /> Decision Gate</li>
          <li><span className="swatch swatch--control" /> Control Flow</li>
          <li><span className="swatch swatch--automation" /> Automation Step</li>
          <li><span className="swatch swatch--compensation" /> Compensation / Rollback</li>
          <li><span className="swatch swatch--terminal" /> Terminal</li>
        </ul>
      </aside>

      <section className="canvas-wrap">
        <ReactFlow
          fitView
          nodes={rendered.nodes}
          edges={rendered.edges}
          nodeTypes={nodeTypes}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable
          panOnDrag
          zoomOnScroll
        >
          <Background variant={BackgroundVariant.Dots} gap={18} size={1.2} />
          <MiniMap pannable zoomable />
          <Controls />
          <Panel position="top-right" className="panel-note">
            Viewing mode: {modeLabel[mode]}. Compare the same scenario with different spatial grammars.
          </Panel>
        </ReactFlow>
      </section>
    </main>
  );
}
