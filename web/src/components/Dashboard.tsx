import {
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  CircleAlert,
  Cloud,
  Edit3,
  FileCode2,
  FileSliders,
  Gauge,
  GitBranch,
  ListFilter,
  Loader2,
  Play,
  RefreshCw,
  Search,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Terminal,
  TerminalSquare,
  Trash2,
  WifiOff,
  X,
  Zap
} from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import type { Dispatch, KeyboardEvent, MouseEvent, ReactNode, SetStateAction } from 'react';
import { api } from '../api/client';
import { CaddyEditor, EntryModal } from './CaddyEditor';
import type { CaddyEditorForm, ConfigForms, TestResults } from '../hooks/useConfigForms';
import {
  cloudflareStateText,
  compactSourceKind,
  disabledSyncValues,
  dnsResultClass,
  formatConfigKey,
  serviceFilterOptions,
  serviceMeta,
  serviceOrder,
  serviceStateText,
  sourceLabel,
  statusClassByCode,
  statusFilterOptions,
  syncOptions
} from '../lib/services';
import { getHostnameDecision } from '../lib/hostnameDecision';
import type { ConfigResponse, ConfigServiceSummary, EntriesResponse, Entry, ServiceKey, SyncAction } from '../types';

type AppView = 'dashboard' | 'caddy-editor';

export function AppShell({
  view,
  setView,
  config,
  loading,
  message,
  messageKind,
  report,
  summary,
  statusFilter,
  setStatusFilter,
  serviceFilter,
  setServiceFilter,
  search,
  setSearch,
  entries,
  selectedEntry,
  selectedHostname,
  setSelectedHostname,
  mutationEnabled,
  syncService,
  setSyncService,
  syncLoading,
  syncProgress,
  syncLog,
  plannedActions,
  canSyncNow,
  onRefresh,
  onPreview,
  onDryRun,
  onSync,
  onRemoveEntry,
  onSyncAll,
  configOpen,
  setConfigOpen,
  forms,
  setForms,
  savedForms,
  configStatus,
  configStatusKind,
  testResults,
  onSaveConfig,
  onSaveCaddyEditor,
  onTestConfig
}: {
  view: AppView;
  setView: (v: AppView) => void;
  config: ConfigResponse | null;
  loading: boolean;
  message: string;
  messageKind: 'info' | 'error' | 'ok';
  report: EntriesResponse['report'];
  summary: { entries: number; inSync: number; out: number; caddyOnly: number; stale: number; cloudflare: number; issues: number };
  statusFilter: string;
  setStatusFilter: (value: string) => void;
  serviceFilter: string;
  setServiceFilter: (value: string) => void;
  search: string;
  setSearch: (value: string) => void;
  entries: Entry[];
  selectedEntry?: Entry;
  selectedHostname: string;
  setSelectedHostname: (value: string) => void;
  mutationEnabled: boolean;
  syncService: string;
  setSyncService: (value: string) => void;
  syncLoading: boolean;
  syncProgress: { title: string; detail: string };
  syncLog: string;
  plannedActions: SyncAction[];
  canSyncNow: boolean;
  onRefresh: () => void;
  onPreview: (service?: string, hostname?: string) => Promise<boolean>;
  onDryRun: () => Promise<void>;
  onSync: () => Promise<void>;
  onRemoveEntry: (hostname: string, service?: string) => Promise<void>;
  onSyncAll: () => Promise<void>;
  configOpen: boolean;
  setConfigOpen: (value: boolean) => void;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  savedForms: ConfigForms;
  configStatus: string;
  configStatusKind: 'info' | 'error' | 'ok';
  testResults: TestResults;
  onSaveConfig: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onSaveCaddyEditor: () => Promise<void>;
  onTestConfig: (service: ServiceKey) => Promise<void>;
}) {
  const enabledServices = config?.enabled || {};
  const [modalOpen, setModalOpen] = useState(false);
  const [modalHostname, setModalHostname] = useState('');
  const [modalAutoSync, setModalAutoSync] = useState(false);
  const [logBarOpen, setLogBarOpen] = useState(false);

  // Global remote-ahead status — polled every 60s regardless of active tab.
  type GitRemoteStatus = { remote_ahead: number; local_ahead: number; branch: string; remote: string; fetch_error?: string };
  const [remoteStatus, setRemoteStatus] = useState<GitRemoteStatus | null>(null);
  const [remoteChecking, setRemoteChecking] = useState(false);
  const [pulling, setPulling] = useState(false);
  const [pullOutput, setPullOutput] = useState('');

  const checkRemote = useCallback(async () => {
    setRemoteChecking(true);
    try { setRemoteStatus(await api.caddyGitStatus()); } catch { /* non-fatal */ } finally { setRemoteChecking(false); }
  }, []);

  const handlePull = useCallback(async () => {
    setPulling(true); setPullOutput('');
    try {
      const res = await api.caddyGitPull();
      setPullOutput(res.output || 'Already up to date.');
      await checkRemote();
      onRefresh();
    } catch (err) { setPullOutput(`Error: ${String(err)}`); }
    finally { setPulling(false); }
  }, [checkRemote, onRefresh]);

  useEffect(() => {
    void checkRemote();
    const id = setInterval(() => { void checkRemote(); }, 60_000);
    return () => clearInterval(id);
  }, [checkRemote]);

  const modalEntry = entries.find(e => e.hostname === modalHostname);

  const openModify = (hostname: string) => {
    setModalHostname(hostname);
    setSelectedHostname(hostname);
    setModalAutoSync(false);
    setModalOpen(true);
  };

  const openQuickSync = (hostname: string) => {
    setModalHostname(hostname);
    setSelectedHostname(hostname);
    setModalAutoSync(true);
    setModalOpen(true);
  };

  // (log bar opens itself when it sees activity)

  const showRemoteBanner = remoteStatus && (remoteStatus.remote_ahead > 0 || remoteStatus.fetch_error);

  return (
    <div className="console-layout">
      <div className="console-main">
        <Topbar config={config} loading={loading} syncLoading={syncLoading} view={view} setView={setView} onRefresh={onRefresh} onOpenConfig={() => setConfigOpen(true)} />

        {showRemoteBanner && (
          <div className={`git-remote-banner global ${remoteStatus!.fetch_error ? 'error' : 'warn'}`}>
            <GitBranch size={14} />
            {remoteStatus!.fetch_error ? (
              <span>Caddy repo: could not reach remote — {remoteStatus!.fetch_error}</span>
            ) : (
              <span>
                Caddy repo: remote <strong>{remoteStatus!.remote}/{remoteStatus!.branch}</strong> is{' '}
                <strong>{remoteStatus!.remote_ahead}</strong> commit{remoteStatus!.remote_ahead !== 1 ? 's' : ''} ahead — pull before deploying
              </span>
            )}
            <div className="git-remote-banner-actions">
              {remoteStatus!.remote_ahead > 0 && mutationEnabled && (
                <button type="button" className="btn-sm btn-warn" onClick={() => void handlePull()} disabled={pulling}>
                  {pulling ? <><Loader2 size={12} className="spin" /> Pulling…</> : '⬇ Pull'}
                </button>
              )}
              <button type="button" className="btn-sm" onClick={() => void checkRemote()} disabled={remoteChecking}>
                {remoteChecking ? <Loader2 size={12} className="spin" /> : <RefreshCw size={12} />}
              </button>
            </div>
          </div>
        )}

        {pullOutput && (
          <div className="git-pull-output global">
            <pre>{pullOutput}</pre>
            <button type="button" className="git-pull-dismiss" onClick={() => setPullOutput('')}><X size={13} /></button>
          </div>
        )}

        <div className="console-scroll">
          {view === 'caddy-editor' ? (
            <main className="dashboard-shell caddy-editor-shell">
              <CaddyEditor mutationEnabled={mutationEnabled} remoteStatus={remoteStatus} remoteChecking={remoteChecking} pulling={pulling} onPull={handlePull} onCheckRemote={checkRemote} />
            </main>
          ) : (
            <main className="dashboard-shell">
              <OperationsHeader
                loading={loading}
                message={message}
                messageKind={messageKind}
                summary={summary}
              />
              <MetricGrid summary={summary} statusFilter={statusFilter} setStatusFilter={setStatusFilter} />
              <EntriesToolbar
                entriesCount={entries.length}
                statusFilter={statusFilter}
                setStatusFilter={setStatusFilter}
                serviceFilter={serviceFilter}
                setServiceFilter={setServiceFilter}
                search={search}
                setSearch={setSearch}
                enabledServices={enabledServices}
                mutationEnabled={mutationEnabled}
                onSyncAll={onSyncAll}
              />
              <CFRepairBanner
                entries={entries}
                mutationEnabled={mutationEnabled}
                cfEnabled={enabledServices.cloudflare === true}
                onRepaired={onRefresh}
              />
              <EntriesTable
                entries={entries}
                selectedHostname={selectedHostname}
                mutationEnabled={mutationEnabled}
                enabledServices={enabledServices}
                caddyServerIP={config?.caddy.server_ip || ''}
                onSelect={setSelectedHostname}
                onQuickSync={openQuickSync}
                onOpenModify={openModify}
                onRemove={onRemoveEntry}
              />
            </main>
          )}
        </div>
        <LogBar open={logBarOpen} onToggle={() => setLogBarOpen(o => !o)} />
      </div>
      <SyncModal
        open={modalOpen}
        autoSync={modalAutoSync}
        onClose={() => setModalOpen(false)}
        hostname={modalHostname}
        entry={modalEntry}
        enabledServices={enabledServices}
        caddyServerIP={config?.caddy.server_ip || ''}
        syncService={syncService}
        setSyncService={setSyncService}
        syncLoading={syncLoading}
        syncProgress={syncProgress}
        syncLog={syncLog}
        plannedActions={plannedActions}
        canSyncNow={canSyncNow}
        mutationEnabled={mutationEnabled}
        onPreviewFor={(service, hostname) => onPreview(service, hostname)}
        onDryRun={onDryRun}
        onSync={onSync}
        onRefresh={onRefresh}
        onRemoveEntry={onRemoveEntry}
      />
      <ConfigModal
        open={configOpen}
        onClose={() => setConfigOpen(false)}
        config={config}
        forms={forms}
        setForms={setForms}
        savedForms={savedForms}
        mutationEnabled={mutationEnabled}
        status={configStatus}
        statusKind={configStatusKind}
        testResults={testResults}
        onSave={onSaveConfig}
        onSaveCaddyEditor={onSaveCaddyEditor}
        onTest={onTestConfig}
      />
    </div>
  );
}

