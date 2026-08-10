import { create } from 'zustand';
import { api } from './api/client';
import { isIssue } from './lib/hostnameDecision';
import { renderActionLine, serviceEnabled, formatTestDetails } from './lib/services';
import type {
  ConfigResponse,
  EntriesResponse,
  Entry,
  HostAuth,
  ProgressEvent,
  ServiceKey,
  SyncAction,
} from './types';
import { emptyForms, type ConfigForms, type TestResults } from './hooks/useConfigForms';

// ─── Types ──────────────────────────────────────────────────────────────────

type PlanState = {
  actions: SyncAction[];
  actionIDs: string[];
  planID: string;
  service: string;
  hostname: string;
};

const emptyPlan: PlanState = { actions: [], actionIDs: [], planID: '', service: '', hostname: '' };

type GitRemoteStatus = {
  remote_ahead: number;
  local_ahead: number;
  branch: string;
  remote: string;
  fetch_error?: string;
};

type Summary = {
  entries: number;
  inSync: number;
  out: number;
  caddyOnly: number;
  stale: number;
  cloudflare: number;
  issues: number;
};

// ─── Store ───────────────────────────────────────────────────────────────────

type AppState = {
  // ── View / UI ──
  view: string;
  configOpen: boolean;
  mobile: boolean;
  tableScrolls: boolean;
  modalOpen: boolean;
  modalHostname: string;
  modalAutoSync: boolean;
  visualizeOpen: boolean;
  visualizeHostname: string;
  logBarOpen: boolean;

  // ── Runtime data ──
  config: ConfigResponse | null;
  entries: Entry[];
  report: EntriesResponse['report'];
  loading: boolean;
  message: string;
  messageKind: 'info' | 'error' | 'ok';
  progress: Record<string, ProgressEvent>;

  // ── Auth inventory (cached, shared across components) ──
  authHosts: Map<string, HostAuth>;
  authLoading: boolean;
  authError: string | null;
  authSources: { cloudflare_access: boolean; authentik: boolean } | null;

  // ── Filters ──
  statusFilter: string;
  serviceFilter: string;
  search: string;
  selectedHostname: string;
  suppressed: Set<string>;

  // ── Sync ──
  syncService: string;
  syncLoading: boolean;
  syncProgress: { title: string; detail: string };
  syncLog: string;
  plan: PlanState;

  // ── Config forms ──
  forms: ConfigForms;
  savedForms: ConfigForms;
  configStatus: string;
  configStatusKind: 'info' | 'error' | 'ok';
  testResults: TestResults;

  // ── Git remote ──
  remoteStatus: GitRemoteStatus | null;
  remoteChecking: boolean;
  pulling: boolean;
  pullOutput: string;

  // ── Derived (computed via selectors) ──
  mutationEnabled: boolean;

  // ── Actions ──
  setView: (v: string) => void;
  setConfigOpen: (open: boolean) => void;
  setMobile: (m: boolean) => void;
  setTableScrolls: (s: boolean) => void;
  setModalOpen: (open: boolean) => void;
  setModalHostname: (hostname: string) => void;
  setModalAutoSync: (auto: boolean) => void;
  setVisualizeOpen: (open: boolean) => void;
  setVisualizeHostname: (hostname: string) => void;
  setLogBarOpen: (open: boolean) => void;

  // Runtime data actions
  setConfig: (config: ConfigResponse) => void;
  setEntries: (entries: Entry[]) => void;
  setReport: (report: EntriesResponse['report']) => void;
  setLoading: (loading: boolean) => void;
  setMessage: (message: string) => void;
  setMessageKind: (kind: 'info' | 'error' | 'ok') => void;
  setProgress: (progress: Record<string, ProgressEvent>) => void;
  updateProgress: (service: string, ev: ProgressEvent) => void;

  // Auth inventory actions
  setAuthHosts: (hosts: Map<string, HostAuth>) => void;
  setAuthLoading: (loading: boolean) => void;
  setAuthError: (error: string | null) => void;
  setAuthSources: (sources: { cloudflare_access: boolean; authentik: boolean } | null) => void;
  refreshAuth: () => Promise<void>;

  // Filter actions
  setStatusFilter: (filter: string) => void;
  setServiceFilter: (filter: string) => void;
  setSearch: (search: string) => void;
  setSelectedHostname: (hostname: string) => void;
  toggleSuppress: (key: string) => void;

  // Sync actions
  setSyncService: (service: string) => void;
  setSyncLoading: (loading: boolean) => void;
  setSyncProgress: (progress: { title: string; detail: string }) => void;
  setSyncLog: (log: string) => void;
  setPlan: (plan: PlanState) => void;
  clearPlan: (message?: string) => void;

  // Config form actions
  setForms: (forms: ConfigForms) => void;
  setSavedForms: (forms: ConfigForms) => void;
  setConfigStatus: (status: string) => void;
  setConfigStatusKind: (kind: 'info' | 'error' | 'ok') => void;
  setTestResults: (results: TestResults) => void;
  updateTestResult: (service: ServiceKey, result: { text: string; kind: 'info' | 'ok' | 'error' }) => void;

  // Git remote actions
  setRemoteStatus: (status: GitRemoteStatus | null) => void;
  setRemoteChecking: (checking: boolean) => void;
  setPulling: (pulling: boolean) => void;
  setPullOutput: (output: string) => void;
};

