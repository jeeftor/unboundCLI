import {
  AlertTriangle,
  Cloud,
  CheckCircle2,
  Globe,
  HelpCircle,
  Loader2,
  LockKeyhole,
  Network,
  RefreshCw,
  Route,
  ShieldX,
  Stethoscope,
  Trash2,
  Wrench,
} from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import type {
  DiagnosticCategory,
  DiagnosticIssue,
  DiagnosticSeverity,
  DiagnosticsResponse,
  PruneAction,
  PruneResponse,
} from '../types';

// ─── Category metadata ──────────────────────────────────────────────────────

const CATEGORY_META: Record<DiagnosticCategory, { label: string; icon: typeof Network }> = {
  dns: { label: 'DNS', icon: Globe },
  cloudflare: { label: 'Cloudflare', icon: Cloud },
  sync: { label: 'Sync', icon: RefreshCw },
  hostname: { label: 'Hostname', icon: Network },
  auth: { label: 'Auth', icon: LockKeyhole },
};

const SEVERITY_META: Record<DiagnosticSeverity, { label: string; icon: typeof AlertTriangle; cls: string }> = {
  critical: { label: 'Critical', icon: ShieldX, cls: 'critical' },
  warning: { label: 'Warning', icon: AlertTriangle, cls: 'warning' },
  info: { label: 'Info', icon: HelpCircle, cls: 'info' },
};

// ─── Main component ─────────────────────────────────────────────────────────

