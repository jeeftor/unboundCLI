import {
  Activity,
  CheckCircle2,
  Cloud,
  Database,
  FileSliders,
  Gauge,
  Globe2,
  HardDrive,
  Loader2,
  Network,
  Play,
  RefreshCw,
  Search,
  ShieldCheck,
  SlidersHorizontal,
  TerminalSquare,
  WifiOff,
  Zap
} from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { Dispatch, ReactNode, SetStateAction } from 'react';
import type {
  ApplyResponse,
  ConfigResponse,
  ConfigServiceSummary,
  ConfigTestResponse,
  EntriesResponse,
  Entry,
  PlanResponse,
  ServiceKey,
  SyncAction
} from './types';

const serviceOrder: ServiceKey[] = ['caddy', 'unbound', 'adguard', 'dhcp', 'cloudflare'];

const serviceMeta: Record<ServiceKey, { label: string; icon: typeof Activity; tone: string }> = {
  caddy: { label: 'Caddy', icon: Network, tone: 'green' },
  unbound: { label: 'Unbound', icon: Database, tone: 'blue' },
  adguard: { label: 'AdGuard', icon: ShieldCheck, tone: 'teal' },
  dhcp: { label: 'DHCP', icon: HardDrive, tone: 'yellow' },
  cloudflare: { label: 'Cloudflare', icon: Cloud, tone: 'violet' }
};

type ConfigForms = {
  unbound: { base_url: string; api_key: string; api_secret: string; insecure: boolean };
  adguard: { enabled: boolean; base_url: string; username: string; password: string; insecure: boolean };
  cloudflare: {
    enabled: boolean;
    api_token: string;
    account_id: string;
    zone_id: string;
    tunnel_id: string;
    caddy_service_url: string;
    insecure: boolean;
  };
};

type TestResults = Partial<Record<ServiceKey, { text: string; kind: 'info' | 'ok' | 'error' }>>;

const emptyForms: ConfigForms = {
  unbound: { base_url: '', api_key: '', api_secret: '', insecure: false },
  adguard: { enabled: false, base_url: '', username: '', password: '', insecure: false },
  cloudflare: {
    enabled: false,
    api_token: '',
    account_id: '',
    zone_id: '',
    tunnel_id: '',
    caddy_service_url: '',
    insecure: false
  }
};

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path);
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || response.statusText);
  return data as T;
}

async function postJSON<T>(path: string, payload: unknown): Promise<T> {
  const response = await fetch(path, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-UnboundCLI-Token': window.UNBOUNDCLI_WEB_CONFIG?.applyToken || ''
    },
    body: JSON.stringify(payload)
  });
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || response.statusText);
  return data as T;
}

