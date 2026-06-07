import {
  CheckCircle2,
  ChevronDown,
  CircleAlert,
  CircleDot,
  Cloud,
  FileSliders,
  Gauge,
  ListFilter,
  Play,
  RefreshCw,
  Search,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  TerminalSquare,
  WifiOff,
  Zap
} from 'lucide-react';
import { useState } from 'react';
import type { Dispatch, KeyboardEvent, ReactNode, SetStateAction } from 'react';
import type { ConfigForms, TestResults } from '../hooks/useConfigForms';
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
import type { ConfigResponse, ConfigServiceSummary, EntriesResponse, Entry, ServiceKey, SyncAction } from '../types';

export function AppShell({
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
  configOpen,
  setConfigOpen,
  forms,
  setForms,
  configStatus,
  configStatusKind,
  testResults,
  onSaveConfig,
  onTestConfig
}: {
  config: ConfigResponse | null;
  loading: boolean;
  message: string;
  messageKind: 'info' | 'error' | 'ok';
  report: EntriesResponse['report'];
  summary: { entries: number; inSync: number; out: number; caddyOnly: number; stale: number; cloudflare: number };
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
  configOpen: boolean;
  setConfigOpen: (value: boolean) => void;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  configStatus: string;
  configStatusKind: 'info' | 'error' | 'ok';
  testResults: TestResults;
  onSaveConfig: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onTestConfig: (service: ServiceKey) => Promise<void>;
}) {
  const enabledServices = config?.enabled || {};
  return (
    <div className="console-layout">
      <Sidebar config={config} report={report} onOpenConfig={() => setConfigOpen(true)} />
      <div className="console-main">
        <Topbar config={config} loading={loading} onRefresh={onRefresh} onOpenConfig={() => setConfigOpen(true)} />
        <main className="dashboard-shell">
          <OperationsHeader
            loading={loading}
            message={message}
            messageKind={messageKind}
            summary={summary}
          />
          <section className="workspace-grid">
            <section className="content-stack">
              <MetricGrid summary={summary} />
              <EntriesToolbar
                entriesCount={entries.length}
                statusFilter={statusFilter}
                setStatusFilter={setStatusFilter}
                serviceFilter={serviceFilter}
                setServiceFilter={setServiceFilter}
                search={search}
                setSearch={setSearch}
                enabledServices={enabledServices}
              />
              <EntriesTable
                entries={entries}
                selectedHostname={selectedHostname}
                mutationEnabled={mutationEnabled}
                enabledServices={enabledServices}
                onSelect={setSelectedHostname}
                onPreview={onPreview}
                onSync={async (service, hostname) => {
                  setSelectedHostname(hostname);
                  if (await onPreview(service, hostname)) await onSync();
                }}
              />
            </section>
            <aside className="right-rail">
              <SyncPanel
                enabledServices={enabledServices}
                syncService={syncService}
                setSyncService={setSyncService}
                syncLoading={syncLoading}
                syncProgress={syncProgress}
                syncLog={syncLog}
                plannedActions={plannedActions}
                canSyncNow={canSyncNow}
                mutationEnabled={mutationEnabled}
                onPreview={() => onPreview()}
                onDryRun={onDryRun}
                onSync={onSync}
              />
              <HostInspector
                entry={selectedEntry}
                mutationEnabled={mutationEnabled}
                onPreview={(hostname) => onPreview(syncService, hostname)}
                onSync={async (hostname) => {
                  if (await onPreview(syncService, hostname)) await onSync();
                }}
              />
            </aside>
          </section>
        </main>
      </div>
      <ConfigModal
        open={configOpen}
        onClose={() => setConfigOpen(false)}
        config={config}
        forms={forms}
        setForms={setForms}
        mutationEnabled={mutationEnabled}
        status={configStatus}
        statusKind={configStatusKind}
        testResults={testResults}
        onSave={onSaveConfig}
        onTest={onTestConfig}
      />
    </div>
  );
}

