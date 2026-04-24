import dagre from "dagre";
import { Position, type Edge, type Node } from "@xyflow/react";
import type { RunbookNodeData } from "../types";

const NODE_WIDTH = 260;
const NODE_HEIGHT = 132;

export function layoutNodes(
  nodes: Node<RunbookNodeData>[],
  edges: Edge[],
  direction: "TB" | "LR" = "TB",
): Node<RunbookNodeData>[] {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({
    rankdir: direction,
    ranksep: 120,
    nodesep: 80,
  });

  nodes.forEach((node) => {
    g.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  });

  edges.forEach((edge) => {
    g.setEdge(edge.source, edge.target);
  });

  dagre.layout(g);

  return nodes.map((node) => {
    const pos = g.node(node.id);
    return {
      ...node,
      position: {
        x: pos.x - NODE_WIDTH / 2,
        y: pos.y - NODE_HEIGHT / 2,
      },
      sourcePosition: direction === "TB" ? Position.Bottom : Position.Right,
      targetPosition: direction === "TB" ? Position.Top : Position.Left,
    };
  });
}
