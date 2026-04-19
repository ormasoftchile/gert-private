import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import type { RunbookNodeData } from "../types";

type RunbookNodeType = Node<RunbookNodeData, "runbook">;

const stereotypeLabel: Record<RunbookNodeData["stereotype"], string> = {
  trigger: "Trigger",
  human: "Human",
  decision: "Decision",
  control: "Control",
  automation: "Automation",
  compensation: "Compensation",
  terminal: "Terminal",
};

export function RunbookNode({ data, selected }: NodeProps<RunbookNodeType>) {
  return (
    <div
      className={[
        "rb-node",
        `rb-node--${data.stereotype}`,
        selected ? "rb-node--selected" : "",
      ].join(" ")}
    >
      <Handle type="target" position={Position.Top} className="rb-handle" />

      <div className="rb-node__head">
        <span className="rb-node__stereotype">{stereotypeLabel[data.stereotype]}</span>
      </div>

      <div className="rb-node__title">{data.title}</div>
      <div className="rb-node__subtitle">{data.subtitle}</div>
      {data.notes ? <div className="rb-node__notes">{data.notes}</div> : null}

      <Handle type="source" position={Position.Bottom} className="rb-handle" />
    </div>
  );
}
