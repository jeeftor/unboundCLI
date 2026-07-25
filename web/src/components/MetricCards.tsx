import {
  CheckCircle2,
  CircleAlert,
  Cloud,
  ListFilter,
  ShieldCheck,
  SlidersHorizontal
} from 'lucide-react';
import type { ReactNode } from 'react';

export function MetricGrid({
  summary,
  statusFilter,
  setStatusFilter,
}: {
  summary: { entries: number; inSync: number; out: number; caddyOnly: number; stale: number; cloudflare: number; issues: number };
  statusFilter: string;
  setStatusFilter: (value: string) => void;
}) {
  function toggle(status: string) {
    setStatusFilter(statusFilter === status ? 'all' : status);
  }
  return (
    <section id="summary" className="metric-grid" aria-live="polite">
      <Metric label="Total entries" value={summary.entries} sublabel="hostnames" icon={<ListFilter size={20} />} tone="neutral" status="all" activeFilter={statusFilter} onFilter={toggle} />
      <Metric label="In sync" value={summary.inSync} sublabel="perfect" icon={<CheckCircle2 size={20} />} tone="ok" status="synced" activeFilter={statusFilter} onFilter={toggle} />
      <Metric label="Caddy only" value={summary.caddyOnly} sublabel="not in DNS" icon={<CircleAlert size={20} />} tone={summary.caddyOnly > 0 ? 'warn' : 'ok'} status="caddy_only" activeFilter={statusFilter} onFilter={toggle} />
      <Metric label="Stale DNS" value={summary.stale} sublabel="needs cleanup" icon={<SlidersHorizontal size={20} />} tone={summary.stale > 0 ? 'bad' : 'ok'} status="stale" activeFilter={statusFilter} onFilter={toggle} />
      <Metric label="Issues" value={summary.issues} sublabel="need review" icon={<ShieldCheck size={20} />} tone={summary.issues > 0 ? 'bad' : 'ok'} status="issues" activeFilter={statusFilter} onFilter={toggle} />
      <Metric label="Cloudflare routed" value={summary.cloudflare} sublabel="via tunnel" icon={<Cloud size={20} />} tone="violet" status="cloudflare" activeFilter={statusFilter} onFilter={toggle} />
    </section>
  );
}

function Metric({
  label, value, sublabel, icon, tone, status, activeFilter, onFilter,
}: {
  label: string; value: number; sublabel: string; icon: ReactNode; tone: string;
  status: string; activeFilter: string; onFilter: (status: string) => void;
}) {
  const isActive = activeFilter === status && status !== 'all';
  return (
    <article
      className={`metric-card ${tone}${isActive ? ' metric-active' : ''}`}
      role="button"
      tabIndex={0}
      title={isActive ? `Clear filter: ${label}` : `Filter: ${label}`}
      onClick={() => onFilter(status)}
      onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && onFilter(status)}
    >
      <div>
        <span>{label}</span>
        <strong>{value}</strong>
        <small>{sublabel}</small>
      </div>
      {icon}
    </article>
  );
}
