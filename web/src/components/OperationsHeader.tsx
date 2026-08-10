import { SERVICE_META } from '../store';
import type { ProgressEvent } from '../types';

const SERVICE_ORDER = ['caddy', 'unbound', 'adguard', 'dhcp', 'cloudflare', 'dns'];

export function OperationsHeader({
  loading,
  message,
  messageKind,
  progress,
  summary: _summary,
}: {
  loading: boolean;
  message: string;
  messageKind: 'info' | 'error' | 'ok';
  progress: Record<string, ProgressEvent>;
  summary: { entries: number; inSync: number; out: number; caddyOnly: number; stale: number; cloudflare: number; issues: number };
}) {
  const progressPct = loading ? 72 : 100;

  // Count how many services have completed (loaded/failed/skipped).
  const completed = SERVICE_ORDER.filter(s => {
    const ev = progress[s];
    return ev && ev.status !== 'pending';
  }).length;
  const totalServices = SERVICE_ORDER.length;

  return (
    <section className="operations-header">
      <div>
        <h2>DNS Operations</h2>
        <p>Monitor DNS health, review entries, and apply server-issued sync plans.</p>
      </div>
      <div className={`message ${messageKind}`} id="message" aria-live="polite">
        {message}
      </div>
      <div
        id="top-progress"
        className="loading-panel"
        role="progressbar"
        aria-live="polite"
        aria-label="Loading service status"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={progressPct}
        hidden={!loading}
      >
        <div className="loading-copy">
          <span id="top-progress-title">Loading service status...</span>
          <strong>{completed}/{totalServices}</strong>
        </div>
        <div className="progress-track"><span style={{ width: `${Math.min(100, Math.round((completed / totalServices) * 100))}%` }} /></div>
        <div className="service-progress-chips">
          {SERVICE_ORDER.map(svc => {
            const ev = progress[svc];
            const meta = SERVICE_META[svc];
            if (!meta) return null;
            const status = ev?.status || 'pending';
            const cls = `service-chip ${status}`;
            const label = ev?.status === 'loaded'
              ? `${meta.label} ${ev.count}`
              : ev?.status === 'failed'
              ? `${meta.label} ✗`
              : ev?.status === 'skipped'
              ? `${meta.label} —`
              : meta.label;
            return (
              <span key={svc} className={cls} title={ev?.error || status}>
                <span className="service-chip-icon">{meta.icon}</span>
                {label}
              </span>
            );
          })}
        </div>
      </div>
    </section>
  );
}