export function App() {
  const [config, setConfig] = useState<ConfigResponse | null>(null);
  const [entries, setEntries] = useState<Entry[]>([]);
  const [report, setReport] = useState<EntriesResponse['report']>({});
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState('Loading service status...');
  const [messageKind, setMessageKind] = useState<'info' | 'error' | 'ok'>('info');
  const [statusFilter, setStatusFilter] = useState('all');
  const [serviceFilter, setServiceFilter] = useState('all');
  const [search, setSearch] = useState('');
  const [selectedHostname, setSelectedHostname] = useState('');
  const [syncService, setSyncService] = useState('all');
  const [syncLoading, setSyncLoading] = useState(false);
  const [syncProgress, setSyncProgress] = useState({ title: 'Planning sync', detail: 'Building a server-issued action plan...' });
  const [syncLog, setSyncLog] = useState('No actions planned.');
  const [plannedActions, setPlannedActions] = useState<SyncAction[]>([]);
  const [plannedActionIDs, setPlannedActionIDs] = useState<string[]>([]);
  const [plannedPlanID, setPlannedPlanID] = useState('');
  const [plannedService, setPlannedService] = useState('');
  const [plannedHostname, setPlannedHostname] = useState('');
  const [forms, setForms] = useState<ConfigForms>(emptyForms);
  const [configOpen, setConfigOpen] = useState(false);
  const [configStatus, setConfigStatus] = useState('');
  const [configStatusKind, setConfigStatusKind] = useState<'info' | 'error' | 'ok'>('info');
  const [testResults, setTestResults] = useState<TestResults>({});
  const [mobile, setMobile] = useState(false);
  const [tableScrolls, setTableScrolls] = useState(false);
  const entriesPanelRef = useRef<HTMLElement | null>(null);
  const e2eRan = useRef(false);
  const formsRef = useRef(forms);
  const syncServiceRef = useRef(syncService);
  const planRef = useRef({
    actions: [] as SyncAction[],
    actionIDs: [] as string[],
    planID: '',
    service: '',
    hostname: ''
  });

  const mutationEnabled = Boolean(config?.mutation_enabled && window.UNBOUNDCLI_WEB_CONFIG?.mutationEnabled);
  const enabledServices: Partial<Record<ServiceKey, boolean>> = config?.enabled || {};

  const shouldHoldLoadingForE2E = useCallback(() => {
    if (window.UNBOUNDCLI_TEST_HOOKS !== true) return false;
    const script = new URLSearchParams(window.location.search).get('e2e') || '';
    return script.split(',').includes('holdloading');
  }, []);

  const applyConfig = useCallback((nextConfig: ConfigResponse) => {
    setConfig(nextConfig);
    setForms((current) => ({
      unbound: {
        ...current.unbound,
        base_url: nextConfig.summary.unbound?.endpoint || '',
        insecure: Boolean(nextConfig.summary.unbound?.insecure)
      },
      adguard: {
        ...current.adguard,
        enabled: Boolean(nextConfig.summary.adguard?.enabled),
        base_url: nextConfig.summary.adguard?.endpoint || '',
        insecure: Boolean(nextConfig.summary.adguard?.insecure)
      },
      cloudflare: {
        ...current.cloudflare,
        enabled: Boolean(nextConfig.summary.cloudflare?.enabled),
        caddy_service_url: nextConfig.summary.cloudflare?.details?.caddy_service_url || '',
        insecure: Boolean(nextConfig.summary.cloudflare?.insecure)
      }
    }));
    const unboundAvailable = nextConfig.enabled?.unbound !== false;
    const adguardAvailable = nextConfig.enabled?.adguard !== false;
    if (!unboundAvailable && !adguardAvailable && syncService === 'all') {
      syncServiceRef.current = 'dhcp';
      setSyncService('dhcp');
    }
  }, [syncService]);

  const refreshEntries = useCallback(async () => {
    setLoading(true);
    setMessage('Loading service status...');
    setMessageKind('info');
    try {
      const [nextConfig, data] = await Promise.all([
        getJSON<ConfigResponse>('/api/config'),
        getJSON<EntriesResponse>('/api/entries')
      ]);
      applyConfig(nextConfig);
      setEntries(data.entries || []);
      setReport(data.report || {});
      setMessage((data.entries || []).length ? 'Loaded service status.' : 'No entries found.');
      setMessageKind('info');
    } catch (err) {
      const text = err instanceof Error ? err.message : String(err);
      setMessage(text);
      setMessageKind('error');
    } finally {
      if (!shouldHoldLoadingForE2E()) setLoading(false);
    }
  }, [applyConfig, shouldHoldLoadingForE2E]);

  useEffect(() => {
    void refreshEntries();
  }, [refreshEntries]);

  const filteredEntries = useMemo(() => entries.filter((entry) => {
    if (statusFilter === 'out_of_sync' && entry.overall_status !== 2) return false;
    if (statusFilter === 'caddy_only' && entry.overall_status !== 3) return false;
    if (statusFilter === 'stale' && entry.overall_status !== 4) return false;
    if (serviceFilter === 'unbound' && !entry.unbound_status?.configured) return false;
    if (serviceFilter === 'adguard' && !entry.adguard_status?.configured) return false;
    if (serviceFilter === 'cloudflare' && !entry.cloudflare_status?.configured) return false;
    return !search.trim() || entry.hostname.toLowerCase().includes(search.trim().toLowerCase());
  }), [entries, search, serviceFilter, statusFilter]);

  useEffect(() => {
    if (selectedHostname && filteredEntries.some((entry) => entry.hostname === selectedHostname)) return;
    setSelectedHostname(filteredEntries[0]?.hostname || '');
  }, [filteredEntries, selectedHostname]);

  const selectedEntry = useMemo(
    () => entries.find((entry) => entry.hostname === selectedHostname),
    [entries, selectedHostname]
  );

  const summary = useMemo(() => ({
    entries: filteredEntries.length,
    out: filteredEntries.filter((entry) => entry.overall_status === 2).length,
    caddyOnly: filteredEntries.filter((entry) => entry.overall_status === 3).length,
    stale: filteredEntries.filter((entry) => entry.overall_status === 4).length,
    cloudflare: filteredEntries.filter((entry) => entry.cloudflare_status?.configured).length
  }), [filteredEntries]);

  const canSyncNow = mutationEnabled && plannedPlanID !== '' && plannedActionIDs.length > 0;
  const unboundAvailable = enabledServices.unbound !== false;
  const adguardAvailable = enabledServices.adguard !== false;

  useEffect(() => {
    formsRef.current = forms;
  }, [forms]);

  useEffect(() => {
    syncServiceRef.current = syncService;
  }, [syncService]);

  const clearPlannedActions = useCallback(() => {
    planRef.current = { actions: [], actionIDs: [], planID: '', service: '', hostname: '' };
    setPlannedActions([]);
    setPlannedActionIDs([]);
    setPlannedPlanID('');
    setPlannedService('');
    setPlannedHostname('');
  }, []);

  const renderActions = useCallback((actions: SyncAction[], title = 'Planned actions') => {
    if (!actions.length) {
      setSyncLog('No actions needed.');
      return;
    }
    const lines = actions.map((action) => {
      const target = action.new_ip ? ` -> ${action.new_ip}` : '';
      return `${action.type.toUpperCase()} ${action.service} ${action.hostname}${target}`;
    });
    setSyncLog(`${title}\n${lines.join('\n')}`);
  }, []);

  const previewSync = useCallback(async (service = syncService, hostname = '') => {
    if (service === 'dhcp') {
      clearPlannedActions();
      setSyncLog('DHCP apply is not implemented; preview only.');
      return;
    }
    setSyncLoading(true);
    setSyncProgress({
      title: hostname ? 'Planning selected host' : 'Planning sync',
      detail: hostname ? `Checking available actions for ${hostname}...` : 'Checking Caddy entries against available DNS targets...'
    });
    setSyncLog(hostname ? `Planning ${service} sync for ${hostname}...` : `Planning ${service} sync...`);
    try {
      const data = await getJSON<PlanResponse>(`/api/sync/plan?service=${encodeURIComponent(service)}`);
      const paired = (data.actions || []).map((action, index) => ({ action, id: data.action_ids?.[index] }));
      const selected = paired.filter((item) => !hostname || item.action.hostname === hostname);
      const nextActions = selected.map((item) => item.action);
      const nextActionIDs = selected.map((item) => item.id).filter(Boolean) as string[];
      planRef.current = {
        actions: nextActions,
        actionIDs: nextActionIDs,
        planID: data.plan_id || '',
        service,
        hostname
      };
      setPlannedActions(nextActions);
      setPlannedActionIDs(nextActionIDs);
      setPlannedPlanID(data.plan_id || '');
      setPlannedService(service);
      setPlannedHostname(hostname);
      renderActions(nextActions, hostname ? `Planned actions for ${hostname}` : 'Planned actions');
    } finally {
      setSyncLoading(false);
    }
  }, [clearPlannedActions, renderActions, syncService]);

  const dryRunSync = useCallback(async () => {
    const plan = planRef.current;
    const activeService = plan.service || syncService;
    if (activeService === 'dhcp') {
      setSyncLog('DHCP apply is not implemented.');
      return;
    }
    if (!plan.actions.length) {
      setSyncLog('Preview sync before running a dry run.');
      return;
    }
    if (!plan.hostname && plan.service !== syncServiceRef.current) {
      clearPlannedActions();
      setSyncLog('Preview sync again for the selected target.');
      return;
    }
    const data = await postJSON<ApplyResponse>('/api/sync/apply', { dry_run: true, actions: plan.actions });
    setSyncLog(`${data.result.message}\nadded=${data.result.items_added} updated=${data.result.items_updated} deleted=${data.result.items_deleted}`);
  }, [clearPlannedActions, syncService]);

  const syncNow = useCallback(async () => {
    const plan = planRef.current;
    if (!mutationEnabled || !plan.planID || !plan.actionIDs.length) {
      setSyncLog(mutationEnabled ? 'Preview sync before syncing.' : 'Sync is unavailable for this web session.');
      return;
    }
    setSyncLoading(true);
    setSyncProgress({
      title: plan.hostname ? 'Syncing selected host' : 'Applying sync plan',
      detail: plan.hostname ? `Applying DNS updates for ${plan.hostname}...` : 'Applying server-issued DNS updates...'
    });
    setSyncLog(plan.hostname ? `Syncing ${plan.hostname}...` : 'Syncing planned actions...');
    try {
      const data = await postJSON<ApplyResponse>('/api/sync/apply', {
        dry_run: false,
        plan_id: plan.planID,
        action_ids: plan.actionIDs
      });
      setSyncLog(`${data.result.message}\nadded=${data.result.items_added} updated=${data.result.items_updated} deleted=${data.result.items_deleted}`);
      clearPlannedActions();
      await refreshEntries();
    } finally {
      setSyncLoading(false);
    }
  }, [clearPlannedActions, mutationEnabled, refreshEntries]);

  const saveConfig = useCallback(async (service: 'unbound' | 'adguard' | 'cloudflare', nextForms = formsRef.current) => {
    if (!mutationEnabled) {
      setConfigStatus('Config changes are unavailable for this web session.');
      setConfigStatusKind('error');
      return;
    }
    setConfigStatus(`Saving ${service} config...`);
    setConfigStatusKind('info');
    const payload = buildConfigUpdate(service, nextForms);
    const nextConfig = await postJSON<ConfigResponse>('/api/config', payload);
    applyConfig(nextConfig);
    setConfigStatus(`Saved ${service} config.`);
    setConfigStatusKind('ok');
  }, [applyConfig, mutationEnabled]);

  const testConfig = useCallback(async (service: ServiceKey) => {
    if (!mutationEnabled) {
      setConfigStatus('Config tests are unavailable for this web session.');
      setConfigStatusKind('error');
      return;
    }
    setConfigStatus(`Testing ${service} config...`);
    setConfigStatusKind('info');
    setTestResults((current) => ({ ...current, [service]: { text: 'Testing connection...', kind: 'info' } }));
    const result = await postJSON<ConfigTestResponse>('/api/config/test', { service });
    const detail = formatTestDetails(result.details || {});
    const text = `${result.message}${detail ? ` ${detail}` : ''}`;
    const kind = result.success ? 'ok' : 'error';
    setTestResults((current) => ({ ...current, [service]: { text, kind } }));
    setConfigStatus(result.message);
    setConfigStatusKind(kind);
  }, [mutationEnabled]);

  useEffect(() => {
    const updateResponsive = () => {
      setMobile(window.innerWidth <= 760);
      const panel = entriesPanelRef.current;
      setTableScrolls(Boolean(panel && panel.scrollWidth > panel.clientWidth));
    };
    updateResponsive();
    window.addEventListener('resize', updateResponsive);
    return () => window.removeEventListener('resize', updateResponsive);
  }, [entries, filteredEntries]);

  useEffect(() => {
    if (!config || e2eRan.current || window.UNBOUNDCLI_TEST_HOOKS !== true) return;
    e2eRan.current = true;
    const script = new URLSearchParams(window.location.search).get('e2e');
    if (!script) return;
    const run = async () => {
      for (const action of script.split(',')) {
        const [name, ...parts] = action.split(':');
        const value = parts.join(':');
        if (name === 'filter') setStatusFilter(value);
        if (name === 'search') setSearch(value);
        if (name === 'preview') {
          syncServiceRef.current = value;
          setSyncService(value);
          await previewSync(value);
        }
        if (name === 'rowpreview') {
          const [hostname, service = 'unbound'] = parts;
          setSelectedHostname(hostname);
          await previewSync(service, hostname);
        }
        if (name === 'dryrun') await dryRunSync();
        if (name === 'sync') await syncNow();
        if (name === 'toggleconfig') setConfigOpen(value !== 'closed');
        if (name === 'setconfig' && value === 'unbound') {
          const nextForms = {
            ...formsRef.current,
            unbound: { ...formsRef.current.unbound, base_url: 'https://saved.example.test', api_key: 'saved-key' }
          };
          formsRef.current = nextForms;
          setForms(nextForms);
          await saveConfig('unbound', nextForms);
        }
        if (name === 'testconfig') await testConfig(value as ServiceKey);
      }
      document.getElementById('app')?.setAttribute('data-e2e', 'done');
    };
    void run();
  }, [config, dryRunSync, previewSync, saveConfig, syncNow, testConfig]);

  return (
    <div
      id="app"
      data-loading={String(loading)}
      data-preview-loading={String(syncLoading)}
      data-dry-run-enabled={String(plannedActions.length > 0)}
      data-sync-enabled={String(canSyncNow)}
      data-unbound-enabled={String(unboundAvailable)}
      data-adguard-enabled={String(adguardAvailable)}
      data-mutation-enabled={String(mutationEnabled)}
      data-mobile={String(mobile)}
      data-table-scrolls={String(tableScrolls)}
    >
      <header className="topbar">
        <div className="brand-lockup">
          <div className="brand-mark"><Zap size={18} /></div>
          <div>
            <h1>Caddy DNS Sync</h1>
            <div id="runtime" className="runtime">{config ? `Caddy ${config.caddy.server_ip}:${config.caddy.server_port}` : 'Loading runtime...'}</div>
          </div>
        </div>
        <div className="top-actions">
          <button id="refresh" type="button" onClick={() => void refreshEntries()} disabled={loading}>
            <RefreshCw size={16} /> Refresh
          </button>
        </div>
      </header>

      <ProgressBanner loading={loading} />

      <main className="dashboard-shell">
        <aside className="side-nav">
          <NavItem icon={Gauge} label="Overview" active />
          <NavItem icon={FileSliders} label="Config" />
          <NavItem icon={TerminalSquare} label="Sync" />
        </aside>

        <section className="main-stage">
          <section className="hero-panel">
            <div>
              <p className="eyebrow">Local DNS operations</p>
              <h2>Service state, sync plans, and config checks in one workspace.</h2>
            </div>
            <div id="message" className={`message ${messageKind}`} aria-live="polite">{message}</div>
          </section>

          <section id="service-health" className="service-health" aria-label="Service health">
            {serviceOrder.map((service) => (
              <ServiceCard key={service} service={service} config={config} report={report} />
            ))}
          </section>

          <details id="config-panel" className="config-summary panel" open={configOpen} onToggle={(event) => setConfigOpen(event.currentTarget.open)}>
            <summary>
              <span>Configuration</span>
              <small>Runtime config</small>
            </summary>
            {config && (
              <ConfigWorkspace
                config={config}
                forms={forms}
                setForms={setForms}
                mutationEnabled={mutationEnabled}
                status={configStatus}
                statusKind={configStatusKind}
                testResults={testResults}
                onSave={saveConfig}
                onTest={testConfig}
              />
            )}
          </details>

          <section id="summary" className="summary" aria-live="polite">
            <Metric className="neutral" value={summary.entries} label="entries" />
            <Metric className="bad" value={summary.out} label="out of sync" />
            <Metric className="warn" value={summary.caddyOnly} label="caddy only" />
            <Metric className="bad" value={summary.stale} label="stale" />
            <Metric className="cf" value={summary.cloudflare} label="cloudflare" />
          </section>

          <section className="workspace">
            <div className="entries-workbench">
              <section className="toolbar panel">
                <div className="toolbar-title">
                  <strong>Entries</strong>
                  <span>Filter and select a hostname to inspect.</span>
                </div>
                <Select value={statusFilter} onChange={setStatusFilter} ariaLabel="Status filter" options={[
                  ['all', 'All entries'],
                  ['out_of_sync', 'Out of sync'],
                  ['caddy_only', 'Caddy only'],
                  ['stale', 'Stale']
                ]} />
                <Select value={serviceFilter} onChange={setServiceFilter} ariaLabel="Service filter" options={[
                  ['all', 'All services'],
                  ['unbound', 'Unbound'],
                  ['adguard', 'AdGuard'],
                  ['cloudflare', 'Cloudflare']
                ]} disabledValues={new Set(serviceOrder.filter((service) => service !== 'caddy' && enabledServices[service] === false))} />
                <label className="field-label" htmlFor="search">Search</label>
                <div className="search-box">
                  <Search size={15} />
                  <input id="search" type="search" aria-label="Search hostnames" placeholder="Search hostnames" value={search} onChange={(event) => setSearch(event.target.value)} />
                </div>
              </section>
              <section id="entries-panel" ref={entriesPanelRef} className="panel entries-panel">
                <EntriesTable
                  entries={filteredEntries}
                  selectedHostname={selectedHostname}
                  mutationEnabled={mutationEnabled}
                  onSelect={setSelectedHostname}
                  onPreview={previewSync}
                  onSync={async (service, hostname) => {
                    setSelectedHostname(hostname);
                    await previewSync(service, hostname);
                    await syncNow();
                  }}
                />
              </section>
            </div>

            <aside className="action-rail">
              <section id="sync-panel" className="panel sync-panel">
                <div className="rail-heading">
                  <strong>Sync workbench</strong>
                  <span>Preview before applying changes.</span>
                </div>
                <div className="sync-toolbar">
                  <Select value={syncService} onChange={(value) => {
                    syncServiceRef.current = value;
                    setSyncService(value);
                    clearPlannedActions();
                    setSyncLog('No actions planned.');
                  }} ariaLabel="Sync target" options={[
                    ['all', unboundAvailable && adguardAvailable ? 'Unbound + AdGuard' : 'Available DNS targets'],
                    ['unbound', 'Unbound'],
                    ['adguard', 'AdGuard'],
                    ['dhcp', 'DHCP preview']
                  ]} disabledValues={new Set([
                    ...(!unboundAvailable && !adguardAvailable ? ['all'] : []),
                    ...(!unboundAvailable ? ['unbound'] : []),
                    ...(!adguardAvailable ? ['adguard'] : [])
                  ])} idForAll="sync-service-all" />
                  <button id="preview-sync" type="button" onClick={() => void previewSync()} disabled={syncLoading}>
                    <Play size={15} /> Preview sync
                  </button>
                  <button id="dry-run-sync" type="button" onClick={() => void dryRunSync()} disabled={syncLoading || !plannedActions.length}>Dry-run sync</button>
                  <button id="sync-now" type="button" onClick={() => void syncNow()} disabled={syncLoading || !canSyncNow} title={mutationEnabled ? 'Apply the selected server-issued sync plan' : 'Real sync is unavailable for this web session'}>Sync now</button>
                </div>
                <InlineProgress loading={syncLoading} title={syncProgress.title} detail={syncProgress.detail} />
                <div id="sync-log" className="log" role="status" aria-live="polite">{syncLog}</div>
              </section>
              <HostInspector entry={selectedEntry} mutationEnabled={mutationEnabled} onPreview={(hostname) => previewSync(syncService, hostname)} onSync={async (hostname) => {
                await previewSync(syncService, hostname);
                await syncNow();
              }} />
            </aside>
          </section>
        </section>
      </main>
    </div>
  );
}