export const useStore = create<AppState>((set, _get) => ({
  // ── View / UI ──
  view: 'dashboard',
  configOpen: false,
  mobile: false,
  tableScrolls: false,
  modalOpen: false,
  modalHostname: '',
  modalAutoSync: false,
  visualizeOpen: false,
  visualizeHostname: '',
  logBarOpen: false,

  // ── Runtime data ──
  config: null,
  entries: [],
  report: {},
  loading: true,
  message: 'Loading service status...',
  messageKind: 'info',
  progress: {},

  // Auth inventory
  authHosts: new Map(),
  authLoading: false,
  authError: null,
  authSources: null,

  // ── Filters ──
  statusFilter: 'all',
  serviceFilter: 'all',
  search: '',
  selectedHostname: '',
  suppressed: new Set<string>(),

  // ── Sync ──
  syncService: 'all',
  syncLoading: false,
  syncProgress: { title: 'Planning sync', detail: 'Building a server-issued action plan...' },
  syncLog: 'No actions planned.',
  plan: emptyPlan,

  // ── Config forms ──
  forms: emptyForms,
  savedForms: emptyForms,
  configStatus: '',
  configStatusKind: 'info',
  testResults: {},

  // ── Git remote ──
  remoteStatus: null,
  remoteChecking: false,
  pulling: false,
  pullOutput: '',

  // ── Derived ──
  mutationEnabled: false,

  // ── Actions: UI ──
  setView: (v) => set({ view: v }),
  setConfigOpen: (open) => set({ configOpen: open }),
  setMobile: (m) => set({ mobile: m }),
  setTableScrolls: (s) => set({ tableScrolls: s }),
  setModalOpen: (open) => set({ modalOpen: open }),
  setModalHostname: (hostname) => set({ modalHostname: hostname }),
  setModalAutoSync: (auto) => set({ modalAutoSync: auto }),
  setVisualizeOpen: (open) => set({ visualizeOpen: open }),
  setVisualizeHostname: (hostname) => set({ visualizeHostname: hostname }),
  setLogBarOpen: (open) => set({ logBarOpen: open }),

  // ── Actions: Runtime data ──
  setConfig: (config) => {
    const mutationEnabled = Boolean(
      config?.mutation_enabled && window.UNBOUNDCLI_WEB_CONFIG?.mutationEnabled
    );
    set({ config, mutationEnabled });
  },
  setEntries: (entries) => set({ entries }),
  setReport: (report) => set({ report }),
  setLoading: (loading) => set({ loading }),
  setMessage: (message) => set({ message }),
  setMessageKind: (kind) => set({ messageKind: kind }),
  setProgress: (progress) => set({ progress }),
  updateProgress: (service, ev) =>
    set((state) => ({ progress: { ...state.progress, [service]: ev } })),

  // ── Actions: Auth inventory ──
  setAuthHosts: (hosts) => set({ authHosts: hosts }),
  setAuthLoading: (loading) => set({ authLoading: loading }),
  setAuthError: (error) => set({ authError: error }),
  setAuthSources: (sources) => set({ authSources: sources }),
  refreshAuth: async () => {
    set({ authLoading: true, authError: null });
    try {
      const res = await api.authInventory();
      const map = new Map<string, HostAuth>();
      for (const h of res.hosts) map.set(h.hostname, h);
      set({ authHosts: map, authLoading: false, authSources: res.sources });
    } catch (e) {
      set({ authLoading: false, authError: e instanceof Error ? e.message : String(e) });
    }
  },

  // ── Actions: Filters ──
  setStatusFilter: (filter) => set({ statusFilter: filter }),
  setServiceFilter: (filter) => set({ serviceFilter: filter }),
  setSearch: (search) => set({ search }),
  setSelectedHostname: (hostname) => set({ selectedHostname: hostname }),
  toggleSuppress: (key) =>
    set((state) => {
      const next = new Set(state.suppressed);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return { suppressed: next };
    }),

  // ── Actions: Sync ──
  setSyncService: (service) =>
    set({ syncService: service, plan: emptyPlan, syncLog: 'No actions planned.' }),
  setSyncLoading: (loading) => set({ syncLoading: loading }),
  setSyncProgress: (progress) => set({ syncProgress: progress }),
  setSyncLog: (log) => set({ syncLog: log }),
  setPlan: (plan) => set({ plan }),
  clearPlan: (message) =>
    set({ plan: emptyPlan, syncLog: message ?? 'No actions planned.' }),

  // ── Actions: Config forms ──
  setForms: (forms) => set({ forms }),
  setSavedForms: (forms) => set({ savedForms: forms }),
  setConfigStatus: (status) => set({ configStatus: status }),
  setConfigStatusKind: (kind) => set({ configStatusKind: kind }),
  setTestResults: (results) => set({ testResults: results }),
  updateTestResult: (service, result) =>
    set((state) => ({ testResults: { ...state.testResults, [service]: result } })),

  // ── Actions: Git remote ──
  setRemoteStatus: (status) => set({ remoteStatus: status }),
  setRemoteChecking: (checking) => set({ remoteChecking: checking }),
  setPulling: (pulling) => set({ pulling }),
  setPullOutput: (output) => set({ pullOutput: output }),
}));

