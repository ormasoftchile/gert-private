import { useMemo, useState } from "react";
import {
  ReactFlow,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  Panel,
} from "@xyflow/react";
import type { NodeTypes } from "@xyflow/react";
import { RunbookNode } from "./components/RunbookNode";
import { scenarios } from "./flows/scenarios";

const nodeTypes: NodeTypes = {
  runbook: RunbookNode,
};

export function App() {
  const [scenarioId, setScenarioId] = useState(scenarios[0].id);

  const scenario = useMemo(
    () => scenarios.find((item) => item.id === scenarioId) ?? scenarios[0],
    [scenarioId],
  );

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

        <div className="scenario-meta">
          <div>
            <strong>Archetype:</strong> {scenario.archetype}
          </div>
          <div>{scenario.summary}</div>
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
          nodes={scenario.nodes}
          edges={scenario.edges}
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
            Primitive mix: custom nodes, smoothstep edges, animated loop edges, minimap, controls.
          </Panel>
        </ReactFlow>
      </section>
    </main>
  );
}
