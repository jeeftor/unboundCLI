// Caddy DNS Sync — Vanilla JS dashboard
// Replaces the React bundle. No build step required.
'use strict';

const wc = () => window.UNBOUNDCLI_WEB_CONFIG || { applyToken: '', mutationEnabled: false };

// ── State ──────────────────────────────────────────────────────────────────
const S = {
  config: null, entries: [], report: {},
  loading: true, message: '', msgKind: 'info',
  search: '', statusFilter: 'all', statusFilterInverse: false, serviceFilter: 'all',
  selectedHostname: '',
  syncService: 'all', syncLoading: false, syncLog: '',
  syncProgress: { title: '', detail: '' },
  plannedActions: [], planId: '', actionIds: [], canSyncNow: false, disabledPlanIndices: new Set(),
  e2eDone: false,
  cfWizard: { open: false, hostname: '', loading: false, actions: [], planId: '', actionIds: [], log: '', applied: false,
              originMode: 'via-caddy', noTLSVerify: false, disableChunkedEncoding: false, selectedTunnelId: '',
              availableTunnels: [], tunnelsLoading: false },
  configOpen: false, configTab: 'caddy', configStatus: '', configStatusKind: '',
  cfDiscover: { loading: false, verifyOk: false, verifyMsg: '', accounts: [], tunnels: [], zones: [] },
  testResults: {},
  probe: { loading: false, result: null, hostname: '' },
  inspectorPlan: { loading: false, actions: [], planId: '', actionIds: [], log: '', applying: false, applyLog: '', disabledIndices: new Set() },
  rowModal: { open: false, hostname: '', loading: false, actions: [], planId: '', actionIds: [], applying: false, applyLog: '', disabledIndices: new Set(), autoApply: false, unsyncLoading: '', svcOps: {} },
  forms: {
    unbound:    { base_url: '', api_key: '', api_secret: '', insecure: false },
    adguard:    { enabled: false, base_url: '', username: '', password: '', insecure: false },
    cloudflare: { enabled: false, api_token: '', account_id: '', zone_id: '', tunnel_id: '', caddy_service_url: '', insecure: false },
  },
};

// ── Derived ────────────────────────────────────────────────────────────────
function filteredEntries() {
  const q = S.search.toLowerCase();
  return S.entries.filter(e => {
    if (q && !e.hostname.toLowerCase().includes(q)) return false;
    const st = S.statusFilter;
    const inv = S.statusFilterInverse;
    function matchStatus() {
      if (st === 'all')        return true;
      if (st === 'synced')     return e.overall_status === 0;
      if (st === 'out_of_sync') return e.overall_status > 1;
      if (st === 'caddy_only') return e.overall_status === 3;
      if (st === 'stale')      return e.overall_status === 4;
      if (st === 'cloudflare') return !!e.cloudflare_status?.configured;
      return true;
    }
    if (st !== 'all' && matchStatus() === inv) return false;
    const sf = S.serviceFilter;
    if (sf === 'caddy'      && !e.caddy_upstream)                return false;
    if (sf === 'unbound'    && !e.unbound_status?.configured)    return false;
    if (sf === 'adguard'    && !e.adguard_status?.configured)    return false;
    if (sf === 'dhcp'       && !e.dhcp_status?.configured)       return false;
    if (sf === 'cloudflare' && !e.cloudflare_status?.configured) return false;
    return true;
  });
}

function summary() {
  const e = S.entries;
  return {
    total:      e.length,
    inSync:     e.filter(x => x.overall_status <= 1).length,
    caddyOnly:  e.filter(x => x.overall_status === 3).length,
    stale:      e.filter(x => x.overall_status === 4).length,
    cloudflare: e.filter(x => x.cloudflare_status?.configured).length,
  };
}

function selectedEntry() { return S.entries.find(e => e.hostname === S.selectedHostname); }