// ─── Selectors (derived state) ───────────────────────────────────────────────

export function selectFilteredEntries(state: AppState): Entry[] {
  const { entries, statusFilter, serviceFilter, search, suppressed, config } = state;
  const caddyServerIP = config?.caddy?.server_ip ?? '';
  return entries.filter((entry) => {
    if (statusFilter === 'synced' && entry.overall_status !== 0 && entry.overall_status !== 1) return false;
    if (statusFilter === 'out_of_sync' && entry.overall_status !== 2) return false;
    if (statusFilter === 'caddy_only' && entry.overall_status !== 3) return false;
    if (statusFilter === 'stale' && entry.overall_status !== 4) return false;
    if (statusFilter === 'cloudflare' && !entry.cloudflare_status?.configured) return false;
    if (statusFilter === 'issues' && !isIssue(entry, caddyServerIP, suppressed)) return false;
    if (serviceFilter === 'unbound' && !entry.unbound_status?.configured) return false;
    if (serviceFilter === 'adguard' && !entry.adguard_status?.configured) return false;
    if (serviceFilter === 'dhcp' && !entry.dhcp_status?.configured) return false;
    if (serviceFilter === 'cloudflare' && !entry.cloudflare_status?.configured) return false;
    return !search.trim() || entry.hostname.toLowerCase().includes(search.trim().toLowerCase());
  });
}

export function selectSummary(state: AppState): Summary {
  const { entries, suppressed, config } = state;
  const caddyServerIP = config?.caddy?.server_ip ?? '';
  return {
    entries: entries.length,
    inSync: entries.filter((e) => e.overall_status === 0 || e.overall_status === 1).length,
    out: entries.filter((e) => e.overall_status === 2).length,
    caddyOnly: entries.filter((e) => e.overall_status === 3).length,
    stale: entries.filter((e) => e.overall_status === 4).length,
    cloudflare: entries.filter((e) => e.cloudflare_status?.configured).length,
    issues: entries.filter((e) => isIssue(e, caddyServerIP, suppressed)).length,
  };
}

export function selectSelectedEntry(state: AppState): Entry | undefined {
  return state.entries.find((e) => e.hostname === state.selectedHostname);
}

