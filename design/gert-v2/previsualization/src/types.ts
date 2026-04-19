import type { Edge, Node } from "@xyflow/react";

export type VisualStereotype =
  | "trigger"
  | "human"
  | "decision"
  | "control"
  | "automation"
  | "compensation"
  | "terminal";

export type RunbookNodeData = {
  title: string;
  subtitle: string;
  stereotype: VisualStereotype;
  notes?: string;
};

export type FlowScenario = {
  id: string;
  name: string;
  archetype: string;
  summary: string;
  nodes: Node<RunbookNodeData>[];
  edges: Edge[];
};