export function DiagnosticsTab() {
  const [data, setData] = useState<DiagnosticsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterSeverity, setFilterSeverity] = useState<DiagnosticSeverity | 'all'>('all');
  const [filterCategory, setFilterCategory] = useState<DiagnosticCategory | 'all'>('all');
  const [prunePreview, setPrunePreview] = useState<PruneResponse | null>(null);
  const [pruneLoading, setPruneLoading] = useState(false);
  const [pruneResult, setPruneResult] = useState<PruneResponse | null>(null);
  const [pruneError, setPruneError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await fetch('/api/diagnostics');
      if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}`);
      }
      const json = await resp.json() as DiagnosticsResponse;
      setData(json);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  const pruneDryRun = useCallback(async (hostname?: string) => {
    setPruneLoading(true);
    setPruneError(null);
    setPruneResult(null);
    try {
      const body = JSON.stringify({ dry_run: true, ...(hostname ? { hostname } : {}) });
      const resp = await fetch('/api/diagnostics/prune', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body,
      });
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      const json = await resp.json() as PruneResponse;
      setPrunePreview(json);
    } catch (e) {
      setPruneError(e instanceof Error ? e.message : String(e));
    } finally {
      setPruneLoading(false);
    }
  }, []);

  const pruneExecute = useCallback(async (hostname?: string) => {
    setPruneLoading(true);
    setPruneError(null);
    try {
      const body = JSON.stringify({ dry_run: false, ...(hostname ? { hostname } : {}) });
      const resp = await fetch('/api/diagnostics/prune', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body,
      });
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      const json = await resp.json() as PruneResponse;
      setPruneResult(json);
      setPrunePreview(null);
      // Reload diagnostics after prune
      void load();
    } catch (e) {
      setPruneError(e instanceof Error ? e.message : String(e));
    } finally {
      setPruneLoading(false);
    }
  }, [load]);

  useEffect(() => {
    void load();
  }, [load]);

  const issues = data?.issues ?? [];
  const filteredIssues = issues.filter(i => {
    if (filterSeverity !== 'all' && i.severity !== filterSeverity) return false;
    if (filterCategory !== 'all' && i.category !== filterCategory) return false;
    return true;
  });

  // Find stale entries (in DNS/CF but not in Caddy) for prune
  const staleHostnames = new Set<string>();
  for (const issue of issues) {
    if (issue.category === 'sync' && issue.title.toLowerCase().includes('stale')) {
      staleHostnames.add(issue.hostname);
    }
  }

  // Group issues by hostname for display
  const byHostname = new Map<string, DiagnosticIssue[]>();
  for (const issue of filteredIssues) {
    const list = byHostname.get(issue.hostname) ?? [];
    list.push(issue);
    byHostname.set(issue.hostname, list);
  }
  const sortedHostnames = Array.from(byHostname.keys()).sort();

  return (
    <main className="dashboard-shell diagnostics-shell">
      <div className="diagnostics-header">
        <div className="diagnostics-header-title">
          <Stethoscope size={20} />
          <h2>Diagnostics</h2>
          {loading && <span className="auth-phase-tag"><Loader2 size={12} className="spin" /> Scanning…</span>}
          {!loading && data && (
            <span className={`auth-phase-tag ${data.issue_count === 0 ? 'done' : 'enriching'}`}>
              {data.issue_count === 0
                ? <><CheckCircle2 size={12} /> All healthy</>
                : <><AlertTriangle size={12} /> {data.issue_count} issue{data.issue_count !== 1 ? 's' : ''}</>
              }
            </span>
          )}
        </div>
        <button type="button" className="btn-sm" onClick={() => void load()} disabled={loading}>
          {loading ? <Loader2 size={14} className="spin" /> : <RefreshCw size={14} />} Refresh
        </button>
      </div>

      {error && (
        <div className="auth-error">
          <ShieldX size={16} />
          <div>
            <strong>Failed to load diagnostics:</strong> {error}
          </div>
        </div>
      )}

      {/* Summary cards */}
      {data && (
        <div className="diagnostics-summary">
          <div className="diagnostics-summary-card">
            <span className="diagnostics-summary-number">{data.total_entries}</span>
            <span className="diagnostics-summary-label">Total Entries</span>
          </div>
          <div className="diagnostics-summary-card healthy">
            <CheckCircle2 size={20} />
            <div>
              <span className="diagnostics-summary-number">{data.healthy_count}</span>
              <span className="diagnostics-summary-label">Healthy</span>
            </div>
          </div>
          <div className={`diagnostics-summary-card ${data.summary.critical > 0 ? 'critical' : ''}`}>
            <ShieldX size={20} />
            <div>
              <span className="diagnostics-summary-number">{data.summary.critical ?? 0}</span>
              <span className="diagnostics-summary-label">Critical</span>
            </div>
          </div>
          <div className={`diagnostics-summary-card ${data.summary.warning > 0 ? 'warning' : ''}`}>
            <AlertTriangle size={20} />
            <div>
              <span className="diagnostics-summary-number">{data.summary.warning ?? 0}</span>
              <span className="diagnostics-summary-label">Warnings</span>
            </div>
          </div>
        </div>
      )}

      {/* Filters */}
      {issues.length > 0 && (
        <div className="diagnostics-filters">
          <div className="diagnostics-filter-group">
            <span className="diagnostics-filter-label">Severity:</span>
            <button
              type="button"
              className={`diagnostics-filter-chip ${filterSeverity === 'all' ? 'active' : ''}`}
              onClick={() => setFilterSeverity('all')}
            >All</button>
            {(['critical', 'warning', 'info'] as DiagnosticSeverity[]).map(sev => {
              const meta = SEVERITY_META[sev];
              const Icon = meta.icon;
              const count = issues.filter(i => i.severity === sev).length;
              return (
                <button
                  key={sev}
                  type="button"
                  className={`diagnostics-filter-chip ${meta.cls} ${filterSeverity === sev ? 'active' : ''}`}
                  onClick={() => setFilterSeverity(sev)}
                >
                  <Icon size={12} /> {meta.label} ({count})
                </button>
              );
            })}
          </div>
          <div className="diagnostics-filter-group">
            <span className="diagnostics-filter-label">Category:</span>
            <button
              type="button"
              className={`diagnostics-filter-chip ${filterCategory === 'all' ? 'active' : ''}`}
              onClick={() => setFilterCategory('all')}
            >All</button>
            {(Object.keys(CATEGORY_META) as DiagnosticCategory[]).map(cat => {
              const meta = CATEGORY_META[cat];
              const Icon = meta.icon;
              const count = issues.filter(i => i.category === cat).length;
              if (count === 0) return null;
              return (
                <button
                  key={cat}
                  type="button"
                  className={`diagnostics-filter-chip ${filterCategory === cat ? 'active' : ''}`}
                  onClick={() => setFilterCategory(cat)}
                >
                  <Icon size={12} /> {meta.label} ({count})
                </button>
              );
            })}
          </div>
        </div>
      )}

      {/* Issues list */}
      {loading ? (
        <div className="auth-loading">
          <Loader2 size={24} className="spin" />
          <span>Running diagnostics…</span>
        </div>
      ) : filteredIssues.length === 0 && data ? (
        <div className="diagnostics-all-clear">
          <CheckCircle2 size={32} />
          <h3>All clear!</h3>
          <p>No issues found across {data.total_entries} entries.</p>
        </div>
      ) : (
        <div className="diagnostics-issues">
          {sortedHostnames.map(hostname => {
            const hostIssues = byHostname.get(hostname)!;
            const hasCritical = hostIssues.some(i => i.severity === 'critical');
            const isStale = staleHostnames.has(hostname);
            return (
              <div key={hostname} className={`diagnostics-host-card ${hasCritical ? 'has-critical' : ''}`}>
                <div className="diagnostics-host-header">
                  <Network size={14} />
                  <span className="diagnostics-hostname">{hostname}</span>
                  <span className="diagnostics-host-count">
                    {hostIssues.length} issue{hostIssues.length !== 1 ? 's' : ''}
                  </span>
                  {isStale && (
                    <button
                      type="button"
                      className="btn-sm prune-btn"
                      onClick={() => void pruneDryRun(hostname)}
                      disabled={pruneLoading}
                      title="Prune this stale entry from DNS/Cloudflare"
                    >
                      {pruneLoading ? <Loader2 size={12} className="spin" /> : <Trash2 size={12} />} Prune
                    </button>
                  )}
                </div>
                <div className="diagnostics-host-issues">
                  {hostIssues.map((issue, i) => {
                    const sev = SEVERITY_META[issue.severity];
                    const cat = CATEGORY_META[issue.category];
                    const SevIcon = sev.icon;
                    const CatIcon = cat.icon;
                    return (
                      <div key={i} className={`diagnostics-issue ${sev.cls}`}>
                        <div className="diagnostics-issue-header">
                          <div className="diagnostics-issue-icon">
                            <SevIcon size={14} />
                          </div>
                          <span className="diagnostics-issue-title">{issue.title}</span>
                          <span className={`diagnostics-issue-cat cat-${issue.category}`}>
                            <CatIcon size={11} /> {cat.label}
                          </span>
                        </div>
                        <p className="diagnostics-issue-detail">{issue.detail}</p>
                        {issue.suggestion && (
                          <div className="diagnostics-issue-suggestion">
                            <Wrench size={12} />
                            <span>{issue.suggestion}</span>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Prune preview / result modal */}
      {(prunePreview || pruneResult || pruneError) && (
        <div className="prune-modal-overlay" onClick={() => { setPrunePreview(null); setPruneResult(null); setPruneError(null); }}>
          <div className="prune-modal" onClick={e => e.stopPropagation()}>
            <div className="prune-modal-header">
              <Trash2 size={18} />
              <h3>{pruneResult ? 'Prune Results' : 'Prune Preview (Dry Run)'}</h3>
              <button type="button" className="btn-sm" onClick={() => { setPrunePreview(null); setPruneResult(null); setPruneError(null); }}>Close</button>
            </div>
            {pruneError && (
              <div className="auth-error">
                <ShieldX size={16} />
                <div><strong>Prune failed:</strong> {pruneError}</div>
              </div>
            )}
            {prunePreview && (
              <>
                <p className="prune-modal-desc">
                  Found <strong>{prunePreview.total}</strong> action{prunePreview.total !== 1 ? 's' : ''} to clean up stale entries.
                  This will remove DNS overrides, CF tunnel routes, and CF DNS records for hostnames not in Caddy.
                </p>
                <div className="prune-actions-list">
                  {prunePreview.actions.map((a, i) => <PruneActionRow key={i} action={a} />)}
                </div>
                {prunePreview.total > 0 && (
                  <div className="prune-modal-footer">
                    <button type="button" className="btn-sm" onClick={() => setPrunePreview(null)}>Cancel</button>
                    <button type="button" className="btn-sm danger" onClick={() => void pruneExecute()} disabled={pruneLoading}>
                      {pruneLoading ? <Loader2 size={14} className="spin" /> : <Trash2 size={14} />} Confirm & Delete
                    </button>
                  </div>
                )}
              </>
            )}
            {pruneResult && (
              <>
                <p className="prune-modal-desc">
                  Completed <strong>{pruneResult.total}</strong> action{pruneResult.total !== 1 ? 's' : ''}.
                </p>
                <div className="prune-actions-list">
                  {pruneResult.actions.map((a, i) => <PruneActionRow key={i} action={a} />)}
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </main>
  );
}

// ─── Prune action row ───────────────────────────────────────────────────────

function PruneActionRow({ action }: { action: PruneAction }) {
  const serviceIcon = action.service === 'unbound' ? <Globe size={12} />
    : action.service === 'adguard' ? <ShieldX size={12} />
    : action.service === 'cloudflare_tunnel' ? <Route size={12} />
    : <Cloud size={12} />;
  return (
    <div className={`prune-action-row ${action.error ? 'error' : action.success ? 'success' : ''}`}>
      <div className="prune-action-service">{serviceIcon} {action.service}</div>
      <div className="prune-action-detail">
        <strong>{action.hostname}</strong> — {action.detail}
        {action.error && <span className="prune-action-error"> ✗ {action.error}</span>}
        {action.success && <span className="prune-action-success"> ✓</span>}
      </div>
    </div>
  );
}
