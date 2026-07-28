import { Cloud, Globe, Monitor, Server } from 'lucide-react';
import type { ComponentType } from 'react';

// ── Flow diagram primitives (shared between AuthFlowsTab legend and per-entry VisualizeModal) ──

export type FlowNodeVariant = 'wan' | 'cf' | 'caddy' | 'app' | 'dns' | 'upstream';

const NODE_ICONS: Record<FlowNodeVariant, ComponentType<{ size?: number }>> = {
  wan: Globe,
  cf: Cloud,
  caddy: Server,
  app: Monitor,
  dns: Server,
  upstream: Server,
};

export function FlowNode({
  variant,
  label,
  sublabel,
  active = true,
}: {
  variant: FlowNodeVariant;
  label: string;
  sublabel?: string;
  active?: boolean;
}) {
  const Icon = NODE_ICONS[variant];
  return (
    <div className={`auth-flow-node ${variant}${active ? '' : ' inactive'}`}>
      <Icon size={20} />
      <span>{label}</span>
      {sublabel && <span className="auth-flow-node-sub">{sublabel}</span>}
    </div>
  );
}

export function FlowArrow({ label, active = true }: { label?: string; active?: boolean }) {
  return (
    <div className="auth-flow-arrow">
      {label && <span className="auth-flow-label">{label}</span>}
      <span className="auth-flow-line">{active ? '→' : '·'}</span>
    </div>
  );
}

export function FlowRow({ children }: { children: React.ReactNode }) {
  return <div className="auth-flow-row">{children}</div>;
}

export function FlowExplanation({ children }: { children: React.ReactNode }) {
  return <div className="auth-flow-explanation">{children}</div>;
}