export function selectCanSyncNow(state: AppState): boolean {
  return state.mutationEnabled && state.plan.planID !== '' && state.plan.actionIDs.length > 0;
}

export function selectPlannedActions(state: AppState): SyncAction[] {
  return state.plan.actions;
}

export function selectEnabledServices(state: AppState): Partial<Record<ServiceKey, boolean>> {
  return state.config?.enabled || {};
}

// ─── Async actions (business logic that was in hooks) ───────────────────────

const CONFIG_CACHE_KEY = 'caddy-dns-sync:config';

function loadCachedConfig(): ConfigResponse | null {
  try {
    const raw = sessionStorage.getItem(CONFIG_CACHE_KEY);
    return raw ? (JSON.parse(raw) as ConfigResponse) : null;
  } catch {
    return null;
  }
}

function saveCachedConfig(cfg: ConfigResponse) {
  try {
    sessionStorage.setItem(CONFIG_CACHE_KEY, JSON.stringify(cfg));
  } catch {
    // non-fatal
  }
}

// Service display metadata for progress indicators.
export const SERVICE_META: Record<string, { label: string; icon: string }> = {
  caddy: { label: 'Caddy', icon: '🌐' },
  unbound: { label: 'Unbound', icon: '📋' },
  adguard: { label: 'AdGuard', icon: '🛡️' },
  dhcp: { label: 'DHCP', icon: '📡' },
  cloudflare: { label: 'Cloudflare', icon: '☁️' },
  dns: { label: 'DNS Resolve', icon: '🔍' },
};

// Runtime data sequence counter (module-level, like the ref in the old hook).
let runtimeSequence = 0;
let eventSource: EventSource | null = null;
let configFetched = false;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectDelay = 1000;
let pendingOnDataChanged: (() => void) | undefined;

function scheduleReconnect() {
  if (reconnectTimer) clearTimeout(reconnectTimer);
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    reconnectDelay = Math.min(reconnectDelay * 2, 30_000);
    void refreshEntries(pendingOnDataChanged);
  }, reconnectDelay);
}

export function cancelReconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  reconnectDelay = 1000;
}

export function refreshEntries(onDataChanged?: () => void) {
  const store = useStore.getState();
  pendingOnDataChanged = onDataChanged;

  // Close any existing SSE connection.
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }

  const requestID = ++runtimeSequence;
  store.setLoading(true);
  store.setProgress({});
  store.setMessage('Loading service status...');
  store.setMessageKind('info');

  // Fetch config in parallel (fast, cached).
  const configPromise = configFetched
    ? Promise.resolve(null)
    : api
        .config()
        .then((cfg) => {
          if (requestID !== runtimeSequence) return null;
          useStore.getState().setConfig(cfg);
          saveCachedConfig(cfg);
          configFetched = true;
          return cfg;
        })
        .catch(() => null);

  // Stream entries via SSE.
  const es = new EventSource('/api/entries/stream');
  eventSource = es;

  es.addEventListener('progress', (e: MessageEvent) => {
    if (requestID !== runtimeSequence) return;
    try {
      const ev = JSON.parse(e.data) as ProgressEvent;
      useStore.getState().updateProgress(ev.service, ev);
      const meta = SERVICE_META[ev.service];
      if (meta) {
        if (ev.status === 'loaded') {
          useStore.getState().setMessage(`Loaded ${meta.label} (${ev.count} entries)`);
        } else if (ev.status === 'failed') {
          useStore.getState().setMessage(`${meta.label} failed: ${ev.error || 'unknown'}`);
        }
      }
    } catch {
      // Ignore malformed progress events.
    }
  });

  es.addEventListener('done', (e: MessageEvent) => {
    if (requestID !== runtimeSequence) return;
    // Reset reconnect delay on successful completion.
    reconnectDelay = 1000;
    try {
      const data = JSON.parse(e.data) as EntriesResponse;
      useStore.getState().setEntries(data.entries || []);
      useStore.getState().setReport(data.report || {});
      onDataChanged?.();
      const entries = data.entries || [];
      useStore.getState().setMessage(entries.length ? 'Loaded service status.' : 'No entries found.');
      useStore.getState().setMessageKind('info');
    } catch {
      useStore.getState().setMessage('Failed to parse entries response');
      useStore.getState().setMessageKind('error');
    } finally {
      if (requestID === runtimeSequence) {
        const shouldHold = window.UNBOUNDCLI_TEST_HOOKS === true &&
          new URLSearchParams(window.location.search).get('e2e')?.split(',').includes('holdloading');
        if (!shouldHold) useStore.getState().setLoading(false);
      }
      es.close();
      eventSource = null;
    }
  });

  es.addEventListener('error', (e: MessageEvent) => {
    if (e.data) {
      try {
        const data = JSON.parse(e.data);
        if (requestID === runtimeSequence) {
          useStore.getState().setMessage(data.error || 'Stream error');
          useStore.getState().setMessageKind('error');
        }
      } catch {
        // Ignore malformed error events.
      }
      return;
    }
    // Native error — connection lost. Auto-reconnect with backoff.
    if (requestID === runtimeSequence) {
      useStore.getState().setMessage(`Connection lost — retrying in ${Math.round(reconnectDelay / 1000)}s...`);
      useStore.getState().setMessageKind('error');
      const shouldHold = window.UNBOUNDCLI_TEST_HOOKS === true &&
        new URLSearchParams(window.location.search).get('e2e')?.split(',').includes('holdloading');
      if (!shouldHold) useStore.getState().setLoading(false);
    }
    es.close();
    eventSource = null;
    // Schedule auto-reconnect unless this request was superseded.
    if (requestID === runtimeSequence) {
      scheduleReconnect();
    }
  });

  void configPromise;
}