function Sidebar({ config, report, view, setView, onOpenConfig }: { config: ConfigResponse | null; report: EntriesResponse['report']; view: AppView; setView: (v: AppView) => void; onOpenConfig: () => void }) {
  return (
    <aside className="sidebar">
      <div className="brand-lockup">
        <div className="brand-mark"><Zap size={18} /></div>
        <div>
          <h1>Caddy DNS Sync</h1>
          <span>Local dashboard</span>
        </div>
      </div>
      <nav className="nav-stack" aria-label="Primary">
        <span className="nav-section">Overview</span>
        <button type="button" className={`nav-item ${view === 'dashboard' ? 'active' : ''}`} onClick={() => setView('dashboard')}><Gauge size={15} /> Dashboard</button>
        <button type="button" className={`nav-item ${view === 'caddy-editor' ? 'active' : ''}`} onClick={() => setView('caddy-editor')}><FileCode2 size={15} /> Caddy Editor</button>
        <a className="nav-item" href="#sync-panel"><SlidersHorizontal size={15} /> Sync plan</a>
        <a className="nav-item" href="#sync-log"><TerminalSquare size={15} /> Logs</a>
        <span className="nav-section">Services</span>
        {serviceOrder.map((service) => {
          const meta = serviceMeta[service];
          const Icon = meta.icon;
          const enabled = config?.enabled?.[service] !== false;
          const count = report.services?.[service]?.count;
          return (
            <button key={service} type="button" className={`nav-item service ${meta.tone} ${enabled ? '' : 'muted'}`} onClick={onOpenConfig}>
              <Icon size={15} /> {meta.shortLabel}
              <span>{typeof count === 'number' ? count : enabled ? 'ok' : '-'}</span>
            </button>
          );
        })}
        <span className="nav-section">System</span>
        <button type="button" className="nav-item" onClick={onOpenConfig}><Settings size={15} /> Configuration</button>
      </nav>
    </aside>
  );
}

function Topbar({ config, loading, syncLoading, view, setView, onRefresh, onOpenConfig }: { config: ConfigResponse | null; loading: boolean; syncLoading: boolean; view: AppView; setView: (v: AppView) => void; onRefresh: () => void; onOpenConfig: () => void }) {
  const busy = loading || syncLoading;
  const busyLabel = syncLoading ? 'Syncing…' : 'Loading…';

  return (
    <header className="topbar">
      <div className="brand-lockup">
        <div className="brand-mark"><Zap size={18} /></div>
        <div>
          <h1>Caddy DNS Sync</h1>
          <span>{config?.version ? <span className="brand-version">{config.version}</span> : 'Local dashboard'}</span>
        </div>
      </div>
      {busy && (
        <div className="topbar-activity" role="status" aria-live="polite">
          <Loader2 size={14} className="spin" />
          <span>{busyLabel}</span>
        </div>
      )}
      <div className="runtime-card" id="runtime">
        <span>Caddy runtime</span>
        <strong>{config ? `${config.caddy.server_ip}:${config.caddy.server_port}` : 'Loading...'}</strong>
        <em className={config?.enabled?.caddy === false ? 'down' : ''}>{config?.enabled?.caddy === false ? 'Offline' : 'Running'}</em>
      </div>
      <div className="top-actions">
        <button type="button" className={view === 'caddy-editor' ? 'active' : ''} onClick={() => setView(view === 'caddy-editor' ? 'dashboard' : 'caddy-editor')}>
          {view === 'caddy-editor'
            ? <><ChevronDown size={16} style={{ transform: 'rotate(90deg)' }} /> Dashboard</>
            : <><FileCode2 size={16} /> Caddy Editor</>}
        </button>
        <button id="refresh" type="button" onClick={onRefresh} disabled={loading}>
          {loading ? <Loader2 size={16} className="spin" /> : <RefreshCw size={16} />} Refresh
        </button>
        <button id="config-toggle" type="button" onClick={onOpenConfig}>
          <Settings size={16} /> Settings
        </button>
      </div>
    </header>
  );
}