function NavItem({ icon: Icon, label, active = false }: { icon: typeof Activity; label: string; active?: boolean }) {
  return <div className={`nav-item ${active ? 'active' : ''}`}><Icon size={17} /><span>{label}</span></div>;
}

function ProgressBanner({ loading }: { loading: boolean }) {
  return (
    <div id="top-progress" className="top-progress" role="progressbar" aria-live="polite" aria-label="Loading service status" hidden={!loading}>
      <div className="progress-copy">
        <strong id="top-progress-title">Loading service status</strong>
        <span id="top-progress-detail">Reading Caddy, DNS targets, and runtime config...</span>
      </div>
      <div className="progress-track"><span /></div>
    </div>
  );
}

function InlineProgress({ loading, title, detail }: { loading: boolean; title: string; detail: string }) {
  return (
    <div id="sync-progress" className="inline-progress" role="progressbar" aria-live="polite" aria-label={title} hidden={!loading}>
      <div className="progress-copy compact">
        <strong id="sync-progress-title">{title}</strong>
        <span id="sync-progress-detail">{detail}</span>
      </div>
      <div className="progress-track"><span /></div>
    </div>
  );
}

function ServiceCard({ service, config, report }: { service: ServiceKey; config: ConfigResponse | null; report: EntriesResponse['report'] }) {
  const meta = serviceMeta[service];
  const Icon = meta.icon;
  const enabled = config?.enabled?.[service] !== false;
  const count = report.services?.[service]?.count;
  return (
    <div className={`service-card ${enabled ? 'ok' : 'missing'} ${meta.tone}`}>
      <span><Icon size={14} /> {meta.label}</span>
      <strong>{typeof count === 'number' ? `${count} loaded` : enabled ? 'configured' : 'not configured'}</strong>
    </div>
  );
}