// Initialize config from cache on module load.
const cachedConfig = loadCachedConfig();
if (cachedConfig) {
  useStore.getState().setConfig(cachedConfig);
}

// ─── Sync actions ────────────────────────────────────────────────────────────

export async function previewSync(service = useStore.getState().syncService, hostname = ''): Promise<boolean> {
  const store = useStore.getState();
  const enabledServices = store.config?.enabled || {};

  if (service === 'dhcp') {
    store.clearPlan('DHCP apply is not implemented; preview only.');
    return false;
  }
  if (!serviceEnabled(enabledServices, service)) {
    store.clearPlan(`${service} is not available in this web session.`);
    return false;
  }

  store.setSyncLoading(true);
  store.setSyncProgress({
    title: hostname ? 'Planning selected host' : 'Planning sync',
    detail: hostname ? `Checking available actions for ${hostname}...` : 'Checking Caddy entries against available DNS targets...',
  });
  store.setSyncLog(hostname ? `Planning ${service} sync for ${hostname}...` : `Planning ${service} sync...`);

  try {
    const data = await api.planSync(service, hostname);
    const nextPlan: PlanState = {
      actions: data.actions || [],
      actionIDs: data.action_ids || [],
      planID: data.plan_id || '',
      service,
      hostname,
    };
    const actions = nextPlan.actions;
    if (!actions.length) {
      store.setSyncLog('No actions needed.');
    } else {
      store.setSyncLog(`${hostname ? `Planned actions for ${hostname}` : 'Planned actions'}\n${actions.map(renderActionLine).join('\n')}`);
    }
    store.setPlan(nextPlan);
    return nextPlan.actionIDs.length > 0;
  } catch (err) {
    store.clearPlan(err instanceof Error ? err.message : String(err));
    return false;
  } finally {
    store.setSyncLoading(false);
  }
}

export async function dryRunSync(): Promise<void> {
  const store = useStore.getState();
  const activePlan = store.plan;
  const activeService = activePlan.service || store.syncService;

  if (activeService === 'dhcp') {
    store.setSyncLog('DHCP apply is not implemented.');
    return;
  }
  if (!activePlan.actions.length) {
    store.setSyncLog('Preview sync before running a dry run.');
    return;
  }
  if (!activePlan.hostname && activePlan.service !== store.syncService) {
    store.clearPlan('Preview sync again for the selected target.');
    return;
  }

  store.setSyncLoading(true);
  store.setSyncProgress({ title: 'Dry-running plan', detail: 'Simulating changes without writing DNS records...' });
  try {
    const data = await api.applySync({ dry_run: true, actions: activePlan.actions });
    store.setSyncLog(`${data.result.message}\nadded=${data.result.items_added} updated=${data.result.items_updated} deleted=${data.result.items_deleted}`);
  } catch (err) {
    store.setSyncLog(err instanceof Error ? err.message : String(err));
  } finally {
    store.setSyncLoading(false);
  }
}

