import '../styles/SyncModal.css';
import {
  ChevronDown,
  ChevronUp,
  CircleAlert,
  Edit3,
  FileCode2,
  Play,
  ShieldCheck,
  Trash2,
  X,
  Zap
} from 'lucide-react';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { api } from '../api/client';
import { getHostnameDecision, suppressionKey } from '../lib/hostnameDecision';
import {
  disabledSyncValues,
  syncOptions
} from '../lib/services';
import { InlineProgress } from './InlineProgress';
import { CloudflareRoutePanel } from './CloudflarePanel';
import { EntryModal } from './CaddyEntryModal';
import { Select } from './Select';
import type { CaddyEntry, Entry, ServiceKey, SyncAction } from '../types';

export function SyncModal({
  open,
  autoSync,
  onClose,
  hostname,
  entry,
  enabledServices,
  caddyServerIP,
  suppressed,
  onToggleSuppress,
  syncService: _syncService,
  setSyncService,
  syncLoading,
  syncProgress,
  syncLog,
  plannedActions: _plannedActions,
  canSyncNow: _canSyncNow,
  mutationEnabled,
  onPreviewFor,
  onDryRun: _onDryRun,
  onSync,
  onRefresh,
  onRemoveEntry,
}: {
  open: boolean;
  autoSync: boolean;
  onClose: () => void;
  hostname: string;
  entry?: Entry;
  enabledServices: Partial<Record<ServiceKey, boolean>>;
  caddyServerIP: string;
  suppressed: Set<string>;
  onToggleSuppress: (key: string) => void;
  syncService: string;
  setSyncService: (value: string) => void;
  syncLoading: boolean;
  syncProgress: { title: string; detail: string };
  syncLog: string;
  plannedActions: SyncAction[];
  canSyncNow: boolean;
  mutationEnabled: boolean;
  onPreviewFor: (service: string, hostname: string) => Promise<boolean>;
  onDryRun: () => Promise<void>;
  onSync: () => Promise<void>;
  onRefresh: () => void;
  onRemoveEntry: (hostname: string, service?: string) => Promise<void>;
}) {
  const isStale = entry?.overall_status === 4;
  const hostnameDecision = entry ? getHostnameDecision(entry, caddyServerIP) : null;
  const [caddyRaw, setCaddyRaw] = useState('');
  const [caddyEntry, setCaddyEntry] = useState<CaddyEntry | null>(null);
  const [caddyTemplates, setCaddyTemplates] = useState<string[]>([]);
  const [caddyDefaultTemplate, setCaddyDefaultTemplate] = useState('default');
  const [caddyRepoPath, setCaddyRepoPath] = useState('');
  const [caddyOpen, setCaddyOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const hasAutoSyncedRef = useRef(false);

  // Track which services have been removed in this modal session so buttons
  // flip immediately without waiting for the next data refresh.
  const [removingService, setRemovingService] = useState<string | null>(null);
  const [localRemoved, setLocalRemoved] = useState<Set<string>>(() => new Set());
  const [confirmSync, setConfirmSync] = useState<{ service: string; apply: () => Promise<void> } | null>(null);

  // Live server log streaming -- poll /api/logs while an operation is in progress.
  const [liveLog, setLiveLog] = useState<string>('');
  const logCursorRef = useRef(0);

  // Reset local remove state and log whenever the modal opens for a new hostname.
  useEffect(() => {
    if (open) {
      // eslint-disable-next-line @eslint-react/set-state-in-effect
      setLocalRemoved(new Set());
      // eslint-disable-next-line @eslint-react/set-state-in-effect
      setRemovingService(null);
      // eslint-disable-next-line @eslint-react/set-state-in-effect
      setConfirmSync(null);
      // eslint-disable-next-line @eslint-react/set-state-in-effect
      setLiveLog('');
      logCursorRef.current = 0;
    }
  }, [open, hostname]);

  const busy = syncLoading || removingService !== null;

  // Poll server logs while busy.
  useEffect(() => {
    if (!busy) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      // eslint-disable-next-line no-unmodified-loop-condition
      while (!cancelled) {
        try {
          const data = await api.logs(logCursorRef.current);
          if (data.lines.length > 0) {
            logCursorRef.current = data.cursor;
            setLiveLog(prev => {
              const newLines = data.lines.map(l => `[${l.level}] ${l.message}`).join('\n');
              return prev ? `${prev  }\n${  newLines}` : newLines;
            });
          }
        } catch { /* ignore */ }
        await new Promise(r => { timer = setTimeout(r, 400); });
      }
    };
    void poll();
    return () => { cancelled = true; if (timer) clearTimeout(timer); };
  }, [busy]);

  const [caddyError, setCaddyError] = useState('');

  useEffect(() => {
    if (!open || !hostname) {
      // eslint-disable-next-line @eslint-react/set-state-in-effect
      setCaddyRaw('');
      // eslint-disable-next-line @eslint-react/set-state-in-effect
      setCaddyEntry(null);
      // eslint-disable-next-line @eslint-react/set-state-in-effect
      setCaddyError('');
      return;
    }
    // eslint-disable-next-line @eslint-react/set-state-in-effect
    setCaddyError('');
    Promise.all([api.caddyEntries(), api.caddyTemplates()]).then(([res, tmpl]) => {
      const found = res.entries.find(e => e.hostname === hostname) ?? null;
      setCaddyRaw(found?.raw ?? '');
      setCaddyEntry(found);
      setCaddyRepoPath(res.editor?.repo_path ?? '');
      setCaddyTemplates(tmpl.templates);
      setCaddyDefaultTemplate(tmpl.default);
    }).catch((e: unknown) => {
      setCaddyError(e instanceof Error ? e.message : 'Failed to load Caddy data');
    });
  }, [open, hostname]);

  useEffect(() => {
    if (open && autoSync && !hasAutoSyncedRef.current) {
      hasAutoSyncedRef.current = true;
      void (async () => {
        const ok = await onPreviewFor('all', hostname);
        if (ok) { await onSync(); onRefresh(); }
      })();
    }
    if (!open) hasAutoSyncedRef.current = false;
  }, [open, autoSync, hostname, onPreviewFor, onSync, onRefresh]);

  if (!open) return null;

  const serviceRows: Array<{ key: string; label: string; status?: { configured: boolean; in_sync: boolean; ip: string } }> = [
    { key: 'unbound', label: 'Unbound DNS', status: entry?.unbound_status },
    { key: 'adguard', label: 'AdGuard Home', status: entry?.adguard_status },
  ].filter(s => enabledServices[s.key as ServiceKey]);

  const handleServiceRemove = async (key: string) => {
    setRemovingService(key);
    try {
      await onRemoveEntry(hostname, key);
      setLocalRemoved(prev => new Set([...prev, key]));
      onRefresh();
    } finally {
      setRemovingService(null);
    }
  };

  const handleRemoveAll = async () => {
    setRemovingService('all');
    try {
      await onRemoveEntry(hostname, 'all');
      onRefresh();
      onClose();
    } finally {
      setRemovingService(null);
    }
  };

  const runServiceSync = async (serviceKey: string) => {
    setSyncService(serviceKey);
    const ok = await onPreviewFor(serviceKey, hostname);
    if (ok) {
      setConfirmSync({
        service: serviceKey,
        apply: async () => { await onSync(); onRefresh(); }
      });
    }
  };

  const runSyncAll = async () => {
    setSyncService('all');
    const ok = await onPreviewFor('all', hostname);
    if (ok) {
      setConfirmSync({
        service: 'all',
        apply: async () => { await onSync(); onRefresh(); }
      });
    }
  };

  const executeConfirmedSync = async () => {
    if (!confirmSync) return;
    const fn = confirmSync.apply;
    setConfirmSync(null);
    await fn();
  };

  // Any service with a real entry that can be removed.
  const canRemoveAll = serviceRows.some(({ key, status }) => !localRemoved.has(key) && status?.configured === true);

  return (
    <div className="modal-overlay" onClick={(e) => { if (e.target === e.currentTarget && !busy) onClose(); }}>
      <div className="modal sync-modal">
        <div className="modal-header">
          <h3><Zap size={15} />{hostname || 'Sync'}</h3>
          <div className="modal-header-actions">
            <button type="button" className="btn-primary" onClick={() => void runSyncAll()} disabled={busy}>
              <Zap size={13} /> Sync all
            </button>
            <button type="button" className="btn-danger" onClick={() => void handleRemoveAll()} disabled={busy || !mutationEnabled || !canRemoveAll}>
              <Trash2 size={13} /> {removingService === 'all' ? 'Removing...' : 'Remove all'}
            </button>
            <button type="button" className="modal-close" onClick={onClose} disabled={busy}><X size={16} /></button>
          </div>
        </div>
        <div className="modal-body sync-modal-body">
          {confirmSync && (
            <div className="sync-confirm-banner" style={{
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
              padding: '10px 14px',
              margin: '0 0 12px 0',
              border: '1px solid var(--border-warn, #e8a838)',
              borderRadius: '8px',
              background: 'var(--bg-warn, rgba(232,168,56,0.08))',
            }}>
              <CircleAlert size={16} style={{ color: 'var(--text-warn, #e8a838)', flexShrink: 0 }} />
              <span style={{ flex: 1, fontSize: '13px' }}>
                Apply sync for <strong>{confirmSync.service === 'all' ? 'all services' : confirmSync.service}</strong>?
                This will make changes to your DNS configuration.
              </span>
              <button type="button" className="btn-primary" onClick={() => void executeConfirmedSync()} disabled={busy}>
                Apply
              </button>
              <button type="button" className="btn-secondary" onClick={() => setConfirmSync(null)} disabled={busy}>
                Cancel
              </button>
            </div>
          )}
          {hostnameDecision && (
            <HostnameDecisionPanel decision={hostnameDecision} hostname={hostname} suppressed={suppressed} onToggleSuppress={onToggleSuppress} />
          )}

          {/* Per-service rows */}
          <div className="service-sync-rows">
            {serviceRows.map(({ key, label, status }) => {
              const isLocallyRemoved = localRemoved.has(key);
              const isRemoving = removingService === key;
              const configured = status?.configured ?? false;
              const inSync = status?.in_sync ?? false;

              // Badge reflects actual service state.
              let statusText = 'Not present';
              let tone = 'missing';
              if (isLocallyRemoved) { statusText = 'Removed'; tone = 'missing'; }
              else if (configured && inSync) { statusText = status?.ip || 'In sync'; tone = 'ok'; }
              else if (configured && !inSync) { statusText = status?.ip || 'Needs update'; tone = 'warn'; }

              // Buttons reflect current state:
              //   configured + not locally removed -> always show Remove
              //   stale -> only Remove (sync makes no sense for stale entries)
              //   in sync -> show "In sync" indicator + Remove
              //   out of sync / missing -> Sync button (+ Remove if configured)
              const removeBtn = configured && !isLocallyRemoved && mutationEnabled ? (
                <button type="button" className="btn-sm btn-danger-sm" onClick={() => void handleServiceRemove(key)} disabled={busy}>
                  <Trash2 size={12} /> {isRemoving ? 'Removing...' : 'Remove'}
                </button>
              ) : null;

              let primaryBtn: ReactNode = null;
              if (!isLocallyRemoved) {
                if (isStale) {
                  // stale: remove only, no sync action
                } else if (configured && inSync) {
                  primaryBtn = (
                    <span className="btn-sm btn-synced" style={{ cursor: 'default' }}>In sync</span>
                  );
                } else {
                  primaryBtn = (
                    <button type="button" className="btn-sm" onClick={() => void runServiceSync(key)} disabled={busy}>
                      <Zap size={12} /> Sync
                    </button>
                  );
                }
              }

              return (
                <div key={key} className="service-sync-row">
                  <div className="service-sync-info">
                    <span className="service-sync-name">{label}</span>
                    <span className={`service-sync-badge ${tone}`}>{statusText}</span>
                  </div>
                  <div className="service-sync-actions">
                    {primaryBtn}
                    {removeBtn}
                  </div>
                </div>
              );
            })}
            {serviceRows.length === 0 && (
              <div className="service-sync-empty">No DNS services configured</div>
            )}
          </div>

          {entry?.cloudflare_status && enabledServices.cloudflare && (
            <CloudflareRoutePanel
              entry={entry}
              caddyServerIP={caddyServerIP}
              mutationEnabled={mutationEnabled}
              onRefresh={onRefresh}
            />
          )}

          <InlineProgress loading={busy} title={syncProgress.title} detail={syncProgress.detail} />

          {(liveLog || syncLog) && (
            <div className="sync-modal-log">
              <div className="sync-modal-log-label">Log</div>
              <pre>{liveLog || syncLog}</pre>
            </div>
          )}

          {caddyError && (
            <div className="sync-modal-error" role="alert">{caddyError}</div>
          )}

          {(caddyRaw || caddyEntry) && (
            <div className="sync-modal-caddy">
              <div className="sync-caddy-header">
                <button type="button" className="sync-caddy-toggle" onClick={() => setCaddyOpen(o => !o)}>
                  <FileCode2 size={13} /> Caddy config
                  {caddyOpen ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                </button>
                {mutationEnabled && caddyEntry && (
                  <button type="button" className="btn-sm" onClick={() => setEditOpen(true)}>
                    <Edit3 size={12} /> Edit
                  </button>
                )}
              </div>
              {caddyOpen && <pre>{caddyRaw}</pre>}
            </div>
          )}

          {editOpen && (
            <EntryModal
              entry={caddyEntry}
              templates={caddyTemplates}
              defaultTemplate={caddyDefaultTemplate}
              repoPath={caddyRepoPath}
              onClose={() => setEditOpen(false)}
              onSaved={() => {
                setEditOpen(false);
                // Refresh caddy raw + entry
                api.caddyEntries().then(res => {
                  const found = res.entries.find(e => e.hostname === hostname) ?? null;
                  setCaddyRaw(found?.raw ?? '');
                  setCaddyEntry(found);
                }).catch((e: unknown) => {
                  setCaddyError(e instanceof Error ? e.message : 'Failed to refresh Caddy data');
                });
                onRefresh();
              }}
            />
          )}
        </div>
        {!mutationEnabled && (
          <div className="modal-footer">
            <span className="muted" style={{ fontSize: 12 }}>Read-only session -- sync actions are disabled</span>
          </div>
        )}
      </div>
    </div>
  );
}

function DecisionBlock({ kind, severity, title, summary, facts, actions, suppressKey, isSuppressed, onToggle }: {
  kind: string; severity: string; title: string; summary: string;
  facts: string[]; actions: { label: string; description: string }[];
  suppressKey?: string; isSuppressed?: boolean; onToggle?: () => void;
}) {
  return (
    <section className={`hostname-decision ${severity}${isSuppressed ? ' suppressed' : ''}`}>
      <div className="hostname-decision-header">
        {severity === 'warning' ? <CircleAlert size={15} /> : <ShieldCheck size={15} />}
        <div>
          <strong>{title}</strong>
          <span>{summary}</span>
        </div>
        {suppressKey && onToggle && (
          <button
            type="button"
            className={`suppress-toggle-btn${isSuppressed ? ' active' : ''}`}
            title={isSuppressed ? 'Restore warning' : 'Mark as intentional (suppress for this session)'}
            onClick={onToggle}
          >
            {isSuppressed ? 'Restore' : 'Mark intentional'}
          </button>
        )}
      </div>
      {!isSuppressed && (
        <>
          <div className="hostname-decision-facts">
            {facts.map((fact) => <span key={fact}>{fact}</span>)}
          </div>
          {(kind === 'collision' || kind === 'mismatch') && (
            <span className="hostname-decision-note">This is a warning only -- pick a path when you next change DNS or Caddy config.</span>
          )}
          <div className="hostname-decision-actions">
            {actions.filter(a => a.label !== 'Mark intentional').map((action) => (
              <div key={action.label} className="hostname-decision-action">
                <strong>{action.label}</strong>
                <span>{action.description}</span>
              </div>
            ))}
          </div>
        </>
      )}
      {isSuppressed && (
        <span className="hostname-decision-note suppressed-note">Warning suppressed for this session. Click Restore to show it again.</span>
      )}
    </section>
  );
}

function HostnameDecisionPanel({ decision, hostname, suppressed, onToggleSuppress }: {
  decision: ReturnType<typeof getHostnameDecision>;
  hostname: string;
  suppressed: Set<string>;
  onToggleSuppress: (key: string) => void;
}) {
  const primaryKey = suppressionKey(hostname, decision.kind);
  return (
    <>
      <DecisionBlock
        {...decision}
        suppressKey={(decision.kind === 'collision' || decision.kind === 'mismatch') ? primaryKey : undefined}
        isSuppressed={suppressed.has(primaryKey)}
        onToggle={() => onToggleSuppress(primaryKey)}
      />
      {decision.warnings.map((w) => {
        const wKey = suppressionKey(hostname, w.kind);
        return (
          <DecisionBlock key={w.kind} {...w} severity="warning"
            suppressKey={wKey}
            isSuppressed={suppressed.has(wKey)}
            onToggle={() => onToggleSuppress(wKey)}
          />
        );
      })}
    </>
  );
}

export function SyncPanel({
  enabledServices,
  caddyServerIP: _caddyServerIP,
  syncService,
  setSyncService,
  syncLoading,
  syncProgress,
  syncLog,
  plannedActions,
  canSyncNow,
  mutationEnabled,
  onPreview,
  onDryRun,
  onSync
}: {
  enabledServices: Partial<Record<ServiceKey, boolean>>;
  caddyServerIP: string;
  syncService: string;
  setSyncService: (value: string) => void;
  syncLoading: boolean;
  syncProgress: { title: string; detail: string };
  syncLog: string;
  plannedActions: SyncAction[];
  canSyncNow: boolean;
  mutationEnabled: boolean;
  onPreview: () => void;
  onDryRun: () => void;
  onSync: () => void;
}) {
  return (
    <section id="sync-panel" className="panel sync-panel">
      <div className="panel-title">
        <div>
          <strong>Sync Plan</strong>
          <span>Preview before applying changes.</span>
        </div>
        <span className="plan-count">{plannedActions.length} changes</span>
      </div>
      <label className="field-label">Target</label>
      <Select value={syncService} onChange={setSyncService} ariaLabel="Sync target" options={syncOptions(enabledServices)} disabledValues={disabledSyncValues(enabledServices)} idForAll="sync-service-all" />
      <div className="sync-pipeline">
        <button id="preview-sync" type="button" onClick={onPreview} disabled={syncLoading}>
          <Play size={17} />
          <span><strong>Preview sync</strong><small>Preview actions and changes</small></span>
        </button>
        <button id="dry-run-sync" type="button" onClick={onDryRun} disabled={syncLoading || !plannedActions.length}>
          <ShieldCheck size={17} />
          <span><strong>Dry-run sync</strong><small>Simulate without applying</small></span>
        </button>
        <button id="sync-now" type="button" onClick={onSync} disabled={syncLoading || !canSyncNow} title={mutationEnabled ? 'Apply the selected server-issued sync plan' : 'Real sync is unavailable for this web session'}>
          <Zap size={17} />
          <span><strong>Sync now</strong><small>Apply server-issued plan</small></span>
        </button>
      </div>
      <InlineProgress loading={syncLoading} title={syncProgress.title} detail={syncProgress.detail} />
      <div className="log-header"><strong>Plan log</strong><button type="button" disabled>Clear</button></div>
      <div id="sync-log" className="log" role="status" aria-live="polite">{syncLog}</div>
    </section>
  );
}
