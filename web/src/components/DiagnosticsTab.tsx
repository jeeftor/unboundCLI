import '../styles/DiagnosticsTab.css';
import {
  AlertTriangle,
  Cloud,
  CheckCircle2,
  Globe,
  HelpCircle,
  LockKeyhole,
  Network,
  RefreshCw,
  ShieldX,
  Stethoscope,
  Trash2,
  Wrench,
} from 'lucide-react';
import { LoadingSpinner } from './LoadingSpinner';
import { PruneModal } from './PruneModal';
import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type {
  DiagnosticCategory,
  DiagnosticIssue,
  DiagnosticSeverity,
  DiagnosticsResponse,
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
  const [pruneSelected, setPruneSelected] = useState<Set<string>>(() => new Set());

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Use SSE stream for progress feedback during loading.
      const resp = await fetch('/api/diagnostics/stream', {
        headers: { 'Accept': 'text/event-stream' },
      });
      if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}`);
      }
      const reader = resp.body?.getReader();
      if (!reader) {
        throw new Error('No response body');
      }
      const decoder = new TextDecoder();
      let buffer = '';
      let lastResult: DiagnosticsResponse | null = null;

      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() ?? '';
        let currentEvent = '';
        for (const line of lines) {
          if (line.startsWith('event: ')) {
            currentEvent = line.slice(7).trim();
          } else if (line.startsWith('data: ') && currentEvent) {
            try {
              const payload = JSON.parse(line.slice(6)) as DiagnosticsResponse;
              if (currentEvent === 'done') {
                lastResult = payload;
              } else if (currentEvent === 'error') {
                throw new Error((payload as unknown as { error?: string }).error ?? 'Unknown error');
              }
            } catch (e) {
              if (e instanceof SyntaxError) {
                // partial JSON, skip
              } else {
                throw e;
              }
            }
            currentEvent = '';
          }
        }
      }
      if (lastResult) {
        setData(lastResult);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  const pruneDryRun = useCallback(async (hostname?: string, hostnames?: string[]) => {
    setPruneLoading(true);
    setPruneError(null);
    setPruneResult(null);
    try {
      const payload: { dry_run: boolean; hostname?: string; hostnames?: string[] } = { dry_run: true };
      if (hostnames && hostnames.length > 0) {
        payload.hostnames = hostnames;
      } else if (hostname) {
        payload.hostname = hostname;
      }
      const json = await api.diagnosticsPrune(payload);
      setPrunePreview(json);
      // Pre-select all hostnames in the preview
      const allHosts = new Set(json.actions.map(a => a.hostname));
      setPruneSelected(allHosts);
    } catch (e) {
      setPruneError(e instanceof Error ? e.message : String(e));
    } finally {
      setPruneLoading(false);
    }
  }, []);

  const pruneExecute = useCallback(async (hostnames?: string[]) => {
    setPruneLoading(true);
    setPruneError(null);
    try {
      const payload: { dry_run: boolean; hostnames?: string[] } = { dry_run: false };
      if (hostnames && hostnames.length > 0) {
        payload.hostnames = hostnames;
      }
      const json = await api.diagnosticsPrune(payload);
      setPruneResult(json);
      setPrunePreview(null);
      setPruneSelected(new Set());
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
          {loading && <span className="auth-phase-tag"><LoadingSpinner size={12} /> Scanning…</span>}
          {!loading && data && (
            <span className={`auth-phase-tag ${data.issue_count === 0 ? 'done' : 'enriching'}`}>
              {data.issue_count === 0
                ? <><CheckCircle2 size={12} /> All healthy</>
                : <><AlertTriangle size={12} /> {data.issue_count} issue{data.issue_count !== 1 ? 's' : ''}</>
              }
            </span>
          )}
        </div>
        <div className="diagnostics-header-actions">
          {staleHostnames.size > 0 && (
            <button
              type="button"
              className="btn-sm prune-all-btn"
              onClick={() => void pruneDryRun(undefined, Array.from(staleHostnames))}
              disabled={pruneLoading || loading}
              title="Preview pruning all stale entries (in DNS/Cloudflare but not in Caddy)"
            >
              {pruneLoading ? <LoadingSpinner size={14} /> : <Trash2 size={14} />} Prune All Stale ({staleHostnames.size})
            </button>
          )}
          <button type="button" className="btn-sm" onClick={() => void load()} disabled={loading}>
            {loading ? <LoadingSpinner size={14} /> : <RefreshCw size={14} />} Refresh
          </button>
        </div>
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
          <LoadingSpinner size={24} />
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
                      {pruneLoading ? <LoadingSpinner size={12} /> : <Trash2 size={12} />} Prune
                    </button>
                  )}
                </div>
                <div className="diagnostics-host-issues">
                  {hostIssues.map((issue) => {
                    const sev = SEVERITY_META[issue.severity];
                    const cat = CATEGORY_META[issue.category];
                    const SevIcon = sev.icon;
                    const CatIcon = cat.icon;
                    return (
                      <div key={`${hostname}-${issue.title}`} className={`diagnostics-issue ${sev.cls}`}>
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
        <PruneModal
          prunePreview={prunePreview}
          pruneResult={pruneResult}
          pruneError={pruneError}
          pruneSelected={pruneSelected}
          pruneLoading={pruneLoading}
          onClose={() => { setPrunePreview(null); setPruneResult(null); setPruneError(null); setPruneSelected(new Set()); }}
          onSetSelected={setPruneSelected}
          onExecute={(hostnames) => void pruneExecute(hostnames)}
          onCancel={() => { setPrunePreview(null); setPruneSelected(new Set()); }}
        />
      )}
    </main>
  );
}
