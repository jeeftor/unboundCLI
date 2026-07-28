import './ReactFlowDiagram.css';

import {
  ArrowLeftRight,
  Cloud,
  Globe,
  LockKeyhole,
  Monitor,
  Server,
  ShieldCheck,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { Handle, Position } from '@xyflow/react';

// ── Node type definitions ──

export type FlowNodeType =
  | 'wan' | 'cf' | 'cf_access' | 'caddy' | 'authentik' | 'app_auth'
  | 'upstream' | 'app' | 'dns';

type NodeConfig = {
  icon: LucideIcon;
  color: string;
  bg: string;
  border: string;
};

const NODE_CONFIG: Record<FlowNodeType, NodeConfig> = {
  wan:        { icon: Globe,         color: '#7aa2f7', bg: 'rgba(122,162,247,.06)', border: 'rgba(122,162,247,.3)' },
  cf:         { icon: Cloud,         color: '#f09f3c', bg: 'rgba(240,159,60,.06)',  border: 'rgba(240,159,60,.3)' },
  cf_access:  { icon: ShieldCheck,   color: '#f09f3c', bg: 'rgba(240,159,60,.06)',  border: 'rgba(240,159,60,.3)' },
  caddy:      { icon: Server,        color: '#7dcfff', bg: 'rgba(125,207,255,.06)', border: 'rgba(125,207,255,.3)' },
  authentik:  { icon: ArrowLeftRight,color: '#4fa8ff', bg: 'rgba(79,168,255,.06)',  border: 'rgba(79,168,255,.3)' },
  app_auth:   { icon: LockKeyhole,   color: '#3fb971', bg: 'rgba(63,185,113,.06)',  border: 'rgba(63,185,113,.3)' },
  upstream:   { icon: Server,        color: '#3fb971', bg: 'rgba(63,185,113,.06)',  border: 'rgba(63,185,113,.3)' },
  app:        { icon: Monitor,       color: '#7aa2f7', bg: 'rgba(122,162,247,.06)', border: 'rgba(122,162,247,.3)' },
  dns:        { icon: Server,        color: '#7aa2f7', bg: 'rgba(122,162,247,.06)', border: 'rgba(122,162,247,.3)' },
};

// ── Custom node component ──

export type FlowNodeData = {
  nodeType: FlowNodeType;
  label: string;
  sublabel?: string;
  inactive?: boolean;
  warn?: boolean;
};

function FlowNodeComponent({ data }: { data: FlowNodeData }) {
  const cfg = NODE_CONFIG[data.nodeType];
  const Icon = cfg.icon;

  return (
    <div
      className={`rf-node ${data.inactive ? 'rf-node-inactive' : ''} ${data.warn ? 'rf-node-warn' : ''}`}
      style={{
        borderColor: cfg.border,
        background: cfg.bg,
      }}
    >
      <Handle type="target" position={Position.Left} style={{ opacity: 0 }} />
      <div className="rf-node-icon" style={{ color: cfg.color }}>
        <Icon size={16} />
      </div>
      <div className="rf-node-text">
        <div className="rf-node-label">{data.label}</div>
        {data.sublabel && <div className="rf-node-sub">{data.sublabel}</div>}
      </div>
      <Handle type="source" position={Position.Right} style={{ opacity: 0 }} />
    </div>
  );
}

export const nodeTypes = { flowNode: FlowNodeComponent };

// ── Helpers to build nodes/edges from step arrays ──

export type DiagramNode = {
  id: string;
  type: 'flowNode';
  data: FlowNodeData;
  position: { x: number; y: number };
};

export type DiagramEdge = {
  id: string;
  source: string;
  target: string;
  label?: string;
  animated?: boolean;
  type?: string;
  className?: string;
};

export type Step = {
  id: string;
  nodeType?: FlowNodeType;
  label?: string;
  sublabel?: string;
  inactive?: boolean;
  warn?: boolean;
  arrowLabel?: string;
  arrowWarn?: boolean;
};

export function stepsToNodesEdges(steps: Step[]): { nodes: DiagramNode[]; edges: DiagramEdge[] } {
  const nodes: DiagramNode[] = [];
  const edges: DiagramEdge[] = [];

  let prevNodeId: string | null = null;

  for (let i = 0; i < steps.length; i++) {
    const step = steps[i];

    if (step.nodeType && step.label) {
      nodes.push({
        id: step.id,
        type: 'flowNode',
        data: {
          nodeType: step.nodeType,
          label: step.label,
          sublabel: step.sublabel,
          inactive: step.inactive,
          warn: step.warn,
        },
        position: { x: 0, y: 0 },
      });

      if (prevNodeId) {
        edges.push({
          id: `e-${prevNodeId}-${step.id}`,
          source: prevNodeId,
          target: step.id,
          label: step.arrowLabel || undefined,
          animated: step.arrowWarn,
          className: step.arrowWarn ? 'rf-edge-warn' : undefined,
        });
      }

      prevNodeId = step.id;
    }
  }

  return { nodes, edges };
}