function Metric({ value, label, className }: { value: number; label: string; className: string }) {
  return <span className={`metric ${className}`}><strong>{value}</strong>{label}</span>;
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
      <div className="panel-heading">
        <div>
          <strong>Configuration</strong>
          <span>Runtime source and save destination.</span>
        </div>
        <span className="save-target">Save target: {props.config.save_target || '-'}</span>
      </div>
      <div id="config-status" className={`config-status ${props.statusKind}`} role="status" aria-live="polite">{props.status}</div>
      <div className="config-grid">
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
  const fields = Object.entries(summary.fields || {});
  const details = Object.entries(summary.details || {}).filter(([, value]) => value);
  const missing = summary.missing || [];
  const tone = summary.client_ready ? 'ok' : summary.enabled ? 'warn' : 'missing';
  return (
    <article className={`config-card ${tone}`}>
      <header>
        <strong>{summary.label || serviceMeta[service].label}</strong>
        <span>{summary.client_ready ? 'Client ready' : summary.enabled ? 'Configured, incomplete' : 'Not configured'}</span>
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

function EntriesTable({
  entries,
  selectedHostname,
  mutationEnabled,
  onSelect,
  onPreview,
  onSync
}: {
  entries: Entry[];
  selectedHostname: string;
  mutationEnabled: boolean;
  onSelect: (hostname: string) => void;
  onPreview: (service: string, hostname: string) => Promise<void>;
  onSync: (service: string, hostname: string) => Promise<void>;
}) {
  const [rowServices, setRowServices] = useState<Record<string, string>>({});
  return (
    <table>
      <thead>
        <tr>
          <th>Hostname</th><th>Status</th><th>Services</th><th>Caddy</th><th>DNS</th><th>Cloudflare</th><th>Actions</th>
        </tr>
      </thead>
      <tbody id="entries">
        {entries.map((entry) => {
          const rowService = rowServices[entry.hostname] || 'all';
          return (
            <tr key={entry.hostname} data-hostname={entry.hostname} className={entry.hostname === selectedHostname ? 'selected-row' : ''} onClick={() => onSelect(entry.hostname)}>
              <td><strong>{entry.hostname}</strong><span className="subtle">{entry.data_source || '-'}</span></td>
              <td><StatusChip entry={entry} /></td>
              <td><ServiceBadges entry={entry} /></td>
              <td><span>{entry.caddy_upstream || '-'}</span><span className="subtle">admin {entry.caddy_ip || '-'}</span></td>
              <td><span className={`dns-result ${dnsResultClass(entry.dns_resolved)}`}>{entry.dns_resolved || 'FAIL'}</span></td>
              <td><CloudflareDetails status={entry.cloudflare_status} /></td>
              <td>
                <div className="row-actions">
                  <select className="row-sync-service" aria-label={`Sync target for ${entry.hostname}`} value={rowService} onClick={(event) => event.stopPropagation()} onChange={(event) => setRowServices((current) => ({ ...current, [entry.hostname]: event.target.value }))}>
                    <option value="all">Available DNS</option>
                    <option value="unbound">Unbound</option>
                    <option value="adguard">AdGuard</option>
                    <option value="dhcp">DHCP preview</option>
                  </select>
                  <button className="row-preview" type="button" data-hostname={entry.hostname} onClick={(event) => { event.stopPropagation(); onSelect(entry.hostname); void onPreview(rowService, entry.hostname); }}>Preview sync</button>
                  <button className="row-sync" type="button" data-hostname={entry.hostname} disabled={!mutationEnabled} title={mutationEnabled ? 'Apply the selected server-issued sync plan' : 'Real sync is unavailable for this web session'} onClick={(event) => { event.stopPropagation(); onSelect(entry.hostname); void onSync(rowService, entry.hostname); }}>Sync</button>
                </div>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

function StatusChip({ entry }: { entry: Entry }) {
  return <span className={`status-chip ${statusClassByCode(entry.overall_status)}`}>{entry.status_label || 'Unknown'}</span>;
}

function ServiceBadges({ entry }: { entry: Entry }) {
  return (
    <>
      <ServiceBadge name="Unbound" status={entry.unbound_status} />
      <ServiceBadge name="AdGuard" status={entry.adguard_status} />
      <ServiceBadge name="DHCP" status={entry.dhcp_status} />
    </>
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
  if (!status?.configured) return <span className="cloudflare-detail missing">Not routed</span>;
  return (
    <span className={`cloudflare-detail ${status.http_host_header ? 'ok' : 'bad'}`}>
      <strong>{status.tunnel_name || 'Tunnel'}</strong>
      <span>{status.service || '-'}</span>
      <span>{status.http_host_header ? `Host header ${status.http_host_header}` : 'Missing HTTPHostHeader'}</span>
      <span>{status.has_access_policy ? 'Access policy' : 'No access policy'}</span>
    </span>
  );
}

function HostInspector({ entry, mutationEnabled, onPreview, onSync }: { entry?: Entry; mutationEnabled: boolean; onPreview: (hostname: string) => Promise<void>; onSync: (hostname: string) => Promise<void> }) {
  if (!entry) {
    return (
      <section id="host-inspector" className="panel inspector" aria-live="polite">
        <div className="rail-heading"><strong>Host inspector</strong><span>Select a row to inspect service state.</span></div>
        <div className="empty-state"><WifiOff size={18} /> No hostname selected.</div>
      </section>
    );
  }
  return (
    <section id="host-inspector" className="panel inspector" aria-live="polite">
      <div className="rail-heading"><strong>{entry.hostname}</strong><span>{entry.status_label || 'Unknown'}</span></div>
      <div className="inspector-status">
        <StatusChip entry={entry} />
        <span className={`dns-result ${dnsResultClass(entry.dns_resolved)}`}>{entry.dns_resolved || 'FAIL'}</span>
      </div>
      <div className="inspector-grid">
        <InspectorLine label="Caddy upstream" value={entry.caddy_upstream || '-'} />
        <InspectorLine label="Source" value={entry.data_source || '-'} />
        <InspectorLine label="Unbound" value={serviceStateText(entry.unbound_status)} />
        <InspectorLine label="AdGuard" value={serviceStateText(entry.adguard_status)} />
        <InspectorLine label="DHCP" value={serviceStateText(entry.dhcp_status)} />
        <InspectorLine label="Cloudflare" value={cloudflareStateText(entry.cloudflare_status)} />
      </div>
      <div className="inspector-actions">
        <button type="button" id="inspector-preview" onClick={() => void onPreview(entry.hostname)}>Preview selected</button>
        <button type="button" id="inspector-sync" disabled={!mutationEnabled} onClick={() => void onSync(entry.hostname)}>Sync selected</button>
      </div>
    </section>
  );
}

function InspectorLine({ label, value }: { label: string; value: string }) {
  return <div className="inspector-line"><span>{label}</span><strong>{value}</strong></div>;
}

function Select({ value, onChange, options, ariaLabel, disabledValues = new Set<string>(), idForAll }: { value: string; onChange: (value: string) => void; options: [string, string][]; ariaLabel: string; disabledValues?: Set<string>; idForAll?: string }) {
  return (
    <select value={value} aria-label={ariaLabel} onChange={(event) => onChange(event.target.value)}>
      {options.map(([optionValue, label]) => (
        <option key={optionValue} id={idForAll && optionValue === 'all' ? idForAll : undefined} value={optionValue} data-service={optionValue !== 'all' ? optionValue : undefined} disabled={disabledValues.has(optionValue)} className={disabledValues.has(optionValue) ? 'disabled-service' : undefined}>{label}</option>
      ))}
    </select>
  );
}

function buildConfigUpdate(service: 'unbound' | 'adguard' | 'cloudflare', forms: ConfigForms) {
  if (service === 'unbound') {
    return pruneUndefined({ unbound: { ...forms.unbound } });
  }
  if (service === 'adguard') {
    return pruneUndefined({ adguard: { ...forms.adguard } });
  }
  return pruneUndefined({ cloudflare: { ...forms.cloudflare } });
}

function pruneUndefined(value: Record<string, unknown>): Record<string, unknown> {
  Object.keys(value).forEach((key) => {
    const child = value[key];
    if (child === undefined || child === '') {
      delete value[key];
    } else if (child && typeof child === 'object' && !Array.isArray(child)) {
      pruneUndefined(child as Record<string, unknown>);
    }
  });
  return value;
}

function sourceLabel(source?: { label?: string; kind?: string; path?: string }) {
  const label = source?.label || source?.kind || 'Unknown';
  return source?.path ? `${label}: ${source.path}` : label;
}

function formatConfigKey(key: string) {
  return key.replace(/_/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase())
    .replace(/\bApi\b/g, 'API')
    .replace(/\bUrl\b/g, 'URL')
    .replace(/\bTls\b/g, 'TLS')
    .replace(/\bDhcp\b/g, 'DHCP')
    .replace(/\bId\b/g, 'ID');
}

function formatTestDetails(details: Record<string, string>) {
  return Object.entries(details)
    .filter(([, value]) => value !== '')
    .map(([key, value]) => `${formatConfigKey(key)}: ${value}`)
    .join(', ');
}

function statusClassByCode(status: number) {
  if (status === 0 || status === 1) return 'ok';
  if (status === 2 || status === 4 || status === 5) return 'bad';
  return 'warn';
}

function dnsResultClass(value: string) {
  const normalized = String(value || '').toLowerCase();
  return normalized && normalized !== 'fail' ? 'ok' : 'bad';
}

function serviceStateText(status: { configured: boolean; in_sync: boolean; ip: string }) {
  if (!status?.configured) return 'Missing';
  if (status.in_sync) return status.ip ? `In sync (${status.ip})` : 'In sync';
  return status.ip ? `Mismatch (${status.ip})` : 'Mismatch';
}

function cloudflareStateText(status: Entry['cloudflare_status']) {
  if (!status?.configured) return 'Not routed';
  if (status.http_host_header) return status.service || 'Routed';
  return 'Missing host header';
}