function Sidebar({ config, report, onOpenConfig }: { config: ConfigResponse | null; report: EntriesResponse['report']; onOpenConfig: () => void }) {
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
        <a className="nav-item active" href="#entries-panel"><Gauge size={15} /> Dashboard</a>
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

function Topbar({ config, loading, onRefresh, onOpenConfig }: { config: ConfigResponse | null; loading: boolean; onRefresh: () => void; onOpenConfig: () => void }) {
  return (
    <header className="topbar">
      <div className="runtime-card" id="runtime">
        <span>Caddy runtime</span>
        <strong>{config ? `${config.caddy.server_ip}:${config.caddy.server_port}` : 'Loading...'}</strong>
        <em className={config?.enabled?.caddy === false ? 'down' : 'up'}><CircleDot size={12} /> {config?.enabled?.caddy === false ? 'Offline' : 'Running'}</em>
      </div>
      <div className="top-actions">
        <button id="refresh" type="button" onClick={onRefresh} disabled={loading}>
          <RefreshCw size={16} /> Refresh
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
  summary: { entries: number; inSync: number; out: number; caddyOnly: number; stale: number; cloudflare: number };
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

function MetricGrid({ summary }: { summary: { entries: number; inSync: number; out: number; caddyOnly: number; stale: number; cloudflare: number } }) {
  return (
    <section id="summary" className="metric-grid" aria-live="polite">
      <Metric label="Total entries" value={summary.entries} sublabel="hostnames" icon={<ListFilter size={20} />} tone="neutral" />
      <Metric label="In sync" value={summary.inSync} sublabel="perfect" icon={<CheckCircle2 size={20} />} tone="ok" />
      <Metric label="Caddy only" value={summary.caddyOnly} sublabel="not in DNS" icon={<CircleAlert size={20} />} tone="warn" />
      <Metric label="Stale DNS" value={summary.stale} sublabel="needs cleanup" icon={<SlidersHorizontal size={20} />} tone="bad" />
      <Metric label="Cloudflare routed" value={summary.cloudflare} sublabel="via tunnel" icon={<Cloud size={20} />} tone="violet" />
    </section>
  );
}

function Metric({ label, value, sublabel, icon, tone }: { label: string; value: number; sublabel: string; icon: ReactNode; tone: string }) {
  return (
    <article className={`metric-card ${tone}`}>
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
  enabledServices
}: {
  entriesCount: number;
  statusFilter: string;
  setStatusFilter: (value: string) => void;
  serviceFilter: string;
  setServiceFilter: (value: string) => void;
  search: string;
  setSearch: (value: string) => void;
  enabledServices: Partial<Record<ServiceKey, boolean>>;
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
    </section>
  );
}

function EntriesTable({
  entries,
  selectedHostname,
  mutationEnabled,
  enabledServices,
  onSelect,
  onPreview,
  onSync
}: {
  entries: Entry[];
  selectedHostname: string;
  mutationEnabled: boolean;
  enabledServices: Partial<Record<ServiceKey, boolean>>;
  onSelect: (hostname: string) => void;
  onPreview: (service: string, hostname: string) => Promise<boolean>;
  onSync: (service: string, hostname: string) => Promise<void>;
}) {
  const disabledValues = disabledSyncValues(enabledServices);
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
              disabledValues={disabledValues}
              onSelect={onSelect}
              onPreview={onPreview}
              onSync={onSync}
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
  disabledValues,
  onSelect,
  onPreview,
  onSync
}: {
  entry: Entry;
  selected: boolean;
  mutationEnabled: boolean;
  disabledValues: Set<string>;
  onSelect: (hostname: string) => void;
  onPreview: (service: string, hostname: string) => Promise<boolean>;
  onSync: (service: string, hostname: string) => Promise<void>;
}) {
  const [rowService, setRowService] = useState('all');
  const safeRowService = disabledValues.has(rowService) ? 'all' : rowService;
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
      </td>
      <td data-label="Status"><StatusChip entry={entry} /><span className="status-subtext">{statusDetail}</span></td>
      <td data-label="Services"><ServiceBadges entry={entry} /></td>
      <td data-label="Caddy upstream"><span>{entry.caddy_upstream || '-'}</span><span className="subtle">admin {entry.caddy_ip || '-'}</span><span className="protocol-pill">HTTP</span></td>
      <td data-label="DNS"><span className={`dns-result ${dnsResultClass(entry.dns_resolved)}`}>{entry.dns_resolved || 'FAIL'}</span><span className="status-subtext">{dnsOK ? 'A record' : 'NXDOMAIN'}</span></td>
      <td data-label="Cloudflare route"><CloudflareDetails status={entry.cloudflare_status} /></td>
      <td data-label="Actions">
        <div className="row-actions">
          <Select value={safeRowService} onChange={setRowService} ariaLabel={`Sync target for ${entry.hostname}`} options={syncOptions({})} disabledValues={disabledValues} className="row-sync-service" />
          <button className="row-preview" type="button" data-hostname={entry.hostname} onClick={(event) => { event.stopPropagation(); onSelect(entry.hostname); void onPreview(safeRowService, entry.hostname); }}>Preview</button>
          <button className="row-sync" type="button" data-hostname={entry.hostname} disabled={!mutationEnabled} title={mutationEnabled ? 'Apply the selected server-issued sync plan' : 'Real sync is unavailable for this web session'} onClick={(event) => { event.stopPropagation(); onSelect(entry.hostname); void onSync(safeRowService, entry.hostname); }}>Sync</button>
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

function CloudflareDetails({ status }: { status: Entry['cloudflare_status'] }) {
  if (!status?.configured) return <span className="cloudflare-detail missing"><strong>Not routed</strong><span>No tunnel rule</span></span>;
  return (
    <span className={`cloudflare-detail ${status.http_host_header ? 'ok' : 'bad'}`}>
      <strong>{status.tunnel_name || 'Tunnel'}</strong>
      <span>{status.service || '-'}</span>
      <span>{status.http_host_header ? `Host header ${status.http_host_header}` : 'Missing HTTPHostHeader'}</span>
      <span>{status.has_access_policy ? 'Access policy' : 'No access policy'}</span>
    </span>
  );
}

function SyncPanel({
  enabledServices,
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
  mutationEnabled,
  status,
  statusKind,
  testResults,
  onSave,
  onTest
}: {
  open: boolean;
  onClose: () => void;
  config: ConfigResponse | null;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  mutationEnabled: boolean;
  status: string;
  statusKind: 'info' | 'error' | 'ok';
  testResults: TestResults;
  onSave: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
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
            mutationEnabled={mutationEnabled}
            status={status}
            statusKind={statusKind}
            testResults={testResults}
            onSave={onSave}
            onTest={onTest}
          />
        )}
      </section>
    </div>
  );
}

function ConfigWorkspace(props: {
  config: ConfigResponse;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  mutationEnabled: boolean;
  status: string;
  statusKind: 'info' | 'error' | 'ok';
  testResults: TestResults;
  onSave: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onTest: (service: ServiceKey) => Promise<void>;
}) {
  return (
    <div id="config-summary" className="config-workspace">
      <div className="config-overview">
        <div>
          <strong>Runtime sources</strong>
          <span>Save target and active input source are shown per service.</span>
        </div>
        <span className="save-target">Save target: {props.config.save_target || '-'}</span>
      </div>
      <div id="config-status" className={`config-status ${props.statusKind}`} role="status" aria-live="polite">{props.status}</div>
      <div id="service-health" className="config-grid" aria-label="Service health">
        {serviceOrder.map((service) => (
          <ConfigCard key={service} service={service} {...props} />
        ))}
      </div>
    </div>
  );
}

function ConfigCard({
  service,
  config,
  forms,
  setForms,
  mutationEnabled,
  testResults,
  onSave,
  onTest
}: {
  service: ServiceKey;
  config: ConfigResponse;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
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
      <ConfigEditor service={service} forms={forms} setForms={setForms} mutationEnabled={mutationEnabled} testResult={testResults[service]} onSave={onSave} onTest={onTest} />
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
  mutationEnabled,
  testResult,
  onSave,
  onTest
}: {
  service: ServiceKey;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  mutationEnabled: boolean;
  testResult?: { text: string; kind: 'info' | 'ok' | 'error' };
  onSave: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onTest: (service: ServiceKey) => Promise<void>;
}) {
  if (service === 'caddy') {
    return <div className="config-editor compact" data-config-editor="caddy"><ConfigTestResult service={service} result={testResult} /><button type="button" data-config-test="caddy" disabled={!mutationEnabled} onClick={() => void onTest('caddy')}>Test Caddy</button></div>;
  }
  if (service === 'dhcp') return null;
  if (service === 'unbound') {
    return (
      <div className="config-editor" data-config-editor="unbound">
        <Field label="Base URL"><input id="config-unbound-base-url" type="url" value={forms.unbound.base_url} placeholder="https://opnsense.local" onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, base_url: event.target.value } }))} /></Field>
        <Field label="API key"><input id="config-unbound-api-key" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.unbound.api_key} onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, api_key: event.target.value } }))} /></Field>
        <Field label="API secret"><input id="config-unbound-api-secret" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.unbound.api_secret} onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, api_secret: event.target.value } }))} /></Field>
        <label className="checkbox-row"><input id="config-unbound-insecure" type="checkbox" checked={forms.unbound.insecure} onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, insecure: event.target.checked } }))} /> Insecure TLS</label>
        <ConfigTestResult service={service} result={testResult} />
        <div className="config-actions">
          <button type="button" data-config-test="unbound" disabled={!mutationEnabled} onClick={() => void onTest('unbound')}>Test OPNSense</button>
          <button type="button" data-config-save="unbound" disabled={!mutationEnabled} onClick={() => void onSave('unbound')}>Set OPNSense</button>
        </div>
      </div>
    );
  }
  if (service === 'adguard') {
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
          <button type="button" data-config-save="adguard" disabled={!mutationEnabled} onClick={() => void onSave('adguard')}>Set AdGuard</button>
        </div>
      </div>
    );
  }
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
        <button type="button" data-config-save="cloudflare" disabled={!mutationEnabled} onClick={() => void onSave('cloudflare')}>Set Cloudflare</button>
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