export async function syncNow(): Promise<void> {
  const store = useStore.getState();
  const activePlan = store.plan;

  if (!store.mutationEnabled || !activePlan.planID || !activePlan.actionIDs.length) {
    store.setSyncLog(store.mutationEnabled ? 'Preview sync before syncing.' : 'Sync is unavailable for this web session.');
    return;
  }

  store.setSyncLoading(true);
  store.setSyncProgress({
    title: activePlan.hostname ? 'Syncing selected host' : 'Applying sync plan',
    detail: activePlan.hostname ? `Applying DNS updates for ${activePlan.hostname}...` : 'Applying server-issued DNS updates...',
  });
  store.setSyncLog(activePlan.hostname ? `Syncing ${activePlan.hostname}...` : 'Syncing planned actions...');

  try {
    const data = await api.applySync({
      dry_run: false,
      plan_id: activePlan.planID,
      action_ids: activePlan.actionIDs,
    });
    store.setSyncLog(`${data.result.message}\nadded=${data.result.items_added} updated=${data.result.items_updated} deleted=${data.result.items_deleted}`);
    store.clearPlan();
    void refreshEntries();
    // Refresh auth cache — backend also refreshes its cache, but this
    // updates the frontend store so the VisualizeModal shows fresh data.
    void useStore.getState().refreshAuth();
  } catch (err) {
    store.setSyncLog(err instanceof Error ? err.message : String(err));
  } finally {
    store.setSyncLoading(false);
  }
}

export async function removeEntry(hostname: string, service: 'all' | 'unbound' | 'adguard' = 'all'): Promise<void> {
  await api.removeEntry(hostname, service);
  void refreshEntries();
  void useStore.getState().refreshAuth();
}

export async function syncAll(): Promise<void> {
  const ok = await previewSync('all', '');
  if (ok) {
    await syncNow();
    void refreshEntries();
    void useStore.getState().refreshAuth();
  }
}

// ─── Config form actions ─────────────────────────────────────────────────────

function buildConfigUpdate(service: 'unbound' | 'adguard' | 'cloudflare', forms: ConfigForms) {
  if (service === 'unbound') {
    return {
      unbound: omitEmptySecretFields({
        base_url: forms.unbound.base_url,
        insecure: forms.unbound.insecure,
        api_key: forms.unbound.api_key,
        api_secret: forms.unbound.api_secret,
      }, ['api_key', 'api_secret']),
    };
  }
  if (service === 'adguard') {
    return {
      adguard: omitEmptySecretFields({
        enabled: forms.adguard.enabled,
        base_url: forms.adguard.base_url,
        insecure: forms.adguard.insecure,
        username: forms.adguard.username,
        password: forms.adguard.password,
      }, ['username', 'password']),
    };
  }
  return {
    cloudflare: omitEmptySecretFields({
      enabled: forms.cloudflare.enabled,
      caddy_service_url: forms.cloudflare.caddy_service_url,
      insecure: forms.cloudflare.insecure,
      api_token: forms.cloudflare.api_token,
      account_id: forms.cloudflare.account_id,
      zone_id: forms.cloudflare.zone_id,
      tunnel_id: forms.cloudflare.tunnel_id,
    }, ['api_token', 'account_id', 'zone_id', 'tunnel_id']),
  };
}

function omitEmptySecretFields<T extends Record<string, string | boolean>>(value: T, secretFields: string[]) {
  const next = { ...value };
  for (const key of secretFields) {
    if (next[key] === '') delete next[key];
  }
  return next;
}

export async function saveConfig(service: 'unbound' | 'adguard' | 'cloudflare'): Promise<void> {
  const store = useStore.getState();
  if (!store.mutationEnabled) {
    store.setConfigStatus('Config changes are unavailable for this web session.');
    store.setConfigStatusKind('error');
    return;
  }
  store.setConfigStatus(`Saving ${service} config...`);
  store.setConfigStatusKind('info');
  try {
    const payload = buildConfigUpdate(service, store.forms);
    const nextConfig = await api.saveConfig(payload);
    store.setConfig(nextConfig);
    saveCachedConfig(nextConfig);
    store.clearPlan();
    useStore.setState((s) => ({ savedForms: { ...s.savedForms, [service]: s.forms[service] } }));
    store.setConfigStatus(`Saved ${service} config.`);
    store.setConfigStatusKind('ok');
  } catch (err) {
    store.setConfigStatus(err instanceof Error ? err.message : String(err));
    store.setConfigStatusKind('error');
  }
}