function OperationsHeader({
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

function MetricGrid({
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

function EntriesToolbar({
  entriesCount,
  statusFilter,
  setStatusFilter,
  serviceFilter,
  setServiceFilter,
  search,
  setSearch,
  enabledServices,
  mutationEnabled,
  onSyncAll,
}: {
  entriesCount: number;
  statusFilter: string;
  setStatusFilter: (value: string) => void;
  serviceFilter: string;
  setServiceFilter: (value: string) => void;
  search: string;
  setSearch: (value: string) => void;
  enabledServices: Partial<Record<ServiceKey, boolean>>;
  mutationEnabled: boolean;
  onSyncAll: () => Promise<void>;
}) {
  const disabledServices = new Set(serviceOrder.filter((service) => service !== 'caddy' && enabledServices[service] === false));
  return (
    <section className="entries-toolbar panel">
      <div className="search-box">
        <Search size={15} />
        <input id="search" type="search" aria-label="Search hostnames" placeholder="Search hostnames..." value={search} onChange={(event) => setSearch(event.target.value)} />
      </div>
      <Select value={serviceFilter} onChange={setServiceFilter} ariaLabel="Service filter" options={serviceFilterOptions} disabledValues={disabledServices} />
      <Select value={statusFilter} onChange={setStatusFilter} ariaLabel="Status filter" options={statusFilterOptions} />
      <span className="entry-count">{entriesCount} entries</span>
      {mutationEnabled && (
        <button type="button" className="btn-primary btn-sm toolbar-sync-all" onClick={() => void onSyncAll()}>
          <Zap size={13} /> Sync All
        </button>
      )}
    </section>
  );
}

function AlertBadge({ title, summary, facts, actions, variant }: {
  title: string;
  summary: string;
  facts: string[];
  actions: { label: string; description: string }[];
  variant: 'collision' | 'mismatch';
}) {
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null);
  const ref = useRef<HTMLSpanElement>(null);

  const handleMouseEnter = () => {
    if (ref.current) {
      const r = ref.current.getBoundingClientRect();
      setPos({ top: r.bottom + 6, left: r.left });
    }
  };

  return (
    <span
      ref={ref}
      className={`decision-row-alert alert-badge-wrap${variant === 'mismatch' ? ' mismatch' : ''}`}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={() => setPos(null)}
    >
      <CircleAlert size={11} /> {title}
      {pos && createPortal(
        <div
          className={`alert-tooltip${variant === 'mismatch' ? ' mismatch' : ''}`}
          style={{ position: 'fixed', top: pos.top, left: pos.left, zIndex: 9999 }}
        >
          <strong>{title}</strong>
          <span>{summary}</span>
          {facts.length > 0 && <ul>{facts.map((f) => <li key={f}>{f}</li>)}</ul>}
          {actions.length > 0 && (
            <div className="alert-tooltip-actions">
              {actions.map((a) => <span key={a.label}><em>{a.label}:</em> {a.description}</span>)}
            </div>
          )}
        </div>,
        document.body
      )}
    </span>
  );
}

function EntriesTable({
  entries,
  selectedHostname,
  mutationEnabled,
  enabledServices,
  caddyServerIP,
  onSelect,
  onQuickSync,
  onOpenModify,
  onRemove,
}: {
  entries: Entry[];
  selectedHostname: string;
  mutationEnabled: boolean;
  enabledServices: Partial<Record<ServiceKey, boolean>>;
  caddyServerIP: string;
  onSelect: (hostname: string) => void;
  onQuickSync: (hostname: string) => void;
  onOpenModify: (hostname: string) => void;
  onRemove: (hostname: string, service?: string) => Promise<void>;
}) {
  return (
    <section id="entries-panel" className="panel entries-panel">
      <table>
        <thead>
          <tr>
            <th>Hostname</th><th>Status</th><th>Services</th><th>Caddy upstream</th><th>DNS</th><th>Cloudflare route</th><th>Actions</th>
          </tr>
        </thead>
        <tbody id="entries">
          {entries.map((entry) => (
            <EntryRow
              key={entry.hostname}
              entry={entry}
              selected={entry.hostname === selectedHostname}
              mutationEnabled={mutationEnabled}
              caddyServerIP={caddyServerIP}
              onSelect={onSelect}
              onQuickSync={onQuickSync}
              onOpenModify={onOpenModify}
              onRemove={onRemove}
            />
          ))}
        </tbody>
      </table>
    </section>
  );
}

function EntryRow({
  entry,
  selected,
  mutationEnabled,
  caddyServerIP,
  onSelect,
  onQuickSync,
  onOpenModify,
  onRemove,
}: {
  entry: Entry;
  selected: boolean;
  mutationEnabled: boolean;
  caddyServerIP: string;
  onSelect: (hostname: string) => void;
  onQuickSync: (hostname: string) => void;
  onOpenModify: (hostname: string) => void;
  onRemove: (hostname: string, service?: string) => Promise<void>;
}) {
  const isStale = entry.overall_status === 4;
  const decision = getHostnameDecision(entry, caddyServerIP);
  const dnsOK = dnsResultClass(entry.dns_resolved) === 'ok';
  const statusDetail = entry.overall_status === 3
    ? 'Not in DNS'
    : entry.overall_status === 4
      ? 'Needs cleanup'
      : entry.dns_resolved
        ? 'Resolved'
        : 'Not in DNS';
  const selectRow = () => onSelect(entry.hostname);
  const onRowKeyDown = (event: KeyboardEvent<HTMLTableRowElement>) => {
    if (event.key !== 'Enter' && event.key !== ' ') return;
    event.preventDefault();
    selectRow();
  };
  return (
    <tr data-hostname={entry.hostname} className={selected ? 'selected-row' : ''} onClick={selectRow} onKeyDown={onRowKeyDown} tabIndex={0} aria-selected={selected}>
      <td data-label="Hostname">
        <strong>{entry.hostname}</strong>
        <span className="subtle">{entry.data_source || 'Caddy route'} <i /></span>
        {(decision.kind === 'collision' || decision.kind === 'mismatch') && (
          <AlertBadge title={decision.title} summary={decision.summary} facts={decision.facts} actions={decision.actions} variant="collision" />
        )}
        {decision.warnings.map((w) => (
          <AlertBadge key={w.kind} title={w.title} summary={w.summary} facts={w.facts} actions={w.actions} variant="mismatch" />
        ))}
      </td>
      <td data-label="Status"><StatusChip entry={entry} /><span className="status-subtext">{statusDetail}</span></td>
      <td data-label="Services"><ServiceBadges entry={entry} /></td>
      <td data-label="Caddy upstream"><span>{entry.caddy_upstream || '-'}</span><span className="subtle">admin {entry.caddy_ip || '-'}</span><span className="protocol-pill">HTTP</span></td>
      <td data-label="DNS"><span className={`dns-result ${dnsResultClass(entry.dns_resolved)}`}>{entry.dns_resolved || 'FAIL'}</span><span className="status-subtext">{dnsOK ? 'A record' : 'NXDOMAIN'}</span></td>
      <td data-label="Cloudflare route"><CloudflareDetails status={entry.cloudflare_status} hostname={entry.hostname} /></td>
      <td data-label="Actions">
        <div className="row-actions">
          {isStale ? (
            <button className="row-remove-btn" type="button" disabled={!mutationEnabled} onClick={(e) => { e.stopPropagation(); void onRemove(entry.hostname); }}>
              Remove
            </button>
          ) : (
            <button className="row-sync-btn" type="button" onClick={(e) => { e.stopPropagation(); onQuickSync(entry.hostname); }}>
              Sync
            </button>
          )}
          <button className="row-modify-btn" type="button" onClick={(e) => { e.stopPropagation(); onOpenModify(entry.hostname); }}>
            Modify
          </button>
        </div>
      </td>
    </tr>
  );
}

function StatusChip({ entry }: { entry: Entry }) {
  return <span className={`status-chip ${statusClassByCode(entry.overall_status)}`}>{entry.status_label || 'Unknown'}</span>;
}

function ServiceBadges({ entry }: { entry: Entry }) {
  return (
    <div className="service-stack">
      <ServiceBadge name="Unbound" status={entry.unbound_status} />
      <ServiceBadge name="AdGuard" status={entry.adguard_status} />
      <ServiceBadge name="DHCP" status={entry.dhcp_status} />
    </div>
  );
}

function ServiceBadge({ name, status }: { name: string; status: { configured: boolean; in_sync: boolean; ip: string } }) {
  let tone = 'missing';
  let label = 'Missing';
  if (status.configured && status.in_sync) {
    tone = 'ok';
    label = status.ip || 'In sync';
  } else if (status.configured) {
    tone = 'bad';
    label = status.ip || 'Mismatch';
  }
  return <span className={`service-badge ${tone}`}><strong>{name}</strong>{label}</span>;
}

const DNS_RETRY_INTERVAL_MS = 10_000;

function DNSProbe({ hostname }: { hostname: string }) {
  const [state, setState] = useState<'idle' | 'loading' | 'ok' | 'fail'>('idle');
  const [tip, setTip] = useState('');
  const [countdown, setCountdown] = useState(0);

  const probe = useCallback(async () => {
    setState('loading');
    try {
      const res = await api.dnsProbe(hostname);
      if (res.resolved) {
        const detail = res.cname ? `→ ${res.cname}` : (res.addresses?.[0] ?? '');
        setTip(`Resolves via 1.1.1.1: ${detail}`);
        setState('ok');
      } else {
        setTip(res.error ? `Not resolving: ${res.error}` : 'Not resolving yet');
        setState('fail');
        setCountdown(DNS_RETRY_INTERVAL_MS / 1000);
      }
    } catch {
      setTip('Probe failed');
      setState('fail');
      setCountdown(DNS_RETRY_INTERVAL_MS / 1000);
    }
  }, [hostname]);

  const handleClick = useCallback((e: MouseEvent) => {
    e.stopPropagation();
    void probe();
  }, [probe]);

  // Auto-retry countdown when failing
  useEffect(() => {
    if (state !== 'fail') return;
    if (countdown <= 0) {
      void probe();
      return;
    }
    const id = setTimeout(() => setCountdown(c => c - 1), 1000);
    return () => clearTimeout(id);
  }, [state, countdown, probe]);

  if (state === 'idle') {
    return <button type="button" className="dns-probe-btn" onClick={handleClick} title="Check if hostname resolves via 1.1.1.1">DNS?</button>;
  }
  if (state === 'loading') {
    return <span className="dns-probe-badge loading" title="Probing…">⋯</span>;
  }
  if (state === 'ok') {
    return <span className="dns-probe-badge ok" title={tip}>✓ DNS</span>;
  }
  // fail — show countdown until next retry, click to retry immediately
  return (
    <button type="button" className="dns-probe-badge fail" title={tip} onClick={handleClick}>
      ✗ DNS {countdown > 0 ? `(${countdown}s)` : ''}
    </button>
  );
}

function CloudflareDetails({ status, hostname }: { status: Entry['cloudflare_status']; hostname: string }) {
  if (!status?.configured) return <span className="cloudflare-detail missing"><strong>Not routed</strong><span>No tunnel rule</span></span>;
  return (
    <span className={`cloudflare-detail ${status.http_host_header ? 'ok' : 'bad'}`}>
      <strong>{status.tunnel_name || 'Tunnel'}</strong>
      <span>{status.service || '-'}</span>
      <span>{status.http_host_header ? `Host header ${status.http_host_header}` : 'Missing HTTPHostHeader'}</span>
      <span>{status.has_access_policy ? 'Access policy' : 'No access policy'}</span>
      <DNSProbe hostname={hostname} />
    </span>
  );
}

function SyncPanel({
  enabledServices,
  caddyServerIP,
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

function CloudflareRoutePanel({ entry, caddyServerIP, mutationEnabled, onRefresh }: {
  entry: import('../types').Entry;
  caddyServerIP: string;
  mutationEnabled: boolean;
  onRefresh: () => void;
}) {
  const cf = entry.cloudflare_status;
  const [saving, setSaving] = useState(false);
  const [removed, setRemoved] = useState(false);
  const [error, setError] = useState('');
  const [dnsWarning, setDnsWarning] = useState('');

  const viaService = cf.service ?? '';
  const isViaCaddy = !!(caddyServerIP && viaService.includes(caddyServerIP));

  // Derive scheme and host from caddy_upstream for direct mode
  const upstreamRaw = entry.caddy_upstream ?? '';
  const upstreamHasScheme = upstreamRaw.startsWith('http://') || upstreamRaw.startsWith('https://');
  const upstreamBase = upstreamHasScheme ? upstreamRaw.replace(/^https?:\/\//, '') : upstreamRaw;

  const [directScheme, setDirectScheme] = useState<'http' | 'https'>(
    upstreamRaw.startsWith('https://') ? 'https' : 'http'
  );
  const [noTLSVerify, setNoTLSVerify] = useState(cf.no_tls_verify ?? false);

  const directService = upstreamBase ? `${directScheme}://${upstreamBase}` : '';
  const caddyService = caddyServerIP ? `https://${caddyServerIP}` : '';

  const setRoute = async (mode: 'caddy' | 'direct') => {
    setSaving(true);
    setError('');
    setDnsWarning('');
    try {
      const service = mode === 'caddy' ? caddyService : directService;
      const httpHostHeader = mode === 'caddy' ? entry.hostname : '';
      const res = await api.cfSetRoute({ hostname: entry.hostname, service, http_host_header: httpHostHeader, no_tls_verify: mode === 'direct' ? noTLSVerify : false });
      if (res.dns_warning) setDnsWarning(res.dns_warning);
      onRefresh();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed');
    } finally {
      setSaving(false);
    }
  };

  const removeRoute = async () => {
    setSaving(true);
    setError('');
    try {
      await api.cfRemoveRoute(entry.hostname);
      setRemoved(true);
      onRefresh();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed');
    } finally {
      setSaving(false);
    }
  };

  const configured = cf.configured && !removed;

  return (
    <div className="cf-route-panel">
      <div className="cf-route-header">
        <span className="cf-route-title">Cloudflare Tunnel</span>
        {configured
          ? <span className={`service-sync-badge ok`}>{isViaCaddy ? 'Via Caddy' : 'Direct'}</span>
          : <span className="service-sync-badge missing">{removed ? 'Removed' : 'Not in tunnel'}</span>
        }
      </div>

      {configured && (
        <div className="cf-route-service muted" title={viaService}>{viaService}</div>
      )}
      {configured && cf.has_access_policy && (
        <div className="cf-route-tag">Access policy</div>
      )}

      {mutationEnabled && !removed && upstreamBase && !isViaCaddy && (
        <div className="cf-route-direct-opts">
          <span className="cf-route-label">Direct scheme:</span>
          <button type="button" className={`btn-sm${directScheme === 'http' ? ' btn-primary' : ''}`} onClick={() => setDirectScheme('http')}>http</button>
          <button type="button" className={`btn-sm${directScheme === 'https' ? ' btn-primary' : ''}`} onClick={() => setDirectScheme('https')}>https</button>
          {directScheme === 'https' && (
            <label className="cf-route-tls-label">
              <input type="checkbox" checked={noTLSVerify} onChange={e => setNoTLSVerify(e.target.checked)} />
              Skip TLS verify
            </label>
          )}
        </div>
      )}

      {mutationEnabled && !removed && (
        <div className="cf-route-actions">
          {configured ? (
            <>
              {isViaCaddy ? (
                <button type="button" className="btn-sm" onClick={() => void setRoute('direct')} disabled={saving || !directService} title={directService}>
                  {saving ? <Loader2 size={11} className="spin" /> : null} Switch to Direct
                </button>
              ) : (
                <button type="button" className="btn-sm" onClick={() => void setRoute('caddy')} disabled={saving || !caddyService} title={caddyService}>
                  {saving ? <Loader2 size={11} className="spin" /> : null} Switch to Caddy
                </button>
              )}
              <button type="button" className="btn-sm btn-danger-sm" onClick={() => void removeRoute()} disabled={saving}>
                {saving ? <Loader2 size={11} className="spin" /> : <Trash2 size={12} />} Remove
              </button>
            </>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6, alignItems: 'flex-start' }}>
              {caddyService && (
                <button type="button" className="btn-sm" onClick={() => void setRoute('caddy')} disabled={saving}>
                  {saving ? <Loader2 size={11} className="spin" /> : null} Route via Caddy ({caddyServerIP})
                </button>
              )}
              {directService && (
                <button type="button" className="btn-sm" onClick={() => void setRoute('direct')} disabled={saving}>
                  {saving ? <Loader2 size={11} className="spin" /> : null} Route direct ({directService})
                </button>
              )}
            </div>
          )}
          {saving && <span className="cf-route-saving"><Loader2 size={12} className="spin" /> Updating tunnel + DNS…</span>}
        </div>
      )}

      {error && <div className="cf-route-error">{error}</div>}
      {dnsWarning && <div className="cf-route-dns-warning">{dnsWarning}</div>}
    </div>
  );
}

function SyncModal({
  open,
  autoSync,
  onClose,
  hostname,
  entry,
  enabledServices,
  caddyServerIP,
  syncService,
  setSyncService,
  syncLoading,
  syncProgress,
  syncLog,
  plannedActions,
  canSyncNow,
  mutationEnabled,
  onPreviewFor,
  onDryRun,
  onSync,
  onRefresh,
  onRemoveEntry,
}: {
  open: boolean;
  autoSync: boolean;
  onClose: () => void;
  hostname: string;
  entry?: import('../types').Entry;
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
  onPreviewFor: (service: string, hostname: string) => Promise<boolean>;
  onDryRun: () => Promise<void>;
  onSync: () => Promise<void>;
  onRefresh: () => void;
  onRemoveEntry: (hostname: string, service?: string) => Promise<void>;
}) {
  const isStale = entry?.overall_status === 4;
  const hostnameDecision = entry ? getHostnameDecision(entry, caddyServerIP) : null;
  const [caddyRaw, setCaddyRaw] = useState('');
  const [caddyEntry, setCaddyEntry] = useState<import('../types').CaddyEntry | null>(null);
  const [caddyTemplates, setCaddyTemplates] = useState<string[]>([]);
  const [caddyDefaultTemplate, setCaddyDefaultTemplate] = useState('default');
  const [caddyRepoPath, setCaddyRepoPath] = useState('');
  const [caddyOpen, setCaddyOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const hasAutoSynced = useRef(false);

  // Track which services have been removed in this modal session so buttons
  // flip immediately without waiting for the next data refresh.
  const [removingService, setRemovingService] = useState<string | null>(null);
  const [localRemoved, setLocalRemoved] = useState<Set<string>>(new Set());

  // Live server log streaming — poll /api/logs while an operation is in progress.
  const [liveLog, setLiveLog] = useState<string>('');
  const logCursorRef = useRef(0);

  // Reset local remove state and log whenever the modal opens for a new hostname.
  useEffect(() => {
    if (open) {
      setLocalRemoved(new Set());
      setRemovingService(null);
      setLiveLog('');
      logCursorRef.current = 0;
    }
  }, [open, hostname]);

  const busy = syncLoading || removingService !== null;

  // Poll server logs while busy.
  useEffect(() => {
    if (!busy) return;
    let cancelled = false;
    const poll = async () => {
      while (!cancelled) {
        try {
          const data = await api.logs(logCursorRef.current);
          if (data.lines.length > 0) {
            logCursorRef.current = data.cursor;
            setLiveLog(prev => {
              const newLines = data.lines.map(l => `[${l.level}] ${l.message}`).join('\n');
              return prev ? prev + '\n' + newLines : newLines;
            });
          }
        } catch { /* ignore */ }
        await new Promise(r => setTimeout(r, 400));
      }
    };
    void poll();
    return () => { cancelled = true; };
  }, [busy]);

  useEffect(() => {
    if (!open || !hostname) { setCaddyRaw(''); setCaddyEntry(null); return; }
    Promise.all([api.caddyEntries(), api.caddyTemplates()]).then(([res, tmpl]) => {
      const found = res.entries.find(e => e.hostname === hostname) ?? null;
      setCaddyRaw(found?.raw ?? '');
      setCaddyEntry(found);
      setCaddyRepoPath(res.editor?.repo_path ?? '');
      setCaddyTemplates(tmpl.templates);
      setCaddyDefaultTemplate(tmpl.default);
    }).catch(() => {});
  }, [open, hostname]);

  useEffect(() => {
    if (open && autoSync && !hasAutoSynced.current) {
      hasAutoSynced.current = true;
      void (async () => {
        const ok = await onPreviewFor('all', hostname);
        if (ok) { await onSync(); onRefresh(); }
      })();
    }
    if (!open) hasAutoSynced.current = false;
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
    if (ok) { await onSync(); onRefresh(); }
  };

  const runSyncAll = async () => {
    setSyncService('all');
    const ok = await onPreviewFor('all', hostname);
    if (ok) { await onSync(); onRefresh(); }
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
              <Trash2 size={13} /> {removingService === 'all' ? 'Removing…' : 'Remove all'}
            </button>
            <button type="button" className="modal-close" onClick={onClose} disabled={busy}><X size={16} /></button>
          </div>
        </div>
        <div className="modal-body sync-modal-body">
          {hostnameDecision && (
            <HostnameDecisionPanel decision={hostnameDecision} />
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
              //   configured + not locally removed → always show Remove
              //   stale → only Remove (sync makes no sense for stale entries)
              //   in sync → show "✓ In sync" indicator + Remove
              //   out of sync / missing → Sync button (+ Remove if configured)
              const removeBtn = configured && !isLocallyRemoved && mutationEnabled ? (
                <button type="button" className="btn-sm btn-danger-sm" onClick={() => void handleServiceRemove(key)} disabled={busy}>
                  <Trash2 size={12} /> {isRemoving ? 'Removing…' : 'Remove'}
                </button>
              ) : null;

              let primaryBtn: ReactNode = null;
              if (!isLocallyRemoved) {
                if (isStale) {
                  // stale: remove only, no sync action
                } else if (configured && inSync) {
                  primaryBtn = (
                    <span className="btn-sm btn-synced" style={{ cursor: 'default' }}>✓ In sync</span>
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
                }).catch(() => {});
                onRefresh();
              }}
            />
          )}
        </div>
        {!mutationEnabled && (
          <div className="modal-footer">
            <span className="muted" style={{ fontSize: 12 }}>Read-only session — sync actions are disabled</span>
          </div>
        )}
      </div>
    </div>
  );
}

function DecisionBlock({ kind, severity, title, summary, facts, actions }: {
  kind: string; severity: string; title: string; summary: string;
  facts: string[]; actions: { label: string; description: string }[];
}) {
  return (
    <section className={`hostname-decision ${severity}`}>
      <div className="hostname-decision-header">
        {severity === 'warning' ? <CircleAlert size={15} /> : <ShieldCheck size={15} />}
        <div>
          <strong>{title}</strong>
          <span>{summary}</span>
        </div>
      </div>
      <div className="hostname-decision-facts">
        {facts.map((fact) => <span key={fact}>{fact}</span>)}
      </div>
      {(kind === 'collision' || kind === 'mismatch') && (
        <span className="hostname-decision-note">This is a warning only — pick a path when you next change DNS or Caddy config.</span>
      )}
      <div className="hostname-decision-actions">
        {actions.map((action) => (
          <div key={action.label} className="hostname-decision-action">
            <strong>{action.label}</strong>
            <span>{action.description}</span>
          </div>
        ))}
      </div>
    </section>
  );
}

function HostnameDecisionPanel({ decision }: { decision: ReturnType<typeof getHostnameDecision> }) {
  return (
    <>
      <DecisionBlock {...decision} />
      {decision.warnings.map((w) => (
        <DecisionBlock key={w.kind} {...w} severity="warning" />
      ))}
    </>
  );
}

const MAX_LOG_LINES = 300;

function LogBar({ open, onToggle }: { open: boolean; onToggle: () => void }) {
  const preRef = useRef<HTMLPreElement>(null);
  const cursorRef = useRef(0);
  const [lines, setLines] = useState<string[]>([]);
  const [active, setActive] = useState(false);
  const [height, setHeight] = useState(160);
  const dragRef = useRef<{ startY: number; startH: number } | null>(null);

  const onDragStart = (e: React.MouseEvent) => {
    e.preventDefault();
    dragRef.current = { startY: e.clientY, startH: height };
    const onMove = (ev: MouseEvent) => {
      if (!dragRef.current) return;
      const delta = dragRef.current.startY - ev.clientY; // drag up = bigger
      setHeight(Math.max(80, Math.min(600, dragRef.current.startH + delta)));
    };
    const onUp = () => {
      dragRef.current = null;
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
    };
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
  };

  // Continuously poll /api/logs regardless of open state so the count stays current.
  useEffect(() => {
    let cancelled = false;
    let activityTimer: ReturnType<typeof setTimeout> | null = null;

    const poll = async () => {
      while (!cancelled) {
        try {
          const data = await api.logs(cursorRef.current);
          if (data.lines.length > 0) {
            cursorRef.current = data.cursor;
            setActive(true);
            if (activityTimer) clearTimeout(activityTimer);
            activityTimer = setTimeout(() => setActive(false), 2000);
            setLines(prev => {
              const next = [...prev, ...data.lines.map(l => `[${l.level}] ${l.message}`)];
              return next.length > MAX_LOG_LINES ? next.slice(next.length - MAX_LOG_LINES) : next;
            });
          }
        } catch { /* ignore */ }
        await new Promise(r => setTimeout(r, 600));
      }
    };
    void poll();
    return () => {
      cancelled = true;
      if (activityTimer) clearTimeout(activityTimer);
    };
  }, []);

  // Auto-scroll to bottom when new lines arrive and bar is open.
  useEffect(() => {
    if (open && preRef.current) {
      preRef.current.scrollTop = preRef.current.scrollHeight;
    }
  }, [lines, open]);

  return (
    <div className={`log-bar${open ? ' open' : ''}`}>
      {open && <div className="log-bar-resize-handle" onMouseDown={onDragStart} title="Drag to resize" />}
      <button type="button" className="log-bar-toggle" onClick={onToggle}>
        <Terminal size={13} />
        <span>Server log</span>
        {active && <span className="log-bar-pill working">Live</span>}
        {!active && lines.length > 0 && <span className="log-bar-pill">{lines.length} line{lines.length !== 1 ? 's' : ''}</span>}
        {open ? <ChevronDown size={13} /> : <ChevronUp size={13} />}
      </button>
      {open && (
        <pre ref={preRef} className="log-bar-content" style={{ height: `${height}px` }}>
          {lines.length ? lines.join('\n') : 'Waiting for server log output...'}
        </pre>
      )}
    </div>
  );
}

function CFRepairBanner({ entries, mutationEnabled, cfEnabled, onRepaired }: {
  entries: Entry[];
  mutationEnabled: boolean;
  cfEnabled: boolean;
  onRepaired: () => void;
}) {
  const [repairing, setRepairing] = useState(false);
  const [result, setResult] = useState<{ fixed: string[]; failed: string[] } | null>(null);

  const missingCNAME = entries.filter(e =>
    e.cloudflare_status?.configured && !e.cloudflare_status?.has_dns_record
  );

  if (!cfEnabled || missingCNAME.length === 0) return null;

  const repair = async () => {
    setRepairing(true);
    setResult(null);
    try {
      const r = await api.cfRepairDNS();
      setResult(r);
      onRepaired();
    } catch { /* ignore */ } finally {
      setRepairing(false);
    }
  };

  return (
    <div className="cf-repair-banner">
      <CircleAlert size={14} />
      <span>
        <strong>{missingCNAME.length} CF tunnel {missingCNAME.length === 1 ? 'entry is' : 'entries are'} missing a DNS CNAME record</strong>
        {' '}— they won&apos;t resolve publicly until fixed.
      </span>
      {mutationEnabled && (
        <button type="button" className="btn-sm btn-primary" onClick={() => void repair()} disabled={repairing}>
          {repairing ? <Loader2 size={11} className="spin" /> : null}
          {repairing ? ' Repairing…' : 'Repair all missing CNAMEs'}
        </button>
      )}
      {result && result.fixed.length > 0 && (
        <span className="cf-repair-ok">
          <CheckCircle2 size={12} /> Fixed: {result.fixed.join(', ')}
        </span>
      )}
      {result && result.failed.length > 0 && (
        <span className="cf-repair-fail">Failed: {result.failed.join(', ')}</span>
      )}
    </div>
  );
}

function InlineProgress({ loading, title, detail }: { loading: boolean; title: string; detail: string }) {
  return (
    <div id="sync-progress" className="inline-progress" role="status" aria-live="polite" aria-label={title} hidden={!loading}>
      <div className="loading-copy compact">
        <span id="sync-progress-title">{title}</span>
        <strong>Working</strong>
      </div>
      <div className="progress-track"><span /></div>
      <small id="sync-progress-detail">{detail}</small>
    </div>
  );
}

function HostInspector({ entry, mutationEnabled, onPreview, onSync }: { entry?: Entry; mutationEnabled: boolean; onPreview: (hostname: string) => Promise<boolean>; onSync: (hostname: string) => Promise<void> }) {
  if (!entry) {
    return (
      <section id="host-inspector" className="panel inspector" aria-live="polite">
        <div className="panel-title"><strong>Selected host</strong><span>Select a row to inspect service state.</span></div>
        <div className="empty-state"><WifiOff size={18} /> No hostname selected.</div>
      </section>
    );
  }
  return (
    <section id="host-inspector" className="panel inspector" aria-live="polite">
      <div className="host-title">
        <strong>{entry.hostname}</strong>
        <div><StatusChip entry={entry} /><span className={`dns-result ${dnsResultClass(entry.dns_resolved)}`}>{entry.dns_resolved || 'FAIL'}</span></div>
      </div>
      <div className="inspector-grid">
        <InspectorLine label="Caddy upstream" value={entry.caddy_upstream || '-'} />
        <InspectorLine label="Source" value={entry.data_source || '-'} />
        <InspectorLine label="Unbound" value={serviceStateText(entry.unbound_status)} tone={entry.unbound_status?.configured ? 'ok' : 'bad'} />
        <InspectorLine label="AdGuard" value={serviceStateText(entry.adguard_status)} tone={entry.adguard_status?.configured ? 'ok' : 'bad'} />
        <InspectorLine label="DHCP lease" value={serviceStateText(entry.dhcp_status)} />
        <InspectorLine label="Cloudflare route" value={cloudflareStateText(entry.cloudflare_status)} tone={entry.cloudflare_status?.configured ? 'violet' : 'bad'} />
      </div>
      <div className="inspector-actions">
        <button type="button" id="inspector-preview" onClick={() => void onPreview(entry.hostname)}>Preview selected</button>
        <button type="button" id="inspector-sync" disabled={!mutationEnabled} onClick={() => void onSync(entry.hostname)}>Sync selected</button>
      </div>
    </section>
  );
}

function InspectorLine({ label, value, tone = '' }: { label: string; value: string; tone?: string }) {
  return <div className={`inspector-line ${tone}`}><span>{label}</span><strong>{value}</strong></div>;
}

function ConfigModal({
  open,
  onClose,
  config,
  forms,
  setForms,
  savedForms,
  mutationEnabled,
  status,
  statusKind,
  testResults,
  onSave,
  onSaveCaddyEditor,
  onTest
}: {
  open: boolean;
  onClose: () => void;
  config: ConfigResponse | null;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  savedForms: ConfigForms;
  mutationEnabled: boolean;
  status: string;
  statusKind: 'info' | 'error' | 'ok';
  testResults: TestResults;
  onSave: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onSaveCaddyEditor: () => Promise<void>;
  onTest: (service: ServiceKey) => Promise<void>;
}) {
  return (
    <div id="config-panel" className={`config-modal ${open ? 'open' : ''}`} hidden={!open} role="dialog" aria-modal="true" aria-labelledby="config-modal-title">
      <div className="config-backdrop" onClick={onClose} />
      <section className="config-sheet panel">
        <header className="config-sheet-header">
          <div>
            <strong id="config-modal-title"><FileSliders size={16} /> Configuration</strong>
            <span>Runtime sources, editable config, and connection tests.</span>
          </div>
          <button id="config-close" type="button" onClick={onClose}>Close</button>
        </header>
        {config && (
          <ConfigWorkspace
            config={config}
            forms={forms}
            setForms={setForms}
            savedForms={savedForms}
            mutationEnabled={mutationEnabled}
            status={status}
            statusKind={statusKind}
            testResults={testResults}
            onSave={onSave}
            onSaveCaddyEditor={onSaveCaddyEditor}
            onTest={onTest}
          />
        )}
      </section>
    </div>
  );
}

type ConfigTab = ServiceKey | 'caddy-editor';

const configTabLabels: Record<ConfigTab, string> = {
  caddy:        'Caddy',
  unbound:      'Unbound',
  adguard:      'AdGuard',
  dhcp:         'DHCP',
  cloudflare:   'Cloudflare',
  'caddy-editor': 'File Editor',
};

const configTabOrder: ConfigTab[] = ['caddy', 'unbound', 'adguard', 'dhcp', 'cloudflare', 'caddy-editor'];

function ConfigWorkspace(props: {
  config: ConfigResponse;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  savedForms: ConfigForms;
  mutationEnabled: boolean;
  status: string;
  statusKind: 'info' | 'error' | 'ok';
  testResults: TestResults;
  onSave: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onSaveCaddyEditor: () => Promise<void>;
  onTest: (service: ServiceKey) => Promise<void>;
}) {
  const [activeTab, setActiveTab] = useState<ConfigTab>('caddy');
  return (
    <div id="config-summary" className="config-workspace">
      <div id="config-status" className={`config-status ${props.statusKind}`} role="status" aria-live="polite">{props.status}</div>
      <div className="cfg-tab-bar" role="tablist">
        {configTabOrder.map((tab) => (
          <button
            key={tab}
            role="tab"
            aria-selected={activeTab === tab}
            className={`cfg-tab ${activeTab === tab ? 'active' : ''}`}
            onClick={() => setActiveTab(tab)}
          >
            {configTabLabels[tab]}
          </button>
        ))}
      </div>
      <div className="cfg-tab-content">
        {activeTab === 'caddy-editor' ? (
          <CaddyEditorSetupPanel
            form={props.forms.caddyEditor}
            savedForm={props.savedForms.caddyEditor}
            setForm={(updater) => props.setForms((current) => ({ ...current, caddyEditor: typeof updater === 'function' ? updater(current.caddyEditor) : updater }))}
            mutationEnabled={props.mutationEnabled}
            onSave={props.onSaveCaddyEditor}
          />
        ) : (
          <ConfigCard service={activeTab} {...props} />
        )}
      </div>
    </div>
  );
}

function CaddyEditorSetupPanel({
  form,
  savedForm,
  setForm,
  mutationEnabled,
  onSave
}: {
  form: CaddyEditorForm;
  savedForm: CaddyEditorForm;
  setForm: (updater: CaddyEditorForm | ((prev: CaddyEditorForm) => CaddyEditorForm)) => void;
  mutationEnabled: boolean;
  onSave: () => Promise<void>;
}) {
  const dirty = JSON.stringify(form) !== JSON.stringify(savedForm);
  const [availableTemplates, setAvailableTemplates] = useState<string[]>([]);
  const set = <K extends keyof CaddyEditorForm>(key: K, value: CaddyEditorForm[K]) =>
    setForm((prev) => ({ ...prev, [key]: value }));

  useEffect(() => {
    api.caddyTemplates().then((res) => setAvailableTemplates(res.templates)).catch(() => {});
  }, []);
  return (
    <section className="caddy-editor-setup">
      <header className="caddy-editor-setup-header">
        <div>
          <strong><FileCode2 size={15} /> Caddy File Editor</strong>
          <span>Configure the Caddyfile editor — repo path, deploy command, and git settings.</span>
        </div>
      </header>
      <div className="caddy-editor-setup-body">
        <label className="checkbox-row">
          <input id="ce-enabled" type="checkbox" checked={form.enabled} onChange={(e) => set('enabled', e.target.checked)} />
          Enabled
        </label>

        {/* Paths */}
        <div className="ce-section-label">Paths</div>
        <div className="caddy-editor-setup-grid">
          <Field label="Repo path">
            <input id="ce-repo-path" type="text" value={form.repo_path} placeholder="/etc/caddy" onChange={(e) => set('repo_path', e.target.value)} />
          </Field>
          <Field label="Caddyfile (relative to repo path)">
            <input id="ce-caddyfile" type="text" value={form.caddyfile} placeholder="Caddyfile" onChange={(e) => set('caddyfile', e.target.value)} />
          </Field>
          <Field label="Entry template">
            {availableTemplates.length > 0 ? (
              <div className="select-wrapper">
                <select id="ce-entry-template" value={form.entry_template} onChange={(e) => set('entry_template', e.target.value)}>
                  {availableTemplates.map((t) => <option key={t} value={t}>{t}</option>)}
                </select>
                <ChevronDown size={14} />
              </div>
            ) : (
              <input id="ce-entry-template" type="text" value={form.entry_template} placeholder="default" onChange={(e) => set('entry_template', e.target.value)} />
            )}
          </Field>
        </div>

        {/* Commands — textarea so the full string is visible */}
        <div className="ce-section-label">Commands</div>
        <div className="caddy-editor-setup-grid">
          <Field label="Deploy command">
            <textarea id="ce-deploy-command" rows={2} value={form.deploy_command} placeholder="make deploy" onChange={(e) => set('deploy_command', e.target.value)} />
          </Field>
          <Field label="Validate command">
            <textarea id="ce-validate-command" rows={2} value={form.validate_command} placeholder="caddy validate --config Caddyfile" onChange={(e) => set('validate_command', e.target.value)} />
          </Field>
        </div>

        {/* Git */}
        <div className="ce-section-label">Git</div>
        <div className="caddy-editor-setup-grid ce-grid-2col">
          <Field label="Remote">
            <input id="ce-git-remote" type="text" value={form.git_remote} placeholder="origin" onChange={(e) => set('git_remote', e.target.value)} />
          </Field>
          <Field label="Branch">
            <input id="ce-git-branch" type="text" value={form.git_branch} placeholder="main" onChange={(e) => set('git_branch', e.target.value)} />
          </Field>
        </div>
        <div className="caddy-editor-setup-checks">
          <label className="checkbox-row">
            <input id="ce-git-auto-commit" type="checkbox" checked={form.git_auto_commit} onChange={(e) => set('git_auto_commit', e.target.checked)} />
            Auto-commit on save
          </label>
          <label className="checkbox-row">
            <input id="ce-git-auto-push" type="checkbox" checked={form.git_auto_push} onChange={(e) => set('git_auto_push', e.target.checked)} />
            Auto-push on save
          </label>
        </div>

        <div className="config-actions">
          <button type="button" id="ce-save" className={dirty ? 'btn-primary' : ''} disabled={!mutationEnabled} onClick={() => void onSave()}>Save Caddy Editor Config</button>
        </div>
      </div>
    </section>
  );
}

function ConfigCard({
  service,
  config,
  forms,
  setForms,
  savedForms,
  mutationEnabled,
  testResults,
  onSave,
  onTest
}: {
  service: ServiceKey;
  config: ConfigResponse;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  savedForms: ConfigForms;
  mutationEnabled: boolean;
  testResults: TestResults;
  onSave: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onTest: (service: ServiceKey) => Promise<void>;
}) {
  const summary = config.summary[service] as ConfigServiceSummary | undefined;
  if (!summary) return null;
  const meta = serviceMeta[service];
  const Icon = meta.icon;
  const fields = Object.entries(summary.fields || {});
  const details = Object.entries(summary.details || {}).filter(([, value]) => value);
  const missing = summary.missing || [];
  const tone = summary.client_ready ? 'ok' : summary.enabled ? 'warn' : 'missing';
  return (
    <article className={`config-card ${tone} ${meta.tone}`}>
      <header>
        <div>
          <strong><Icon size={15} /> {summary.label || meta.label}</strong>
          <span>{summary.client_ready ? 'Client ready' : summary.enabled ? 'Configured, incomplete' : 'Not configured'}</span>
        </div>
        <em>{compactSourceKind(summary.source)}</em>
      </header>
      <ConfigLine label="Source" value={sourceLabel(summary.source)} />
      {summary.endpoint && <ConfigLine label="Endpoint" value={summary.endpoint} />}
      {summary.insecure && <ConfigLine label="TLS" value="Insecure verification" warn />}
      {details.map(([key, value]) => <ConfigLine key={key} label={formatConfigKey(key)} value={value} />)}
      {fields.map(([key, value]) => <ConfigLine key={key} label={formatConfigKey(key)} value={value ? 'set' : 'missing'} />)}
      <div className={`missing-list ${missing.length ? '' : 'ok'}`}>{missing.length ? `Missing: ${missing.join(', ')}` : 'Required fields present'}</div>
      <ConfigEditor service={service} forms={forms} setForms={setForms} savedForms={savedForms} mutationEnabled={mutationEnabled} testResult={testResults[service]} onSave={onSave} onTest={onTest} />
    </article>
  );
}

function ConfigLine({ label, value, warn = false }: { label: string; value: string; warn?: boolean }) {
  return <div className={`config-line ${warn ? 'warn' : ''}`}><span>{label}</span><strong>{value}</strong></div>;
}

function ConfigEditor({
  service,
  forms,
  setForms,
  savedForms,
  mutationEnabled,
  testResult,
  onSave,
  onTest
}: {
  service: ServiceKey;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  savedForms: ConfigForms;
  mutationEnabled: boolean;
  testResult?: { text: string; kind: 'info' | 'ok' | 'error' };
  onSave: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onTest: (service: ServiceKey) => Promise<void>;
}) {
  const isDirty = (key: keyof ConfigForms) =>
    JSON.stringify(forms[key]) !== JSON.stringify(savedForms[key]);

  if (service === 'caddy') {
    return <div className="config-editor compact" data-config-editor="caddy"><ConfigTestResult service={service} result={testResult} /><button type="button" data-config-test="caddy" disabled={!mutationEnabled} onClick={() => void onTest('caddy')}>Test Caddy</button></div>;
  }
  if (service === 'dhcp') return null;
  if (service === 'unbound') {
    const dirty = isDirty('unbound');
    return (
      <div className="config-editor" data-config-editor="unbound">
        <Field label="Base URL"><input id="config-unbound-base-url" type="url" value={forms.unbound.base_url} placeholder="https://opnsense.local" onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, base_url: event.target.value } }))} /></Field>
        <Field label="API key"><input id="config-unbound-api-key" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.unbound.api_key} onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, api_key: event.target.value } }))} /></Field>
        <Field label="API secret"><input id="config-unbound-api-secret" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.unbound.api_secret} onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, api_secret: event.target.value } }))} /></Field>
        <label className="checkbox-row"><input id="config-unbound-insecure" type="checkbox" checked={forms.unbound.insecure} onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, insecure: event.target.checked } }))} /> Insecure TLS</label>
        <ConfigTestResult service={service} result={testResult} />
        <div className="config-actions">
          <button type="button" data-config-test="unbound" disabled={!mutationEnabled} onClick={() => void onTest('unbound')}>Test OPNSense</button>
          <button type="button" data-config-save="unbound" className={dirty ? 'btn-primary' : ''} disabled={!mutationEnabled} onClick={() => void onSave('unbound')}>Save OPNSense</button>
        </div>
      </div>
    );
  }
  if (service === 'adguard') {
    const dirty = isDirty('adguard');
    return (
      <div className="config-editor" data-config-editor="adguard">
        <label className="checkbox-row"><input id="config-adguard-enabled" type="checkbox" checked={forms.adguard.enabled} onChange={(event) => setForms((current) => ({ ...current, adguard: { ...current.adguard, enabled: event.target.checked } }))} /> Enabled</label>
        <Field label="Base URL"><input id="config-adguard-base-url" type="url" value={forms.adguard.base_url} placeholder="https://adguard.local" onChange={(event) => setForms((current) => ({ ...current, adguard: { ...current.adguard, base_url: event.target.value } }))} /></Field>
        <Field label="Username"><input id="config-adguard-username" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.adguard.username} onChange={(event) => setForms((current) => ({ ...current, adguard: { ...current.adguard, username: event.target.value } }))} /></Field>
        <Field label="Password"><input id="config-adguard-password" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.adguard.password} onChange={(event) => setForms((current) => ({ ...current, adguard: { ...current.adguard, password: event.target.value } }))} /></Field>
        <label className="checkbox-row"><input id="config-adguard-insecure" type="checkbox" checked={forms.adguard.insecure} onChange={(event) => setForms((current) => ({ ...current, adguard: { ...current.adguard, insecure: event.target.checked } }))} /> Insecure TLS</label>
        <ConfigTestResult service={service} result={testResult} />
        <div className="config-actions">
          <button type="button" data-config-test="adguard" disabled={!mutationEnabled} onClick={() => void onTest('adguard')}>Test AdGuard</button>
          <button type="button" data-config-save="adguard" className={dirty ? 'btn-primary' : ''} disabled={!mutationEnabled} onClick={() => void onSave('adguard')}>Save AdGuard</button>
        </div>
      </div>
    );
  }
  const dirty = isDirty('cloudflare');
  return (
    <div className="config-editor" data-config-editor="cloudflare">
      <label className="checkbox-row"><input id="config-cloudflare-enabled" type="checkbox" checked={forms.cloudflare.enabled} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, enabled: event.target.checked } }))} /> Enabled</label>
      <Field label="API token"><input id="config-cloudflare-api-token" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.cloudflare.api_token} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, api_token: event.target.value } }))} /></Field>
      <Field label="Account ID"><input id="config-cloudflare-account-id" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.cloudflare.account_id} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, account_id: event.target.value } }))} /></Field>
      <Field label="Zone ID"><input id="config-cloudflare-zone-id" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.cloudflare.zone_id} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, zone_id: event.target.value } }))} /></Field>
      <Field label="Tunnel ID"><input id="config-cloudflare-tunnel-id" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.cloudflare.tunnel_id} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, tunnel_id: event.target.value } }))} /></Field>
      <Field label="Caddy service URL"><input id="config-cloudflare-caddy-service-url" type="url" value={forms.cloudflare.caddy_service_url} placeholder="http://127.0.0.1:80" onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, caddy_service_url: event.target.value } }))} /></Field>
      <label className="checkbox-row"><input id="config-cloudflare-insecure" type="checkbox" checked={forms.cloudflare.insecure} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, insecure: event.target.checked } }))} /> Insecure TLS</label>
      <ConfigTestResult service={service} result={testResult} />
      <div className="config-actions">
        <button type="button" data-config-test="cloudflare" disabled={!mutationEnabled} onClick={() => void onTest('cloudflare')}>Test Cloudflare</button>
        <button type="button" data-config-save="cloudflare" className={dirty ? 'btn-primary' : ''} disabled={!mutationEnabled} onClick={() => void onSave('cloudflare')}>Save Cloudflare</button>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return <label>{label}{children}</label>;
}

function ConfigTestResult({ service, result }: { service: ServiceKey; result?: { text: string; kind: 'info' | 'ok' | 'error' } }) {
  return <div id={`config-test-${service}`} className={`config-test-result ${result?.kind || ''}`} role="status" aria-live="polite">{result?.text || ''}</div>;
}

function Select({ value, onChange, options, ariaLabel, disabledValues = new Set<string>(), idForAll, className }: { value: string; onChange: (value: string) => void; options: [string, string][]; ariaLabel: string; disabledValues?: Set<string>; idForAll?: string; className?: string }) {
  return (
    <span className={`select-wrap ${className || ''}`}>
      <select value={value} aria-label={ariaLabel} onClick={(event) => event.stopPropagation()} onChange={(event) => onChange(event.target.value)}>
        {options.map(([optionValue, label]) => (
          <option key={optionValue} id={idForAll && optionValue === 'all' ? idForAll : undefined} value={optionValue} data-service={optionValue !== 'all' ? optionValue : undefined} disabled={disabledValues.has(optionValue)} className={disabledValues.has(optionValue) ? 'disabled-service' : undefined}>{label}</option>
        ))}
      </select>
      <ChevronDown size={14} />
    </span>
  );
}
