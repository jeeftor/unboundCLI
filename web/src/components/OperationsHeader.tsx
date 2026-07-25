export function OperationsHeader({
  loading,
  message,
  messageKind,
  summary
}: {
  loading: boolean;
  message: string;
  messageKind: 'info' | 'error' | 'ok';
  summary: { entries: number; inSync: number; out: number; caddyOnly: number; stale: number; cloudflare: number; issues: number };
}) {
  const totalSignals = Math.max(1, summary.entries + summary.cloudflare + summary.out + summary.stale);
  const progress = loading ? 72 : 100;
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
        aria-valuenow={progress}
        hidden={!loading}
      >
        <div className="loading-copy">
          <span id="top-progress-title">Loading service status...</span>
          <strong>{progress}%</strong>
        </div>
        <div className="progress-track"><span style={{ width: `${Math.min(100, Math.round((summary.entries / totalSignals) * 100) || progress)}%` }} /></div>
        <small id="top-progress-detail">Scanning Caddy routes and DNS services...</small>
      </div>
    </section>
  );
}