// ── API ────────────────────────────────────────────────────────────────────
async function api(path, opts = {}) {
  const headers = { 'Content-Type': 'application/json' };
  if (wc().mutationEnabled && wc().applyToken) headers['X-UnboundCLI-Token'] = wc().applyToken;
  const res  = await fetch(path, { ...opts, headers: { ...headers, ...(opts.headers || {}) } });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

async function refresh() {
  S.loading = true; S.message = ''; render();
  try {
    const [cfg, ents] = await Promise.all([api('/api/config'), api('/api/entries')]);
    S.config  = cfg;
    S.entries = ents.entries || [];
    S.report  = ents.report  || {};
    if (!S.selectedHostname && S.entries.length) S.selectedHostname = S.entries[0].hostname;
    if (!syncTargetOptions().some(([value]) => value === S.syncService)) S.syncService = 'all';
    // Pre-populate form toggles from saved config so Save never accidentally flips them
    const ag = cfg.summary?.adguard;
    if (ag) {
      S.forms.adguard.enabled  = !!ag.enabled;
      S.forms.adguard.insecure = !!ag.insecure;
    }
    const cf = cfg.summary?.cloudflare;
    if (cf) {
      S.forms.cloudflare.enabled  = !!cf.enabled;
      S.forms.cloudflare.insecure = !!cf.insecure;
    }
  } catch (err) { S.message = `Load error: ${err.message}`; S.msgKind = 'error'; }
  S.loading = false; render();
}

// fetchInspectorPlan: scoped to one hostname, results go to the inspector panel.
// opts: { service: 'all'|'unbound'|'adguard', unsync: bool }
async function fetchInspectorPlan(hostname, opts = {}) {
  S.inspectorPlan = { loading: true, actions: [], planId: '', actionIds: [], log: '', applying: false, applyLog: '', disabledIndices: new Set() };
  render();
  try {
    const params = new URLSearchParams({ service: opts.service || 'all', hostname });
    if (opts.unsync) params.set('unsync', 'true');
    const url = `/api/sync/plan?${params}`;
    const data = await api(url);
    const actions = data.actions || [];
    S.inspectorPlan = {
      loading: false, applying: false, applyLog: '',
      actions,
      planId:    data.plan_id    || '',
      actionIds: data.action_ids || [],
      log: fmtPlan(actions),
      disabledIndices: new Set(),
    };
  } catch (err) {
    S.inspectorPlan = { loading: false, actions: [], planId: '', actionIds: [], log: `Error: ${err.message}`, applying: false, applyLog: '', disabledIndices: new Set() };
  }
  render();
}

async function applyInspectorPlan() {
  const p = S.inspectorPlan;
  if (!p.actions.length) return;
  // Filter out disabled (unchecked) actions
  const enabledIds = (p.actionIds || []).filter((_, i) => !p.disabledIndices.has(i));
  if (!enabledIds.length && p.planId) { S.inspectorPlan = { ...p, applyLog: 'No actions selected.' }; render(); return; }
  S.inspectorPlan = { ...p, applying: true, applyLog: '' };
  render();
  try {
    const body = p.planId
      ? { dry_run: false, plan_id: p.planId, action_ids: enabledIds }
      : { dry_run: false, actions: p.actions.filter((_, i) => !p.disabledIndices.has(i)) };
    const { result: r } = await api('/api/sync/apply', { method: 'POST', body: JSON.stringify(body) });
    const applyLog = fmtApplyResult(r, false);
    if (!r?.errors?.length) {
      S.inspectorPlan = { loading: false, actions: [], planId: '', actionIds: [], log: '', applying: false, applyLog };
      await refresh();
      return;
    }
    S.inspectorPlan = { ...S.inspectorPlan, applying: false, applyLog };
  } catch (err) {
    S.inspectorPlan = { ...S.inspectorPlan, applying: false, applyLog: `Apply error: ${err.message}` };
  }
  render();
}

async function fetchPlan(service, hostname) {
  S.syncLoading = true; S.syncProgress = { title: 'Building plan…', detail: `Target: ${service}` }; render();
  try {
    let url = `/api/sync/plan?service=${encodeURIComponent(service || 'all')}`;
    if (hostname) url += `&hostname=${encodeURIComponent(hostname)}`;
    const data       = await api(url);
    S.plannedActions = data.actions    || [];
    S.planId         = data.plan_id    || '';
    S.actionIds      = data.action_ids || [];
    S.canSyncNow     = wc().mutationEnabled && S.plannedActions.length > 0;
    S.syncLog        = fmtPlan(S.plannedActions);
  } catch (err) { S.syncLog += `\nError: ${err.message}`; }
  S.syncLoading = false; S.syncProgress = { title: '', detail: '' }; render();
  return S.plannedActions.length > 0;
}

async function applySync(dryRun) {
  if (!S.plannedActions.length) return;
  S.syncLoading = true; S.syncProgress = { title: dryRun ? 'Dry-running…' : 'Applying…', detail: '' }; render();
  try {
    const body = dryRun
      ? { dry_run: true, actions: S.plannedActions }
      : { dry_run: false, plan_id: S.planId, action_ids: S.actionIds };
    const { result: r } = await api('/api/sync/apply', { method: 'POST', body: JSON.stringify(body) });
    S.syncLog += '\n' + fmtApplyResult(r, dryRun);
    if (!dryRun && !r?.errors?.length) {
      S.message = 'Sync applied.'; S.msgKind = 'ok';
      S.syncLoading = false; S.syncProgress = { title: '', detail: '' };
      await refresh();
      return;
    }
  } catch (err) { S.syncLog += `\nApply error: ${err.message}`; }
  S.syncLoading = false; S.syncProgress = { title: '', detail: '' }; render();
}

async function testConfig(service) {
  try {
    const data = await api('/api/config/test', { method: 'POST', body: JSON.stringify({ service }) });
    S.testResults[service] = { text: data.message, kind: data.success ? 'ok' : 'error' };
  } catch (err) { S.testResults[service] = { text: `Failed: ${err.message}`, kind: 'error' }; }
  render();
}

async function doSave(service) {
  const update = {};
  const uf = S.forms.unbound, af = S.forms.adguard, cf = S.forms.cloudflare;
  if (service === 'unbound') {
    update.unbound = { insecure: uf.insecure };
    if (uf.base_url)   update.unbound.base_url   = uf.base_url;
    if (uf.api_key)    update.unbound.api_key     = uf.api_key;
    if (uf.api_secret) update.unbound.api_secret  = uf.api_secret;
  } else if (service === 'adguard') {
    update.adguard = { enabled: af.enabled, insecure: af.insecure };
    if (af.base_url)  update.adguard.base_url  = af.base_url;
    if (af.username)  update.adguard.username  = af.username;
    if (af.password)  update.adguard.password  = af.password;
  } else if (service === 'cloudflare') {
    update.cloudflare = { enabled: cf.enabled, insecure: cf.insecure };
    if (cf.api_token)        update.cloudflare.api_token        = cf.api_token;
    if (cf.account_id)       update.cloudflare.account_id       = cf.account_id;
    if (cf.zone_id)          update.cloudflare.zone_id          = cf.zone_id;
    if (cf.tunnel_id)        update.cloudflare.tunnel_id        = cf.tunnel_id;
    if (cf.caddy_service_url) update.cloudflare.caddy_service_url = cf.caddy_service_url;
  }
  try {
    S.config = await api('/api/config', { method: 'POST', body: JSON.stringify(update) });
    if (!syncTargetOptions().some(([value]) => value === S.syncService)) S.syncService = 'all';
    S.configStatus = `Saved ${service} config.`; S.configStatusKind = 'ok';
    S.message = `Saved ${service} config.`; S.msgKind = 'ok';
  } catch (err) { S.configStatus = `Save error: ${err.message}`; S.configStatusKind = 'error'; }
  render();
}

// ── Helpers ────────────────────────────────────────────────────────────────
const esc = s => String(s ?? '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');

function fmtPlan(actions) {
  if (!actions.length) return '✓ No changes needed.';
  const verbs = { add:'ADD', update:'UPDATE', delete:'DELETE', remove:'REMOVE', noop:'NOOP' };
  return actions.map(a => {
    let detail = '';
    if (a.service === 'cloudflare') {
      const svc = a.new_service || a.old_service || '';
      const hh = a.new_http_host_header || a.old_http_host_header || '';
      detail = `${svc ? ` → ${svc}` : ''}${hh ? ` host=${hh}` : ''}`;
    } else if (a.new_ip) {
      detail = ` → ${a.new_ip}`;
    } else if (a.old_ip && a.type === 'delete') {
      detail = ` remove ${a.old_ip}`;
    }
    const verb = a.service === 'cloudflare' && a.type === 'delete' ? 'UNSYNC' : (verbs[a.type] || String(a.type || 'ACTION').toUpperCase());
    return `${verb} ${a.service} ${a.hostname}${detail}${a.details ? ` (${a.details})` : ''}`;
  }).join('\n');
}

function fmtActionTarget(a) {
  if (!a) return '';
  if (a.service === 'cloudflare') {
    const svc = a.new_service || a.old_service || '';
    const host = a.new_http_host_header || a.old_http_host_header || '';
    return `${a.hostname}${svc ? ` → ${svc}` : ''}${host ? ` host=${host}` : ''}`;
  }
  if (a.type === 'delete' && a.old_ip) return `${a.hostname} remove ${a.old_ip}`;
  if (a.new_ip) return `${a.hostname} → ${a.new_ip}`;
  return a.hostname || '';
}

function fmtApplyResult(r, dryRun) {
  if (!r) return 'Apply returned no result.';
  const lines = [];
  const prefix = dryRun ? 'Dry-run' : 'Applied';
  if (r.message) lines.push(r.message);
  const added = Number(r.items_added || 0);
  const updated = Number(r.items_updated || 0);
  const deleted = Number(r.items_deleted || 0);
  if (added || updated || deleted) lines.push(`${prefix}: +${added} ~${updated} -${deleted}`);
  lines.push(`added=${added} updated=${updated} deleted=${deleted}`);

  for (const ar of r.action_results || []) {
    const a = ar.action || {};
    const icon = ar.skipped ? '·' : ar.success ? '✓' : '!';
    const verb = a.service === 'cloudflare' && a.type === 'delete' ? 'unsync' : (a.type || 'action');
    const suffix = ar.error ? ` — ${ar.error}` : ar.skipped ? ' — skipped' : '';
    lines.push(`${icon} [${a.service || '?'}] ${verb} ${fmtActionTarget(a)}${suffix}`);
  }

  if (r.errors?.length) lines.push(`! Errors: ${r.errors.map(e => e.message || e).join(', ')}`);
  return lines.join('\n') || '✓ Done — no changes.';
}

function syncTargetOptions() {
  const enabled = S.config?.enabled || {};
  const opts = [['all', 'All enabled targets']];
  if (enabled.unbound !== false) opts.push(['unbound', 'Unbound']);
  if (enabled.adguard !== false) opts.push(['adguard', 'AdGuard']);
  if (enabled.cloudflare) opts.push(['cloudflare', 'Cloudflare']);
  return opts;
}

const statusCls = code => code <= 1 ? 'ok' : (code === 2 || code >= 4) ? 'bad' : 'warn';
const dnsCls    = val  => String(val||'').toLowerCase() && String(val||'').toLowerCase() !== 'fail' ? 'ok' : 'bad';

function svcText(s) {
  if (!s?.configured) return '—';
  if (s.in_sync) return s.ip || 'In sync';
  return (s.ip || 'Mismatch') + ' ✗';
}
const svcTone = s => !s?.configured ? 'missing' : s.in_sync ? 'ok' : 'bad';
function cfBadge(cf, hostname) {
  if (!cf?.configured) {
    const cfReady = S.config?.summary?.cloudflare?.client_ready;
    if (!cfReady) return '';  // CF not configured at all — don't show badge
    return `<button class="service-badge missing cf-unrouted" data-action="cf-wizard" data-hostname="${esc(hostname)}" title="Not in Cloudflare tunnel — click to add"><strong>CF</strong><span>+ Route</span></button>`;
  }
  const label = cf.tunnel_name || 'CF';
  const details = [cf.service ? `Service: ${cf.service}` : '', cf.http_host_header ? `Host header: ${cf.http_host_header}` : ''].filter(Boolean).join(' · ');
  return `<span class="service-badge cf" data-label="Cloudflare route" title="${esc(details)}"><strong>CF</strong><span>${esc(label)}</span></span>`;
}
// Compact pill variant for table rows
function cfPill(cf, hostname) {
  if (!cf?.configured) {
    const cfReady = S.config?.summary?.cloudflare?.client_ready;
    if (!cfReady) return '';
    return `<button class="svc-pill missing cf-unrouted" data-action="cf-wizard" data-hostname="${esc(hostname)}" title="Not in Cloudflare — click to add">CF+</button>`;
  }
  const tip = cf.service ? `CF: ${cf.service}` : `CF: ${cf.tunnel_name||'Routed'}`;
  return `<span class="svc-pill cf" title="${esc(tip)}">CF</span>`;
}

// ── SVG Icons (inline) ─────────────────────────────────────────────────────
const ICON = {
  search:   `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>`,
  chevron:  `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="6 9 12 15 18 9"/></svg>`,
  play:     `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>`,
  shield:   `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>`,
  zap:      `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>`,
  wifi_off: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="1" y1="1" x2="23" y2="23"/><path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55"/><path d="M5 12.55a10.94 10.94 0 0 1 5.17-2.39"/><path d="M10.71 5.05A16 16 0 0 1 22.56 9"/><path d="M1.42 9a15.91 15.91 0 0 1 4.7-2.88"/><path d="M8.53 16.11a6 6 0 0 1 6.95 0"/><line x1="12" y1="20" x2="12.01" y2="20"/></svg>`,
  gear:     `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`,
};

// ── Templates ──────────────────────────────────────────────────────────────
function tTopbar() {
  const c = S.config, running = c?.enabled?.caddy !== false;
  const addr = c ? `${c.caddy.server_ip}:${c.caddy.server_port}` : '…';
  const pills = [
    { key: 'caddy',      label: 'Caddy',   on: c?.enabled?.caddy      !== false },
    { key: 'unbound',    label: 'Unbound', on: c?.enabled?.unbound    !== false },
    { key: 'adguard',    label: 'AdGuard', on: c?.enabled?.adguard    !== false },
    { key: 'dhcp',       label: 'DHCP',    on: c?.enabled?.dhcp       !== false },
    { key: 'cloudflare', label: 'Cloudflare', on: c?.enabled?.cloudflare !== false },
  ];
  return `<header class="topbar">
    <div class="brand-inline">
      <div class="brand-mark">⚡</div>
      <span>Caddy DNS Sync</span>
    </div>
    <nav class="svc-pills" aria-label="Filter by service">
      ${pills.map(p => `<button class="svc-pill ${p.on?'on':'off'}${S.serviceFilter===p.key?' pill-active':''}"
        data-action="filter-svc" data-svc="${p.key}" title="${p.label}: ${p.on?'connected':'offline'}">
        <i class="svc-dot"></i><span>${p.label}</span>
      </button>`).join('')}
      ${S.serviceFilter !== 'all' ? `<button class="svc-pill pill-clear" data-action="filter-svc" data-svc="all">✕ All</button>` : ''}
    </nav>
    <div class="runtime-card">
      <span>Caddy</span>
      <strong>${esc(addr)}</strong>
      <em class="${running?'':'down'}">${running?'Running':'Offline'}</em>
    </div>
    <div class="top-actions">
      <button data-action="refresh" ${S.loading?'disabled':''}>↺ Refresh</button>
      <button data-action="open-config">${ICON.gear} Settings</button>
    </div>
  </header>`;
}

function tMetrics() {
  const s = summary();
  const sf = S.statusFilter, inv = S.statusFilterInverse;
  function mcard(tone, status, label, val, sub) {
    const isActive = sf === status && status !== 'all';
    const cls = isActive ? (inv ? ' metric-active metric-inverse' : ' metric-active') : '';
    const titleHint = isActive ? (inv ? `Showing: NOT ${label} — click to clear` : `Showing: ${label} — click to invert`) : `Click to filter: ${label}`;
    return `<article class="metric-card ${tone}${cls}" data-action="filter-status" data-status="${status}" role="button" tabindex="0" title="${titleHint}">
      <div><span>${label}${isActive ? `<em class="filter-mode-tag">${inv ? '≠' : '='}</em>` : ''}</span><strong>${val}</strong><small>${sub}</small></div>
    </article>`;
  }
  return `<section class="metric-grid">
    ${mcard('neutral','all',    'Total',      s.total,     'hostnames')}
    ${mcard('ok',     'synced', 'In sync',    s.inSync,    'healthy')}
    ${mcard('warn',   'caddy_only','Caddy only', s.caddyOnly,'not in DNS')}
    ${mcard('bad',    'stale',  'Stale DNS',  s.stale,     'needs cleanup')}
    ${mcard('violet', 'cloudflare','Cloudflare',s.cloudflare,'via tunnel')}
  </section>`;
}

function tToolbar(entries) {
  const statusOpts = [
    ['all','All status'],['out_of_sync','Out of sync'],
    ['caddy_only','Caddy only'],['stale','Stale DNS'],['cloudflare','Cloudflare'],
  ];
  return `<section class="entries-toolbar panel">
    <div class="search-box">
      ${ICON.search}
      <input id="search" type="search" placeholder="Search hostnames…"
        value="${esc(S.search)}" autocomplete="off" spellcheck="false"/>
    </div>
    <span class="select-wrap">
      <select id="status-filter" aria-label="Status filter">
        ${statusOpts.map(([v,l]) => `<option value="${v}"${S.statusFilter===v?' selected':''}>${l}</option>`).join('')}
      </select>${ICON.chevron}
    </span>
    <span class="entry-count">${entries.length} entries</span>
  </section>`;
}

function tTable(entries) {
  if (!entries.length) return `<section id="entries-panel" class="panel entries-panel" data-table-scrolls="false">
    <div style="padding:28px 16px;color:var(--text-muted);font-size:13px">No entries match your filters.</div>
  </section>`;

  const rows = entries.map((e, idx) => {
    const sel = e.hostname === S.selectedHostname;
    const ub  = e.unbound_status, ag = e.adguard_status, cf = e.cloudflare_status;
    return `<tr class="${sel?'selected-row':''}" data-action="select-row" data-hostname="${esc(e.hostname)}" data-idx="${idx}" tabindex="0" aria-selected="${sel ? 'true' : 'false'}">
      <td data-label="Hostname"><strong>${esc(e.hostname)}</strong><br><span class="subtle">${esc(e.data_source||'Caddy')}</span></td>
      <td data-label="Status"><span class="status-chip ${statusCls(e.overall_status)}">${esc(e.status_label||'Unknown')}</span></td>
      <td data-label="Services">
        <div class="service-stack">
          <span class="service-badge ${svcTone(ub)}"><strong>UB</strong> ${esc(svcText(ub))}</span>
          <span class="service-badge ${svcTone(ag)}"><strong>AG</strong> ${esc(svcText(ag))}</span>
          ${cfBadge(cf, e.hostname)}
        </div>
      </td>
      <td data-label="Upstream"><span>${esc(e.caddy_upstream||'—')}</span><br><span class="subtle">${esc(e.caddy_ip||'')}</span></td>
      <td data-label="DNS"><span class="dns-result ${dnsCls(e.dns_resolved)}">${esc(e.dns_resolved||'FAIL')}</span></td>
      <td data-label="Actions">
        <div class="row-actions">
          <button class="row-modify" data-action="row-modify" data-hostname="${esc(e.hostname)}">Modify</button>
          <button class="row-sync-direct${e.overall_status===4?' row-cleanup':''}" data-action="row-sync-direct" data-hostname="${esc(e.hostname)}"${wc().mutationEnabled?'':' disabled'}>${e.overall_status===4?'Clean up':'Sync'}</button>
        </div>
      </td>
    </tr>`;
  }).join('');

  return `<section id="entries-panel" class="panel entries-panel" data-table-scrolls="false">
    <table>
      <thead><tr>
        <th>Hostname</th><th>Status</th><th>Services</th><th>Upstream</th><th>DNS</th><th>Actions</th>
      </tr></thead>
      <tbody id="entries">${rows}</tbody>
    </table>
  </section>`;
}

function tSyncPanel() {
  const opts = syncTargetOptions();
  const progressTitle = S.syncProgress.title || 'Sync idle';
  const progressDetail = S.syncProgress.detail || 'No active operation';
  return `<section id="sync-panel" class="panel sync-panel">
    <div class="panel-title">
      <div><strong>Sync Plan</strong><span>Caddy is the source of truth.</span></div>
      <span class="plan-count">${S.plannedActions.length} changes</span>
    </div>
    <label style="font-size:10px;color:var(--text-muted);font-weight:800;letter-spacing:.08em;text-transform:uppercase">Target</label>
    <span class="select-wrap">
      <select id="sync-service" aria-label="Sync target">
        ${opts.map(([v,l]) => `<option value="${v}"${S.syncService===v?' selected':''}>${l}</option>`).join('')}
      </select>${ICON.chevron}
    </span>
    <div class="sync-pipeline">
      <button id="preview-sync" data-action="preview-sync"${S.syncLoading?' disabled':''}>
        ${ICON.play}<span><strong>Preview sync</strong><small>Fetch plan from server</small></span>
      </button>
      <button id="dry-run-sync" data-action="dry-run" data-dry-run-enabled="${(!S.syncLoading && S.plannedActions.length) ? 'true' : 'false'}"${S.syncLoading||!S.plannedActions.length?' disabled':''}>
        ${ICON.shield}<span><strong>Dry-run</strong><small>Simulate, don't apply</small></span>
      </button>
      <button id="sync-now" data-action="sync-now" data-sync-enabled="${(!S.syncLoading && S.canSyncNow) ? 'true' : 'false'}"${S.syncLoading||!S.canSyncNow?' disabled':''}
        title="${wc().mutationEnabled?'Apply server-issued plan':'Sync unavailable in this session'}">
        ${ICON.zap}<span><strong>Sync now</strong><small>Apply the plan</small></span>
      </button>
    </div>
    <div id="sync-progress" class="inline-progress" role="status" aria-live="polite" aria-hidden="${S.syncLoading ? 'false' : 'true'}">
      <div class="loading-copy compact"><span id="sync-progress-title">${esc(progressTitle)}</span><strong id="sync-progress-detail">${S.syncLoading ? 'Working' : esc(progressDetail)}</strong></div>
      <div class="progress-track"><span></span></div>
    </div>
    <div class="log-header"><strong>Plan log</strong></div>
    <div id="sync-log" class="log" role="status" aria-live="polite">${esc(S.syncLog)||'Run a preview to see the sync plan.'}</div>
  </section>`;
}

function tInspectorPlan(e) {
  const p = S.inspectorPlan;
  const isStale = e.overall_status === 4;
  const mut = wc().mutationEnabled;

  // Plan not yet fetched — show a single "Check changes" button
  if (!p.loading && !p.actions.length && !p.log && !p.applyLog) {
    return `<div class="insp-plan-idle">
      <button class="insp-plan-check" data-action="inspector-preview">Check what needs to change</button>
    </div>`;
  }

  // Loading
  if (p.loading) {
    return `<div class="insp-plan-wrap">
      <div class="insp-plan-header"><strong>Changes needed</strong><span class="probe-pending">Checking…</span></div>
    </div>`;
  }

  // Applied result
  if (p.applyLog) {
    return `<div class="insp-plan-wrap">
      <div class="insp-plan-header"><strong>Result</strong></div>
      <pre class="insp-plan-log ok">${esc(p.applyLog)}</pre>
      <button class="insp-plan-check" data-action="inspector-preview">Check again</button>
    </div>`;
  }

  // No changes needed
  if (!p.actions.length) {
    return `<div class="insp-plan-wrap">
      <div class="insp-plan-header"><strong>Changes needed</strong><span class="probe-up">✓ Already in sync</span></div>
      <button class="insp-plan-recheck" data-action="inspector-preview">Re-check</button>
    </div>`;
  }

  // Has actions — show them with checkboxes + apply button
  const enabledCount = p.actions.filter((_, i) => !p.disabledIndices.has(i)).length;
  const rows = p.actions.map((a, i) => {
    const enabled = !p.disabledIndices.has(i);
    const typeClass = a.type === 'delete' ? 'plan-del' : a.type === 'add' ? 'plan-add' : 'plan-upd';
    const verb = a.type === 'delete' ? '−' : a.type === 'add' ? '+' : '~';
    const svc  = (a.service || '').toUpperCase().slice(0, 2);
    const detail = a.new_ip || a.new_service || a.old_ip || '';
    return `<label class="plan-row ${typeClass}${enabled?'':' plan-row-disabled'}" title="${enabled?'Click to skip':'Click to include'}">
      <input type="checkbox" class="plan-row-check" data-action="toggle-insp-action" data-idx="${i}"${enabled?' checked':''}>
      <span class="plan-verb">${verb}</span>
      <span class="plan-svc">${svc}</span>
      <span class="plan-detail">${esc(detail || a.details || '')}</span>
    </label>`;
  }).join('');

  const applyLabel = p.applying ? 'Applying…' : isStale ? 'Clean up' : `Apply${enabledCount < p.actions.length ? ` (${enabledCount}/${p.actions.length})` : ''}`;
  const applyBtn = mut
    ? `<button class="insp-plan-apply" data-action="inspector-apply"${(p.applying || enabledCount === 0)?' disabled':''}>
         ${applyLabel}
       </button>`
    : `<button class="insp-plan-apply" disabled title="Read-only session">Apply (disabled)</button>`;

  return `<div class="insp-plan-wrap">
    <div class="insp-plan-header">
      <strong>Changes needed</strong>
      <button class="insp-plan-recheck" data-action="inspector-preview">Re-check</button>
    </div>
    <div class="insp-plan-rows">${rows}</div>
    ${applyBtn}
  </div>`;
}

function tInspector() {
  const e = selectedEntry();
  if (!e) return `<section id="host-inspector" class="panel inspector" aria-live="polite">
    <div class="panel-title"><strong>Selected host</strong><span>Click a row to inspect.</span></div>
    <div class="empty-state">${ICON.wifi_off} No hostname selected.</div>
  </section>`;

  const ub = e.unbound_status, ag = e.adguard_status, cf = e.cloudflare_status;
  const isStale = e.overall_status === 4;
  // Probe target: prefer caddy_upstream (has real port), fall back to DNS IP from Unbound/AdGuard.
  const probeUpstream = e.caddy_upstream || ub?.ip || ag?.ip || '';

  // Build probe result display
  let probeHtml = '';
  if (probeUpstream) {
    const pr = (S.probe.hostname === e.hostname) ? S.probe.result : null;
    const probeLoading = S.probe.loading && S.probe.hostname === e.hostname;
    let probeResult = '';
    if (probeLoading) {
      probeResult = `<span class="probe-pending">Probing…</span>`;
    } else if (pr) {
      if (pr.reachable) {
        probeResult = `<span class="probe-up">✓ Responding (${pr.status_code||'—'}, ${pr.latency_ms}ms)</span>`;
      } else {
        const msg = pr.error ? pr.error.replace(/.*:.*:\s*/, '') : 'unreachable';
        probeResult = `<span class="probe-down">✗ ${esc(msg)}</span>`;
      }
    }
    probeHtml = `<div class="probe-row">
      <button class="probe-btn" data-action="probe" data-hostname="${esc(e.hostname)}" data-upstream="${esc(probeUpstream)}"${probeLoading?' disabled':''}>Probe</button>
      <span class="probe-target">${esc(probeUpstream)}</span>
      ${probeResult}
    </div>`;
  }

  // Stale banner
  const staleBanner = isStale ? `<div class="stale-banner">
    <strong>Stale DNS entry</strong> — This hostname is no longer in Caddy's config but still has DNS records in ${[ub?.configured&&'Unbound', ag?.configured&&'AdGuard'].filter(Boolean).join(' + ')}. Use <em>Clean up</em> to delete them.
  </div>` : '';

  // Build unsync buttons for services that are currently configured for this hostname
  const mut = wc().mutationEnabled;
  const unsyncBtns = [];
  if (ub?.configured && !isStale) {
    unsyncBtns.push(`<button class="unsync-btn" data-action="unsync-svc" data-hostname="${esc(e.hostname)}" data-svc="unbound"${mut?'':' disabled'} title="Remove this entry from OPNSense / Unbound DNS">Remove from Unbound</button>`);
  }
  if (ag?.configured && !isStale) {
    unsyncBtns.push(`<button class="unsync-btn" data-action="unsync-svc" data-hostname="${esc(e.hostname)}" data-svc="adguard"${mut?'':' disabled'} title="Remove this entry from AdGuard DNS">Remove from AdGuard</button>`);
  }
  const unsyncSection = unsyncBtns.length
    ? `<div class="unsync-section"><span class="unsync-label">Manual removal</span>${unsyncBtns.join('')}</div>`
    : '';

  return `<section id="host-inspector" class="panel inspector" aria-live="polite">
    <div class="host-title">
      <strong>${esc(e.hostname)}</strong>
      <div>
        <span class="status-chip ${statusCls(e.overall_status)}">${esc(e.status_label)}</span>
        <span class="dns-result ${dnsCls(e.dns_resolved)}">${esc(e.dns_resolved||'FAIL')}</span>
      </div>
    </div>
    ${staleBanner}
    <div class="inspector-grid">
      <div class="inspector-line"><span>Upstream</span><strong>${esc(e.caddy_upstream||'—')}</strong></div>
      <div class="inspector-line"><span>Source</span><strong>${esc(e.data_source||'—')}</strong></div>
      <div class="inspector-line ${svcTone(ub)}"><span>Unbound</span><strong>${esc(svcText(ub))}</strong></div>
      <div class="inspector-line ${svcTone(ag)}"><span>AdGuard</span><strong>${esc(svcText(ag))}</strong></div>
      <div class="inspector-line ${cf?.configured?'violet':''}"><span>Cloudflare</span><strong>${cf?.configured?esc(cf.service||'Routed'):'Not routed'}</strong></div>
    </div>
    ${unsyncSection}
    ${probeHtml}
    ${tInspectorPlan(e)}
  </section>`;
}

const SVC_ICONS = {
  caddy:      '⬡',
  unbound:    '◈',
  adguard:    '⬡',
  dhcp:       '⊞',
  cloudflare: '☁',
};

function field(label, inputHtml) {
  return `<div class="cfg-field-row"><label>${label}</label>${inputHtml}</div>`;
}
function inp(type, placeholder, val, form, fld) {
  return `<input type="${type}" placeholder="${esc(placeholder)}"${val?` value="${esc(val)}"`:''} data-form="${form}" data-field="${fld}"/>`;
}
function chk(label, checked, form, fld) {
  return `<label class="cfg-checkbox"><input type="checkbox"${checked?' checked':''} data-form="${form}" data-field="${fld}" data-type="checkbox"/><span>${label}</span></label>`;
}

function tCfgCard(svc, c, tone) {
  const s = c.summary[svc]; if (!s) return '';
  const tr  = S.testResults[svc];
  const mut = wc().mutationEnabled;

  const statusBadge = s.client_ready
    ? `<span class="cfg-badge connected">Connected</span>`
    : s.enabled
      ? `<span class="cfg-badge warn">Not ready</span>`
      : `<span class="cfg-badge off">Disabled</span>`;
  const srcBadge = `<span class="cfg-badge src">${esc(s.source?.kind || 'default')}</span>`;

  const missingHtml = s.missing?.length
    ? s.missing.map(m => `<span class="cfg-missing-tag bad">${esc(m)}</span>`).join('')
    : `<span class="cfg-missing-tag ok">✓ All fields set</span>`;

  const trHtml = tr ? `<div id="config-test-${esc(svc)}" class="cfg-test-result ${tr.kind}">${esc(tr.text)}</div>` : '';

  let fields = '';
  if (svc === 'caddy') {
    fields = `<div class="cfg-fields">
      ${trHtml}
      <div class="cfg-actions">
        <button class="cfg-btn" data-action="test-cfg" data-svc="caddy"${mut?'':' disabled'}>Test Caddy</button>
      </div>
    </div>`;
  } else if (svc === 'unbound') {
    const f = S.forms.unbound;
    fields = `<div class="cfg-fields">
      ${field('Base URL', inp('url','https://opnsense.local', f.base_url, 'unbound', 'base_url'))}
      ${field('API Key',  inp('password','leave unchanged', '', 'unbound', 'api_key'))}
      ${field('API Secret', inp('password','leave unchanged', '', 'unbound', 'api_secret'))}
      ${chk('Skip TLS verification', f.insecure, 'unbound', 'insecure')}
      ${trHtml}
      <div class="cfg-actions">
        <button class="cfg-btn" data-action="test-cfg" data-svc="unbound"${mut?'':' disabled'}>Test OPNSense</button>
        <button class="cfg-btn save" data-action="save-cfg" data-svc="unbound"${mut?'':' disabled'}>Set OPNSense</button>
      </div>
    </div>`;
  } else if (svc === 'adguard') {
    const f = S.forms.adguard;
    fields = `<div class="cfg-fields">
      ${chk('Enable AdGuard sync', f.enabled, 'adguard', 'enabled')}
      ${field('Base URL',  inp('url','https://adguard.local', f.base_url, 'adguard', 'base_url'))}
      ${field('Username',  inp('password','leave unchanged', '', 'adguard', 'username'))}
      ${field('Password',  inp('password','leave unchanged', '', 'adguard', 'password'))}
      ${chk('Skip TLS verification', f.insecure, 'adguard', 'insecure')}
      ${trHtml}
      <div class="cfg-actions">
        <button class="cfg-btn" data-action="test-cfg" data-svc="adguard"${mut?'':' disabled'}>Test</button>
        <button class="cfg-btn save" data-action="save-cfg" data-svc="adguard"${mut?'':' disabled'}>Save</button>
      </div>
    </div>`;
  } else if (svc === 'dhcp') {
    fields = `<div class="cfg-fields" style="color:var(--text-muted);font-size:12px">DHCP / DNSMasq is read-only — no config required.</div>`;
  } else if (svc === 'cloudflare') {
    const f = S.forms.cloudflare;
    const d = S.cfDiscover;
    const tokenSaved = s?.fields?.api_token_set;
    const acctSaved  = s?.fields?.account_id_set;

    // Zone/tunnel selects or text inputs depending on discovery state
    const zoneInput = d.zones.length
      ? `<select class="cf-select" data-form="cloudflare" data-field="zone_id">
           <option value="">Select zone…</option>
           ${d.zones.map(z => `<option value="${esc(z.id)}"${f.zone_id===z.id?' selected':''}>${esc(z.name)} (${esc(z.id.slice(0,8))}…)</option>`).join('')}
         </select>`
      : `<input type="text" placeholder="${acctSaved?'(saved)':'Paste zone ID or verify token to auto-load'}" value="${esc(f.zone_id||'')}" data-form="cloudflare" data-field="zone_id"/>`;

    const tunnelInput = d.tunnels.length
      ? `<select class="cf-select" data-form="cloudflare" data-field="tunnel_id">
           <option value="">Select tunnel…</option>
           ${d.tunnels.map(t => `<option value="${esc(t.id)}"${f.tunnel_id===t.id?' selected':''}>${esc(t.name)}</option>`).join('')}
         </select>`
      : `<input type="text" placeholder="${acctSaved?'(saved)':'Paste tunnel ID or verify token + account to auto-load'}" value="${esc(f.tunnel_id||'')}" data-form="cloudflare" data-field="tunnel_id"/>`;

    fields = `<div class="cfg-fields">
      ${chk('Enable Cloudflare sync', f.enabled, 'cloudflare', 'enabled')}

      <div class="cf-step-block">
        <div class="cf-step-head"><span class="cf-step-num">1</span>API Token</div>
        <div class="cf-inline-row">
          <input type="password" class="cf-token-inp"
            placeholder="${tokenSaved ? '(saved — enter to replace)' : 'Paste API token…'}"
            data-form="cloudflare" data-field="api_token"/>
          <button class="cfg-btn cf-verify-btn" data-action="cf-discover"${d.loading||!mut?' disabled':''}>
            ${d.loading ? '…' : 'Verify →'}
          </button>
        </div>
        ${d.verifyMsg ? `<div class="cf-result ${d.verifyOk?'ok':'err'}">${esc(d.verifyMsg)}</div>` : ''}
      </div>

      <div class="cf-step-block">
        <div class="cf-step-head"><span class="cf-step-num">2</span>Account &amp; Tunnel
          <small class="cf-step-hint">${d.tunnels.length ? `${d.tunnels.length} tunnel${d.tunnels.length!==1?'s':''} found` : 'verify token to auto-load'}</small>
        </div>
        <div class="cfg-field-row"><label>ACCOUNT ID</label>
          <input type="text" placeholder="${acctSaved?'(saved)':'32-char hex account ID'}" value="${esc(f.account_id||'')}" data-form="cloudflare" data-field="account_id"/>
        </div>
        <div class="cfg-field-row"><label>TUNNEL</label>${tunnelInput}</div>
        <div class="cfg-field-row"><label>ZONE</label>${zoneInput}</div>
      </div>

      <div class="cf-step-block">
        <div class="cf-step-head"><span class="cf-step-num">3</span>Connection</div>
        ${field('CADDY URL', inp('url','http://127.0.0.1:80', f.caddy_service_url, 'cloudflare', 'caddy_service_url'))}
        ${chk('Skip TLS verification', f.insecure, 'cloudflare', 'insecure')}
      </div>

      ${trHtml}
      <div class="cfg-actions">
        <button class="cfg-btn" data-action="test-cfg" data-svc="cloudflare"${mut?'':' disabled'}>Test saved config</button>
        <button class="cfg-btn save" data-action="save-cfg" data-svc="cloudflare"${mut?'':' disabled'}>Save</button>
      </div>
    </div>`;
  }

  return `<article class="config-card ${tone}">
    <div class="cfg-card-head">
      <div class="cfg-svc-name">
        <div class="cfg-svc-icon">${SVC_ICONS[svc] || '◉'}</div>
        <strong>${esc(s.label || svc)}</strong>
      </div>
      <div class="cfg-card-badges">${statusBadge}${srcBadge}</div>
    </div>
    ${s.endpoint ? `<div class="cfg-endpoint">${esc(s.endpoint)}</div>` : ''}
    <div class="cfg-missing">${missingHtml}</div>
    ${fields}
  </article>`;
}

function tConfigTestSummary(c) {
  if (!window.UNBOUNDCLI_TEST_HOOKS || !c) return '';
  const unbound = c.summary?.unbound || {};
  const apiKey = unbound.fields?.api_key_set ? 'API Key Set' : 'API Key Missing';
  return `<div class="test-hook-summary" hidden>
    Save target: ${esc(c.save_target || 'default')}
    ${esc(unbound.label || 'OPNSense / Unbound')}
    ${apiKey}
    Set OPNSense
    Test OPNSense
    Test Caddy
    Defaults
    <span>Cloudflare<span></span></span>
  </div>`;
}

// Re-fetch the CF wizard sync plan using current wizard options.
async function cfWizardRefreshPlan() {
  const w = S.cfWizard;
  S.cfWizard.loading = true;
  S.cfWizard.log = '';
  render();
  try {
    const params = new URLSearchParams({
      service: 'cloudflare',
      hostname: w.hostname,
      origin_mode: w.originMode || 'via-caddy',
      no_tls_verify: w.noTLSVerify ? 'true' : 'false',
      disable_chunked_encoding: w.disableChunkedEncoding ? 'true' : 'false',
      tunnel_id: w.selectedTunnelId || '',
    });
    const plan = await api(`/api/sync/plan?${params}`);
    S.cfWizard.actions   = plan.actions    || [];
    S.cfWizard.planId    = plan.plan_id    || '';
    S.cfWizard.actionIds = plan.action_ids || [];
  } catch (err) {
    S.cfWizard.log = `Failed to load plan: ${err.message}`;
  }
  S.cfWizard.loading = false;
  render();
}

function tCfWizardOptions(w, cfg) {
  // Tunnel selector
  const tunnels = w.availableTunnels || [];
  const configuredTunnelId = cfg?.summary?.cloudflare?.details?.tunnel_id || '';
  let tunnelSection = '';
  if (tunnels.length > 1) {
    const opts = tunnels.map(t => {
      const isDefault = t.id === configuredTunnelId;
      const sel = (w.selectedTunnelId || configuredTunnelId) === t.id ? ' selected' : '';
      return `<option value="${esc(t.id)}"${sel}>${esc(t.name)}${isDefault ? ' (default)' : ''}</option>`;
    }).join('');
    tunnelSection = `
      <div class="wiz-opt-row">
        <label class="wiz-opt-label">Tunnel</label>
        <select class="wiz-opt-select" id="wiz-tunnel-select">${opts}</select>
      </div>`;
  } else if (tunnels.length === 1) {
    tunnelSection = `
      <div class="wiz-opt-row">
        <label class="wiz-opt-label">Tunnel</label>
        <span class="wiz-opt-value">${esc(tunnels[0].name)}</span>
      </div>`;
  }

  // Origin mode toggle
  const viaCaddyActive  = w.originMode !== 'direct';
  const directActive    = w.originMode === 'direct';
  const originSection = `
    <div class="wiz-opt-row">
      <label class="wiz-opt-label">Origin</label>
      <div class="wiz-seg">
        <button class="wiz-seg-btn${viaCaddyActive ? ' active' : ''}" data-action="cf-wiz-origin" data-mode="via-caddy" title="CF tunnel → Caddy → service (recommended)">Via Caddy</button>
        <button class="wiz-seg-btn${directActive  ? ' active' : ''}" data-action="cf-wiz-origin" data-mode="direct"    title="CF tunnel → service directly (bypasses Caddy)">Direct to service</button>
      </div>
      <span class="wiz-opt-hint">${directActive ? 'CF → service (bypasses Caddy)' : 'CF → Caddy → service'}</span>
    </div>`;

  // TLS / advanced options checkboxes
  const tlsChecked   = w.noTLSVerify             ? ' checked' : '';
  const chunkChecked = w.disableChunkedEncoding   ? ' checked' : '';
  const viaCaddy     = w.originMode !== 'direct';
  const originSNI    = viaCaddy ? `
      <div class="wiz-opt-row wiz-opt-subrow">
        <label class="wiz-opt-label">SNI</label>
        <span class="wiz-opt-hint">cloudflared will use <strong>${esc(w.hostname)}</strong> as the TLS server name</span>
      </div>` : '';
  const optionsSection = `
    <div class="wiz-opt-row">
      <label class="wiz-opt-label">Options</label>
      <div class="wiz-opt-checks">
        <label class="wiz-opt-check">
          <input type="checkbox" id="wiz-no-tls-verify"${tlsChecked}>
          <span>Skip TLS verify${w.noTLSVerify ? ' <span class="wiz-opt-warn visible">⚠ insecure</span>' : ''}</span>
        </label>
        <label class="wiz-opt-check">
          <input type="checkbox" id="wiz-disable-chunked"${chunkChecked}>
          <span>Disable chunked encoding <span class="wiz-opt-hint2">for WSGI / legacy backends</span></span>
        </label>
      </div>
    </div>${originSNI}`;

  return `<div class="wiz-options">${tunnelSection}${originSection}${optionsSection}</div>`;
}

function tCfWizard() {
  const w = S.cfWizard;
  if (!w.open) return '';
  const cfg = S.config;
  const configuredTunnelId = cfg?.summary?.cloudflare?.details?.tunnel_id || '';
  const tunnels = w.availableTunnels || [];
  // Resolve tunnel name for display
  const selectedId = w.selectedTunnelId || configuredTunnelId;
  const selectedTunnel = tunnels.find(t => t.id === selectedId);
  const tunnelName = selectedTunnel?.name || cfg?.cloudflare?.tunnel_id || 'configured tunnel';
  const serviceURL = cfg?.summary?.cloudflare?.details?.caddy_service_url || (cfg?.caddy?.server_ip ? `http://${cfg.caddy.server_ip}:80` : '');

  let body = '';
  if (w.loading) {
    body = `<div class="wiz-loading"><div class="wiz-spinner"></div><span>Fetching sync plan…</span></div>`;
  } else if (w.applied) {
    body = `<div class="wiz-result ok"><strong>✓ Synced!</strong><p>${esc(w.hostname)} has been added to the Cloudflare tunnel.</p>
      <pre class="wiz-log">${esc(w.log)}</pre></div>`;
  } else if (w.actions.length === 0) {
    body = `<div class="wiz-result ok"><strong>✓ Already routed</strong><p>${esc(w.hostname)} is already in the Cloudflare tunnel — nothing to do.</p></div>`;
  } else {
    const a = w.actions[0];
    const verb = a.type === 'add' ? 'Add to' : a.type === 'update' ? 'Update in' : 'Remove from';
    const svcURL = a.new_service || a.old_service || serviceURL;
    const noTLSNote = a.no_tls_verify ? `<tr><th>TLS verify</th><td class="wiz-warn">⚠ disabled</td></tr>` : '';
    const sniNote = a.origin_server_name ? `<tr><th>TLS SNI</th><td><code>${esc(a.origin_server_name)}</code></td></tr>` : '';
    const chunkedNote = a.disable_chunked_encoding ? `<tr><th>Chunked enc.</th><td class="wiz-warn">disabled</td></tr>` : '';
    body = `
      <div class="wiz-preview">
        <div class="wiz-action-row">
          <span class="wiz-verb ${a.type}">${verb}</span>
          <span class="wiz-tunnel">⟶ ${esc(tunnelName)}</span>
        </div>
        <table class="wiz-detail-table">
          <tr><th>Hostname</th><td><code>${esc(a.hostname)}</code></td></tr>
          ${svcURL ? `<tr><th>Origin service</th><td><code>${esc(svcURL)}</code></td></tr>` : ''}
          ${a.new_http_host_header ? `<tr><th>Host header</th><td><code>${esc(a.new_http_host_header)}</code></td></tr>` : ''}
          ${sniNote}${noTLSNote}${chunkedNote}
          ${a.details ? `<tr><th>Note</th><td>${esc(a.details)}</td></tr>` : ''}
        </table>
        ${w.log ? `<pre class="wiz-log error">${esc(w.log)}</pre>` : ''}
      </div>`;
  }

  const canApply = w.actions.length > 0 && !w.applied && !w.loading && wc().mutationEnabled;

  return `<div class="cf-wizard-backdrop" data-action="cf-wizard-close"></div>
  <div class="cf-wizard" role="dialog" aria-modal="true" aria-labelledby="wiz-title">
    <div class="wiz-head">
      <div>
        <span class="wiz-cf-icon">☁</span>
        <h2 id="wiz-title">Route to Cloudflare</h2>
      </div>
      <button class="wiz-close" data-action="cf-wizard-close" aria-label="Close">✕</button>
    </div>
    <div class="wiz-hostname">${esc(w.hostname)}</div>
    ${!w.applied ? tCfWizardOptions(w, cfg) : ''}
    <div class="wiz-body">${body}</div>
    <div class="wiz-foot">
      <button class="wiz-btn secondary" data-action="cf-wizard-close">Cancel</button>
      ${canApply ? `<button class="wiz-btn primary" data-action="cf-wizard-apply">Sync to Cloudflare</button>` : ''}
      ${w.applied ? `<button class="wiz-btn secondary" data-action="cf-wizard-close">Done</button>` : ''}
    </div>
  </div>`;
}

function tConfigModal() {
  if (!S.configOpen && !window.UNBOUNDCLI_TEST_HOOKS) return '';
  const c = S.config;
  const svcs = [
    { key:'caddy',      tone:'green',  label:'Caddy'      },
    { key:'unbound',    tone:'blue',   label:'Unbound'    },
    { key:'adguard',    tone:'teal',   label:'AdGuard'    },
    { key:'dhcp',       tone:'yellow', label:'DHCP'       },
    { key:'cloudflare', tone:'violet', label:'Cloudflare' },
  ];
  const active = S.configTab || 'caddy';
  const activeSvc = svcs.find(s => s.key === active) || svcs[0];
  const hidden = !S.configOpen && !!window.UNBOUNDCLI_TEST_HOOKS;

  const tabBar = svcs.map(s => {
    const st = c?.summary?.[s.key];
    const dot = st?.client_ready ? 'green' : st?.enabled ? 'amber' : 'muted';
    return `<button class="cfg-tab${s.key===active?' active':''}" data-action="cfg-tab" data-tab="${s.key}">
      <i class="cfg-tab-dot ${dot}"></i>${s.label}
    </button>`;
  }).join('');

  return `<div id="config-panel" class="${hidden ? 'config-modal ' : 'config-modal'}"${hidden ? ' hidden' : ''} role="dialog" aria-modal="true" aria-labelledby="cfg-title">
    <div class="config-backdrop" data-action="close-config"></div>
    <section class="config-sheet panel">
      <header class="config-sheet-header">
        <div>
          <strong id="cfg-title">${ICON.gear} Configuration</strong>
          <code class="save-target">${esc(c?.save_target||'default')}</code>
        </div>
        <button id="config-close" data-action="close-config">Close</button>
      </header>
      <nav class="cfg-tabs">${tabBar}</nav>
      <div class="config-status ${esc(S.configStatusKind)}" role="status" aria-live="polite">${esc(S.configStatus)}</div>
      <div class="cfg-tab-content">
        ${c ? tCfgCard(activeSvc.key, c, activeSvc.tone) : '<p style="color:var(--text-muted);padding:20px">Loading…</p>'}
      </div>
      ${tConfigTestSummary(c)}
    </section>
  </div>`;
}

// ── Render ─────────────────────────────────────────────────────────────────
function tRowModal() {
  const m = S.rowModal;
  if (!m.open) return '';

  const mut = wc().mutationEnabled;
  const entry = S.entries.find(e => e.hostname === m.hostname);
  const ub = entry?.unbound_status, ag = entry?.adguard_status, cf = entry?.cloudflare_status;
  const isStale = entry?.overall_status === 4;
  const ops = m.svcOps || {};

  // Per-service row (Sync or Remove button depending on configured state)
  function svcRow(key, label, status) {
    const op = ops[key] || {};
    const isConfigured = !!status?.configured;
    const currentTxt = isConfigured ? esc(svcText(status)) : '—';
    const toneCls = svcTone(status);

    let actionHtml = '';
    if (op.loading) {
      actionHtml = `<span class="svc-op-busy">Applying…</span>`;
    } else if (op.log) {
      actionHtml = `<span class="svc-op-done">${esc(op.log.replace(/\n[\s\S]*/,'').slice(0,60))}</span>
        <button class="svc-op-redo" data-action="modal-svc-redo" data-svc="${key}">↺</button>`;
    } else if (isConfigured && mut) {
      actionHtml = `<button class="svc-op-remove" data-action="modal-svc-remove" data-svc="${key}">Remove</button>`;
    } else if (!isConfigured && entry?.caddy_upstream && mut) {
      actionHtml = `<button class="svc-op-sync" data-action="modal-svc-sync" data-svc="${key}">Sync ↑</button>`;
    }

    return `<div class="modal-svc-row">
      <span class="modal-svc-name">${label}</span>
      <span class="modal-svc-status ${toneCls}">${currentTxt}</span>
      <div class="modal-svc-btns">${actionHtml}</div>
    </div>`;
  }

  // Host info + per-service rows
  const headerSection = entry ? `<div class="row-modal-host-info">
    <div class="row-modal-meta-row">
      ${entry.caddy_upstream ? `<span class="row-modal-meta-item"><span>Upstream</span><strong>${esc(entry.caddy_upstream)}</strong></span>` : ''}
      ${entry.dns_resolved ? `<span class="row-modal-meta-item"><span>DNS</span><strong class="dns-result ${dnsCls(entry.dns_resolved)}">${esc(entry.dns_resolved)}</strong></span>` : ''}
      ${cf?.configured ? `<span class="row-modal-meta-item"><span>CF</span><strong class="violet">${esc(cf.tunnel_name || 'Routed')}</strong></span>` : ''}
    </div>
  </div>
  <div class="modal-svc-section">
    ${svcRow('unbound', 'Unbound', ub)}
    ${svcRow('adguard', 'AdGuard', ag)}
    ${!cf?.configured && S.config?.summary?.cloudflare?.client_ready ? `<div class="modal-svc-row">
      <span class="modal-svc-name">Cloudflare</span>
      <span class="modal-svc-status missing">—</span>
      <div class="modal-svc-btns"><button class="svc-op-cf" data-action="cf-wizard" data-hostname="${esc(m.hostname)}">+ Route</button></div>
    </div>` : ''}
    ${cf?.configured ? `<div class="modal-svc-row">
      <span class="modal-svc-name">Cloudflare</span>
      <span class="modal-svc-status ok">${esc(cf.service || cf.tunnel_name || 'Routed')}</span>
      <div class="modal-svc-btns"></div>
    </div>` : ''}
  </div>` : '';

  // Stale cleanup or plan section
  let planHtml = '';
  if (isStale) {
    planHtml = `<div class="modal-plan-section">
      <div class="modal-plan-header">Stale DNS entry — hostname no longer in Caddy</div>
      ${mut ? `<div class="modal-stale-btns">
        ${ub?.configured ? `<button class="svc-op-remove" data-action="modal-svc-remove" data-svc="unbound"${(ops.unbound?.loading)?'disabled':''}>Remove from Unbound</button>` : ''}
        ${ag?.configured ? `<button class="svc-op-remove" data-action="modal-svc-remove" data-svc="adguard"${(ops.adguard?.loading)?'disabled':''}>Remove from AdGuard</button>` : ''}
      </div>` : ''}
    </div>`;
  } else if (m.loading) {
    planHtml = `<div class="row-modal-loading"><div class="wiz-spinner"></div><span>Fetching plan…</span></div>`;
  } else if (m.applyLog) {
    planHtml = `<div class="modal-plan-section"><pre class="row-modal-result">${esc(m.applyLog)}</pre></div>`;
  } else if (m.actions.length) {
    const enabledCount = m.actions.filter((_, i) => !m.disabledIndices.has(i)).length;
    const rows = m.actions.map((a, i) => {
      const enabled = !m.disabledIndices.has(i);
      const typeClass = a.type === 'delete' ? 'plan-del' : a.type === 'add' ? 'plan-add' : 'plan-upd';
      const svc = (a.service || '').toUpperCase();
      const detail = a.new_ip || a.new_service || a.old_ip || a.details || '';
      const badge = a.type === 'delete' ? 'Remove' : a.type === 'add' ? 'Add' : 'Update';
      return `<label class="row-modal-action ${typeClass}${enabled ? '' : ' plan-row-disabled'}">
        <input type="checkbox" class="plan-row-check" data-action="toggle-row-modal-action" data-idx="${i}"${enabled ? ' checked' : ''}>
        <span class="row-modal-badge ${a.type}">${badge}</span>
        <span class="row-modal-svc">${esc(svc)}</span>
        <span class="row-modal-detail">${esc(detail)}</span>
      </label>`;
    }).join('');
    const applyBtn = mut
      ? `<button class="row-modal-apply-btn" data-action="row-modal-apply"${(m.applying || enabledCount === 0) ? ' disabled' : ''}>
           ${m.applying ? 'Applying…' : `Apply ${enabledCount} change${enabledCount !== 1 ? 's' : ''}`}
         </button>`
      : `<button class="row-modal-apply-btn" disabled>Apply (read-only)</button>`;
    planHtml = `<div class="modal-plan-section">
      <div class="modal-plan-header">Pending changes</div>
      <div class="row-modal-actions">${rows}</div>
      <div class="row-modal-footer">${applyBtn}</div>
    </div>`;
  }

  return `<div class="row-modal-overlay" data-action="row-modal-close" id="row-modal-overlay">
    <div class="row-modal" role="dialog" aria-modal="true" id="row-modal-inner">
      <div class="row-modal-header">
        <div>
          <div class="row-modal-title">${esc(m.hostname)}</div>
          <div class="row-modal-subtitle">${isStale ? 'Stale — clean up DNS records' : m.loading ? 'Loading…' : m.actions.length ? `${m.actions.length} change${m.actions.length!==1?'s':''} pending` : 'Manage services'}</div>
        </div>
        <button class="row-modal-close" data-action="row-modal-close">✕</button>
      </div>
      <div class="row-modal-body">
        ${headerSection}
        ${planHtml}
      </div>
    </div>
  </div>`;
}

function render() {
  const root = document.getElementById('root');
  if (!root) return;

  const ep          = document.getElementById('entries-panel');
  const tableScroll = ep?.scrollTop || 0;
  const logEl       = document.getElementById('sync-log');
  const logAtBot    = !logEl || logEl.scrollTop + logEl.clientHeight >= logEl.scrollHeight - 4;
  const searchFocus = document.activeElement?.id === 'search';

  const entries = filteredEntries();

  const enabled = S.config?.enabled || {};
  root.innerHTML = `<div id="app-shell" data-e2e="${S.e2eDone ? 'done' : 'app-shell'}"
    data-loading="${S.loading ? 'true' : 'false'}"
    data-mobile="${window.innerWidth <= 600 ? 'true' : 'false'}"
    data-mutation-enabled="${wc().mutationEnabled ? 'true' : 'false'}"
    data-adguard-enabled="${enabled.adguard ? 'true' : 'false'}"
    data-cloudflare-enabled="${enabled.cloudflare ? 'true' : 'false'}">
    ${tTopbar()}
    <main class="dashboard-shell">
      <div id="top-progress" class="top-progress${S.loading ? '' : ' idle'}" aria-hidden="${S.loading ? 'false' : 'true'}">
        <span id="top-progress-title">${S.loading ? 'Loading service status' : 'Idle'}</span>
        <div class="progress-track"><span></span></div>
      </div>
      ${S.loading ? `<div class="loading-panel">
        <div class="loading-copy"><span>Loading service status</span><strong>Scanning Caddy routes and DNS services</strong></div>
        <div class="progress-track"><span></span></div>
      </div>` : ''}
      <div class="message ${esc(S.msgKind)}${S.message ? '' : ' empty'}" id="message" aria-live="polite">${esc(S.message)}</div>
      ${tMetrics()}
      <section class="workspace-grid">
        <section class="content-stack">
          ${tToolbar(entries)}
          ${tTable(entries)}
        </section>
      </section>
    </main>
    ${tConfigModal()}
    ${tCfWizard()}
    ${tRowModal()}
  </div>`;

  const newEp = document.getElementById('entries-panel');
  if (newEp) newEp.scrollTop = tableScroll;
  const newLog = document.getElementById('sync-log');
  if (newLog && logAtBot) newLog.scrollTop = newLog.scrollHeight;
  if (searchFocus) {
    const ns = document.getElementById('search');
    if (ns) { ns.focus(); const v = ns.value; ns.setSelectionRange(v.length, v.length); }
  }
}

// ── Event delegation ───────────────────────────────────────────────────────
document.addEventListener('click', async ev => {
  const el = ev.target.closest('[data-action]');
  if (!el) return;
  const a = el.dataset.action;

  if (a === 'refresh')        { await refresh(); return; }
  if (a === 'open-config')    { S.configOpen = true;  render(); return; }
  if (a === 'close-config')   { S.configOpen = false; render(); return; }
  if (a === 'filter-svc')     { S.serviceFilter = el.dataset.svc || 'all'; render(); return; }
  if (a === 'filter-status') {
    const st = el.dataset.status || 'all';
    if (st === 'all') { S.statusFilter = 'all'; S.statusFilterInverse = false; }
    else if (S.statusFilter !== st) { S.statusFilter = st; S.statusFilterInverse = false; }      // 1st click: filter
    else if (!S.statusFilterInverse) { S.statusFilterInverse = true; }                            // 2nd click: invert
    else { S.statusFilter = 'all'; S.statusFilterInverse = false; }                               // 3rd click: clear
    render(); return;
  }
  if (a === 'cfg-tab')        { S.configTab = el.dataset.tab; render(); return; }

  if (a === 'cf-wizard') {
    ev.stopPropagation();
    const hostname = el.dataset.hostname;
    const cfDetails = S.config?.summary?.cloudflare?.details || {};
    const defaultTunnelId = cfDetails.tunnel_id || '';
    S.cfWizard = {
      open: true, hostname, loading: true, actions: [], planId: '', actionIds: [], log: '', applied: false,
      originMode: 'via-caddy', noTLSVerify: false, disableChunkedEncoding: false, selectedTunnelId: defaultTunnelId,
      availableTunnels: S.cfWizard.availableTunnels || [], tunnelsLoading: false,
    };
    render();
    // Fetch available tunnels once (reuse if already loaded)
    if (!S.cfWizard.availableTunnels.length) {
      S.cfWizard.tunnelsLoading = true;
      try {
        const tunnels = await api('/api/cloudflare/tunnels');
        S.cfWizard.availableTunnels = Array.isArray(tunnels) ? tunnels : [];
      } catch (_) { /* ignore — fall back to no selector */ }
      S.cfWizard.tunnelsLoading = false;
    }
    await cfWizardRefreshPlan();
    return;
  }
  if (a === 'cf-wizard-close') {
    S.cfWizard = { ...S.cfWizard, open: false };
    render(); return;
  }
  if (a === 'cf-wiz-origin') {
    const newMode = el.dataset.mode || 'via-caddy';
    S.cfWizard.originMode = newMode;
    // Via Caddy uses a local Caddy cert → auto-enable NoTLSVerify
    if (newMode === 'via-caddy') S.cfWizard.noTLSVerify = true;
    await cfWizardRefreshPlan(); return;
  }
  if (a === 'cf-wizard-apply') {
    S.cfWizard.loading = true; render();
    try {
      const body = S.cfWizard.planId
        ? { dry_run: false, plan_id: S.cfWizard.planId, action_ids: S.cfWizard.actionIds }
        : { dry_run: false, actions: S.cfWizard.actions };
      const { result: r } = await api('/api/sync/apply', { method: 'POST', body: JSON.stringify(body) });
      S.cfWizard.log = fmtApplyResult(r, false);
      S.cfWizard.applied = !r?.errors?.length;
      if (S.cfWizard.applied) await refresh();
    } catch (err) {
      S.cfWizard.log = `Apply error: ${err.message}`;
    }
    S.cfWizard.loading = false;
    render(); return;
  }

  if (a === 'select-row') {
    const tr = el.closest('tr');
    const h  = tr?.dataset.hostname ?? el.dataset.hostname;
    if (h && h !== S.selectedHostname) {
      S.selectedHostname = h;
      S.inspectorPlan = { loading: false, actions: [], planId: '', actionIds: [], log: '', applying: false, applyLog: '' };
    }
    if (h) render();
    return;
  }
  if (a === 'row-modify') {
    ev.stopPropagation();
    const hostname = el.dataset.hostname;
    S.rowModal = { open: true, hostname, loading: true, actions: [], planId: '', actionIds: [], applying: false, applyLog: '', disabledIndices: new Set(), autoApply: false, unsyncLoading: '', svcOps: {} };
    render();
    try {
      const data = await api(`/api/sync/plan?hostname=${encodeURIComponent(hostname)}&service=`);
      const plan = data.plan || data;
      S.rowModal = { ...S.rowModal, loading: false, actions: plan.actions || [], planId: plan.plan_id || data.plan_id || '', actionIds: plan.action_ids || data.action_ids || [], disabledIndices: new Set() };
    } catch (err) {
      S.rowModal = { ...S.rowModal, loading: false, applyLog: `Error: ${err.message}` };
    }
    render();
    return;
  }
  if (a === 'row-sync-direct') {
    ev.stopPropagation();
    const hostname = el.dataset.hostname;
    S.rowModal = { open: true, hostname, loading: true, actions: [], planId: '', actionIds: [], applying: false, applyLog: '', disabledIndices: new Set(), autoApply: true, unsyncLoading: '', svcOps: {} };
    render();
    try {
      const data = await api(`/api/sync/plan?hostname=${encodeURIComponent(hostname)}&service=`);
      const plan = data.plan || data;
      const actions = plan.actions || [];
      const planId = plan.plan_id || data.plan_id || '';
      const actionIds = plan.action_ids || data.action_ids || [];
      if (!actions.length) {
        S.rowModal = { ...S.rowModal, loading: false, actions: [], planId: '', actionIds: [], applyLog: '✓ Already in sync — nothing to do.' };
      } else {
        S.rowModal = { ...S.rowModal, loading: false, actions, planId, actionIds, applying: true };
        render();
        try {
          const body = planId
            ? { dry_run: false, plan_id: planId, action_ids: actionIds }
            : { dry_run: false, actions };
          const { result: r } = await api('/api/sync/apply', { method: 'POST', body: JSON.stringify(body) });
          S.rowModal = { ...S.rowModal, applying: false, applyLog: fmtApplyResult(r, false), actions: [], planId: '', actionIds: [] };
          await refresh();
        } catch (err) {
          S.rowModal = { ...S.rowModal, applying: false, applyLog: `Apply error: ${err.message}` };
        }
      }
    } catch (err) {
      S.rowModal = { ...S.rowModal, loading: false, applyLog: `Error: ${err.message}` };
    }
    render();
    return;
  }
  if (a === 'modal-svc-sync' || a === 'modal-svc-remove' || a === 'modal-svc-redo') {
    ev.stopPropagation();
    const svc = el.dataset.svc;
    const isRemove = a === 'modal-svc-remove';
    const isRedo   = a === 'modal-svc-redo';
    // Redo just clears the result so buttons reappear
    if (isRedo) { S.rowModal.svcOps = { ...S.rowModal.svcOps, [svc]: {} }; render(); return; }
    S.rowModal.svcOps = { ...(S.rowModal.svcOps||{}), [svc]: { loading: true, log: '' } };
    render();
    try {
      const qs = `hostname=${encodeURIComponent(S.rowModal.hostname)}&service=${encodeURIComponent(svc)}${isRemove ? '&unsync=true' : ''}`;
      const data = await api(`/api/sync/plan?${qs}`);
      const plan = data.plan || data;
      if (!plan.actions?.length) {
        S.rowModal.svcOps = { ...(S.rowModal.svcOps||{}), [svc]: { loading: false, log: isRemove ? '✓ Already removed' : '✓ Already in sync' } };
      } else {
        const body = plan.plan_id
          ? { dry_run: false, plan_id: plan.plan_id, action_ids: plan.action_ids }
          : { dry_run: false, actions: plan.actions };
        const { result: r } = await api('/api/sync/apply', { method: 'POST', body: JSON.stringify(body) });
        S.rowModal.svcOps = { ...(S.rowModal.svcOps||{}), [svc]: { loading: false, log: fmtApplyResult(r, false) } };
        await refresh();
      }
    } catch (err) {
      S.rowModal.svcOps = { ...(S.rowModal.svcOps||{}), [svc]: { loading: false, log: `Error: ${err.message}` } };
    }
    render();
    return;
  }
  if (a === 'row-modal-close') {
    // Close button click → always close
    if (el.classList.contains('row-modal-close')) { S.rowModal = { ...S.rowModal, open: false }; render(); return; }
    // Backdrop click → only close when the actual overlay element was clicked, not a child bubbling up
    if (ev.target !== el) return;
    S.rowModal = { ...S.rowModal, open: false };
    render();
    return;
  }
  if (a === 'toggle-row-modal-action') {
    const idx = parseInt(el.dataset.idx, 10);
    const dis = new Set(S.rowModal.disabledIndices);
    if (dis.has(idx)) dis.delete(idx); else dis.add(idx);
    S.rowModal = { ...S.rowModal, disabledIndices: dis };
    render();
    return;
  }
  if (a === 'row-modal-apply') {
    const m = S.rowModal;
    const enabledIds = (m.actionIds || []).filter((_, i) => !m.disabledIndices.has(i));
    if (!enabledIds.length) return;
    S.rowModal = { ...m, applying: true };
    render();
    try {
      const body = m.planId
        ? { dry_run: false, plan_id: m.planId, action_ids: enabledIds }
        : { dry_run: false, actions: m.actions.filter((_, i) => !m.disabledIndices.has(i)) };
      const { result: r } = await api('/api/sync/apply', { method: 'POST', body: JSON.stringify(body) });
      const log = fmtApplyResult(r, false);
      S.rowModal = { ...S.rowModal, applying: false, applyLog: log, actions: [], planId: '', actionIds: [] };
      await refresh();
    } catch (err) {
      S.rowModal = { ...S.rowModal, applying: false, applyLog: `Apply error: ${err.message}` };
    }
    render();
    return;
  }
  if (a === 'preview-sync')      { await fetchPlan(S.syncService); return; }  // global — no hostname scope
  if (a === 'dry-run')           { await applySync(true);  return; }
  if (a === 'sync-now')          { await applySync(false); return; }
  if (a === 'inspector-preview') { await fetchInspectorPlan(S.selectedHostname); return; }
  if (a === 'inspector-apply')   { await applyInspectorPlan(); return; }
  if (a === 'toggle-insp-action') {
    const idx = parseInt(el.dataset.idx, 10);
    const p = S.inspectorPlan;
    const dis = new Set(p.disabledIndices);
    if (dis.has(idx)) dis.delete(idx); else dis.add(idx);
    S.inspectorPlan = { ...p, disabledIndices: dis };
    render(); return;
  }
  if (a === 'unsync-svc') {
    ev.stopPropagation();
    const hostname = el.dataset.hostname, svc = el.dataset.svc;
    S.selectedHostname = hostname;
    // Fetch an unsync plan (generates delete actions) scoped to this hostname + service
    await fetchInspectorPlan(hostname, { service: svc, unsync: true });
    return;
  }

  if (a === 'probe') {
    ev.stopPropagation();
    const upstream = el.dataset.upstream, hostname = el.dataset.hostname;
    S.probe = { loading: true, result: null, hostname };
    render();
    try {
      const r = await api(`/api/probe?upstream=${encodeURIComponent(upstream)}&hostname=${encodeURIComponent(hostname)}`);
      S.probe = { loading: false, result: r, hostname };
    } catch (err) {
      S.probe = { loading: false, result: { reachable: false, error: err.message }, hostname };
    }
    render(); return;
  }
  if (a === 'test-cfg')  { await testConfig(el.dataset.svc); return; }
  if (a === 'save-cfg')  { await doSave(el.dataset.svc); return; }
  if (a === 'cf-discover') {
    S.cfDiscover = { ...S.cfDiscover, loading: true, verifyMsg: '' };
    render();
    try {
      const data = await api('/api/cloudflare/discover', {
        method: 'POST',
        body: JSON.stringify({ token: S.forms.cloudflare.api_token, account_id: S.forms.cloudflare.account_id })
      });
      if (data.error) {
        S.cfDiscover = { loading: false, verifyOk: false, verifyMsg: data.error, accounts: [], tunnels: [], zones: [] };
      } else {
        const nz = data.zones?.length || 0;
        const nt = data.tunnels?.length || 0;
        S.cfDiscover = {
          loading: false, verifyOk: true,
          verifyMsg: `✓ Token valid — ${nz} zone${nz!==1?'s':''}, ${nt} tunnel${nt!==1?'s':''}`,
          accounts: data.accounts || [], tunnels: data.tunnels || [], zones: data.zones || []
        };
      }
    } catch(err) {
      S.cfDiscover = { loading: false, verifyOk: false, verifyMsg: `Error: ${err.message}`, accounts: [], tunnels: [], zones: [] };
    }
    render();
    return;
  }
});

document.addEventListener('input', ev => {
  if (ev.target.id === 'search') { S.search = ev.target.value; render(); return; }
  const { form, field } = ev.target.dataset;
  if (form && field && S.forms[form]) {
    S.forms[form] = { ...S.forms[form], [field]: ev.target.dataset.type === 'checkbox' ? ev.target.checked : ev.target.value };
  }
});

document.addEventListener('change', async ev => {
  if (ev.target.id === 'status-filter') { S.statusFilter = ev.target.value; S.statusFilterInverse = false; render(); return; }
  if (ev.target.id === 'sync-service')  { S.syncService  = ev.target.value; return; }
  if (ev.target.id === 'wiz-tunnel-select' && S.cfWizard.open) {
    S.cfWizard.selectedTunnelId = ev.target.value;
    await cfWizardRefreshPlan(); return;
  }
  if (ev.target.id === 'wiz-no-tls-verify' && S.cfWizard.open) {
    S.cfWizard.noTLSVerify = ev.target.checked;
    await cfWizardRefreshPlan(); return;
  }
  if (ev.target.id === 'wiz-disable-chunked' && S.cfWizard.open) {
    S.cfWizard.disableChunkedEncoding = ev.target.checked;
    await cfWizardRefreshPlan(); return;
  }
  const { form, field } = ev.target.dataset;
  if (form && field && S.forms[form]) {
    S.forms[form] = { ...S.forms[form], [field]: ev.target.dataset.type === 'checkbox' ? ev.target.checked : ev.target.value };
  }
});

document.addEventListener('keydown', ev => {
  if (ev.key === 'Escape' && S.rowModal.open) { S.rowModal = { ...S.rowModal, open: false }; render(); return; }
  if (ev.key === 'Escape' && S.cfWizard.open) { S.cfWizard = { ...S.cfWizard, open: false }; render(); return; }
  if (ev.key === 'Escape' && S.configOpen) { S.configOpen = false; render(); return; }
  if ((ev.key === 'Enter' || ev.key === ' ') && ev.target.dataset.action === 'select-row') {
    ev.preventDefault();
    const h = ev.target.closest('tr')?.dataset.hostname ?? ev.target.dataset.hostname;
    if (h) { S.selectedHostname = h; render(); }
  }
});

function e2eActions() {
  if (!window.UNBOUNDCLI_TEST_HOOKS) return [];
  return new URLSearchParams(window.location.search).get('e2e')?.split(',').filter(Boolean) || [];
}

async function runE2EActions() {
  const actions = e2eActions();
  if (!actions.length || actions.includes('holdloading')) return;

  for (const action of actions) {
    if (action.startsWith('filter:')) {
      S.statusFilter = action.slice('filter:'.length) || 'all';
      render();
      continue;
    }
    if (action.startsWith('search:')) {
      S.search = action.slice('search:'.length);
      render();
      continue;
    }
    if (action.startsWith('preview:')) {
      S.syncService = action.slice('preview:'.length) || 'all';
      await fetchPlan(S.syncService, '');
      continue;
    }
    if (action === 'dryrun') {
      await applySync(true);
      continue;
    }
    if (action === 'sync') {
      await applySync(false);
      continue;
    }
    if (action.startsWith('rowpreview:')) {
      const [, hostname, service] = action.split(':');
      S.selectedHostname = hostname || '';
      S.syncService = service || 'all';
      await fetchPlan(S.syncService, S.selectedHostname);
      continue;
    }
    if (action.startsWith('testconfig:')) {
      const service = action.slice('testconfig:'.length);
      S.configOpen = true;
      S.configTab = service || 'caddy';
      render();
      await testConfig(service);
      continue;
    }
    if (action === 'toggleconfig:closed') {
      S.configOpen = false;
      render();
      continue;
    }
    if (action === 'setconfig:unbound') {
      S.configOpen = true;
      S.configTab = 'unbound';
      S.forms.unbound = {
        ...S.forms.unbound,
        base_url: 'https://saved.example.test',
        api_key: 'saved-key',
        api_secret: '',
      };
      await doSave('unbound');
      continue;
    }
  }

  S.e2eDone = true;
  render();
}

// ── Boot ───────────────────────────────────────────────────────────────────
render();
if (e2eActions().includes('holdloading')) {
  S.loading = true;
  render();
} else {
  refresh().then(runE2EActions);
}