export async function saveCaddyEditor(): Promise<void> {
  const store = useStore.getState();
  if (!store.mutationEnabled) {
    store.setConfigStatus('Config changes are unavailable for this web session.');
    store.setConfigStatusKind('error');
    return;
  }
  store.setConfigStatus('Saving Caddy editor config...');
  store.setConfigStatusKind('info');
  try {
    const ce = store.forms.caddyEditor;
    const payload = {
      caddy_editor: {
        enabled: ce.enabled,
        repo_path: ce.repo_path,
        caddyfile: ce.caddyfile,
        deploy_command: ce.deploy_command,
        validate_command: ce.validate_command,
        git_auto_commit: ce.git_auto_commit,
        git_auto_push: ce.git_auto_push,
        git_remote: ce.git_remote,
        git_branch: ce.git_branch,
        entry_template: ce.entry_template,
      },
    };
    const nextConfig = await api.saveConfig(payload);
    store.setConfig(nextConfig);
    saveCachedConfig(nextConfig);
    store.clearPlan();
    useStore.setState((s) => ({ savedForms: { ...s.savedForms, caddyEditor: s.forms.caddyEditor } }));
    store.setConfigStatus('Saved Caddy editor config.');
    store.setConfigStatusKind('ok');
  } catch (err) {
    store.setConfigStatus(err instanceof Error ? err.message : String(err));
    store.setConfigStatusKind('error');
  }
}

export async function testConfig(service: ServiceKey): Promise<void> {
  const store = useStore.getState();
  if (!store.mutationEnabled) {
    store.setConfigStatus('Config tests are unavailable for this web session.');
    store.setConfigStatusKind('error');
    return;
  }
  store.setConfigStatus(`Testing ${service} config...`);
  store.setConfigStatusKind('info');
  store.updateTestResult(service, { text: 'Testing connection...', kind: 'info' });
  try {
    const result = await api.testConfig(service);
    const detail = formatTestDetails(result.details || {});
    const text = `${result.message}${detail ? ` ${detail}` : ''}`;
    const kind = result.success ? 'ok' : 'error';
    store.updateTestResult(service, { text, kind });
    store.setConfigStatus(result.message);
    store.setConfigStatusKind(kind);
  } catch (err) {
    const text = err instanceof Error ? err.message : String(err);
    store.updateTestResult(service, { text, kind: 'error' });
    store.setConfigStatus(text);
    store.setConfigStatusKind('error');
  }
}

// Sync config form state from API config response.
export function syncFormsFromConfig(config: ConfigResponse | null) {
  if (!config) return;
  const ce = config.caddy_editor;
  useStore.setState((state) => {
    const next: ConfigForms = {
      unbound: {
        ...state.forms.unbound,
        base_url: config.summary.unbound?.endpoint || '',
        insecure: Boolean(config.summary.unbound?.insecure),
      },
      adguard: {
        ...state.forms.adguard,
        enabled: Boolean(config.summary.adguard?.enabled),
        base_url: config.summary.adguard?.endpoint || '',
        insecure: Boolean(config.summary.adguard?.insecure),
      },
      cloudflare: {
        ...state.forms.cloudflare,
        enabled: Boolean(config.summary.cloudflare?.enabled),
        caddy_service_url: config.summary.cloudflare?.details?.caddy_service_url || '',
        insecure: Boolean(config.summary.cloudflare?.insecure),
      },
      caddyEditor: ce
        ? {
            enabled: ce.enabled,
            repo_path: ce.repo_path || '',
            caddyfile: ce.caddyfile || 'caddy/Caddyfile',
            deploy_command: ce.deploy_command || '',
            validate_command: ce.validate_command || '',
            git_auto_commit: ce.git_auto_commit,
            git_auto_push: ce.git_auto_push,
            git_remote: ce.git_remote || 'origin',
            git_branch: ce.git_branch || '',
            entry_template: ce.entry_template || 'default',
          }
        : state.forms.caddyEditor,
    };
    return { forms: next, savedForms: next };
  });
}
