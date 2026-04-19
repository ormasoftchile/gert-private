import type { Edge, Node } from "@xyflow/react";
import type { FlowScenario, RunbookNodeData } from "../types";
import { layoutNodes } from "./layout";

function n(
  id: string,
  title: string,
  subtitle: string,
  stereotype: RunbookNodeData["stereotype"],
  notes?: string,
): Node<RunbookNodeData> {
  return {
    id,
    type: "runbook",
    position: { x: 0, y: 0 },
    data: { title, subtitle, stereotype, notes },
  };
}

function e(
  id: string,
  source: string,
  target: string,
  label?: string,
  animated = false,
): Edge {
  return {
    id,
    source,
    target,
    label,
    animated,
    type: "smoothstep",
    markerEnd: { type: "arrowclosed" },
  };
}

function buildScenario(base: Omit<FlowScenario, "nodes"> & { nodes: Node<RunbookNodeData>[] }): FlowScenario {
  return {
    ...base,
    nodes: layoutNodes(base.nodes, base.edges, "TB"),
  };
}

const k8sIncident = buildScenario({
  id: "k8s-incident",
  name: "K8s Incident Response",
  archetype: "mitigation",
  summary:
    "Wait-for-event trigger, parallel diagnostics, conditional remediation branches, and explicit approval gates.",
  nodes: [
    n("alert", "Wait for Alert", "type: wait_for_event", "trigger", "prometheus.alert.firing"),
    n("diag", "Gather Diagnostics", "type: parallel", "control", "manifest + describe + logs"),
    n("analyze", "Analyze Failure", "type: branch", "control"),
    n("approval", "SRE Approval", "type: decision", "decision", "approve / reject / escalate"),
    n("remediate", "Apply Fix", "type: cli", "automation", "kubectl apply/patch"),
    n("verify", "Verify Recovery", "type: cli + retry", "automation"),
    n("notify", "Notify Stakeholders", "type: tool", "automation", "slack + pagerduty"),
    n("done", "End", "type: end", "terminal"),
  ],
  edges: [
    e("e1", "alert", "diag"),
    e("e2", "diag", "analyze"),
    e("e3", "analyze", "approval", "config/oom branch"),
    e("e4", "approval", "remediate", "approved"),
    e("e5", "remediate", "verify"),
    e("e6", "verify", "notify"),
    e("e7", "notify", "done"),
  ],
});

const onboarding = buildScenario({
  id: "employee-onboarding",
  name: "Employee Onboarding",
  archetype: "reference",
  summary:
    "Human-heavy flow with explicit decision outcomes and broad parallel provisioning branches.",
  nodes: [
    n("collect", "Collect Employee Data", "type: collector", "human"),
    n("mgr", "Manager Approval", "type: decision", "decision", "approve / reject / escalate"),
    n("mgr_reject", "Stop: Rejected", "type: end", "terminal", "manager rejected request"),
    n("mgr_escalate", "Escalate to HR", "type: end", "terminal", "manager timeout/escalation"),
    n("bg", "Background Check", "type: collector", "human"),
    n("provision", "Provision Resources", "type: parallel", "control", "okta/github/slack/laptop/it-call"),
    n("okta", "Create Okta User", "type: tool", "automation"),
    n("gh", "Invite GitHub", "type: tool", "automation"),
    n("slack", "Invite Slack", "type: tool", "automation"),
    n("laptop", "Order Laptop", "type: collector", "human"),
    n("join", "Join", "parallel join", "control"),
    n("close", "Close Ticket", "type: tool/cli", "automation"),
    n("end", "End", "type: end", "terminal"),
  ],
  edges: [
    e("o1", "collect", "mgr"),
    e("o2", "mgr", "bg", "approved"),
    e("o2b", "mgr", "mgr_reject", "rejected"),
    e("o2c", "mgr", "mgr_escalate", "escalated"),
    e("o3", "bg", "provision"),
    e("o4", "provision", "okta", "branch"),
    e("o5", "provision", "gh", "branch"),
    e("o6", "provision", "slack", "branch"),
    e("o7", "provision", "laptop", "branch"),
    e("o8", "okta", "join"),
    e("o9", "gh", "join"),
    e("o10", "slack", "join"),
    e("o11", "laptop", "join"),
    e("o12", "join", "close"),
    e("o13", "close", "end"),
  ],
});

const canaryDeploy = buildScenario({
  id: "canary-deploy",
  name: "Canary Deploy + Compensation",
  archetype: "mitigation",
  summary:
    "Strong control-flow specimen: assertions, compensation registration, parallel deploy, iterate monitor loop, and promote/rollback branch.",
  nodes: [
    n("assert", "Pre-Deploy Assertions", "type: assert", "control"),
    n("approve", "Release Approval", "type: decision", "decision", "approve / reject / rollback"),
    n("comp", "Register Rollback", "type: compensate", "compensation"),
    n("deploy", "Deploy Canary", "type: parallel", "control"),
    n("wait", "Wait Healthy", "type: cli + retry", "automation"),
    n("monitor", "Monitor Metrics Loop", "type: iterate", "control", "prometheus every 10s"),
    n("decision", "Promote or Rollback", "type: branch", "control"),
    n("promote", "Promote to 100%", "type: cli/tool", "automation"),
    n("rollback", "Execute Compensation", "on failure", "compensation"),
    n("end", "End", "type: end", "terminal"),
  ],
  edges: [
    e("c1", "assert", "approve"),
    e("c2", "approve", "comp"),
    e("c3", "comp", "deploy"),
    e("c4", "deploy", "wait"),
    e("c5", "wait", "monitor"),
    e("c6", "monitor", "monitor", "next iteration", true),
    e("c7", "monitor", "decision"),
    e("c8", "decision", "promote", "healthy"),
    e("c9", "decision", "rollback", "unhealthy"),
    e("c10", "promote", "end"),
    e("c11", "rollback", "end"),
  ],
});

const compliance = buildScenario({
  id: "soc2-evidence",
  name: "SOC2 Evidence Collection",
  archetype: "reference",
  summary:
    "Inclusion-heavy compliance archetype with repeated include patterns, attestations, and final packaging/upload path.",
  nodes: [
    n("init", "Initialize Audit Package", "type: cli", "automation"),
    n("include1", "Collect CC1.x", "type: include", "automation"),
    n("include2", "Collect CC3/4/5", "type: include", "automation"),
    n("include3", "Collect CC6 Bundle", "type: include", "automation"),
    n("manual", "Manual Evidence", "type: collector", "human", "risk assessment upload"),
    n("aggregate", "Aggregate Evidence", "type: cli", "automation"),
    n("report", "Generate Report", "type: extension", "automation"),
    n("officer1", "Compliance Attestation", "type: decision", "decision", "attest / reject / escalate"),
    n("officer2", "Security Attestation", "type: decision", "decision", "attest / reject / escalate"),
    n("vault", "Upload to Vault", "type: cli", "automation"),
    n("end", "End", "type: end", "terminal"),
  ],
  edges: [
    e("s1", "init", "include1"),
    e("s2", "include1", "include2"),
    e("s3", "include2", "include3"),
    e("s4", "include3", "manual"),
    e("s5", "manual", "aggregate"),
    e("s6", "aggregate", "report"),
    e("s7", "report", "officer1"),
    e("s8", "officer1", "officer2"),
    e("s9", "officer2", "vault"),
    e("s10", "vault", "end"),
  ],
});

export const scenarios: FlowScenario[] = [k8sIncident, onboarding, canaryDeploy, compliance];
