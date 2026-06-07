// Caddy DNS Sync — Vanilla JS dashboard
// Replaces the React bundle. No build step required.
'use strict';

const wc = () => window.UNBOUNDCLI_WEB_CONFIG || { applyToken: '', mutationEnabled: false };

// ── State ──────────────────────────────────────────────────────────────────
const S = {
  config: null, entries: [], report: {},
  loading: true, message: '', msgKind: 'info',
  search: '', statusFilter: 'all', serviceFilter: 'all',
  selectedHostname: '',
  syncService: 'all', syncLoading: false, syncLog: '',
  syncProgress: { title: '', detail: '' },
  plannedActions: [], planId: '', actionIds: [], canSyncNow: false,
  e2eDone: false,
  configOpen: false, configTab: 'caddy', configStatus: '', configStatusKind: '',
  cfDiscover: { loading: false, verifyOk: false, verifyMsg: '', accounts: [], tunnels: [], zones: [] },
  testResults: {},
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
    if (st === 'synced'      && e.overall_status !== 0) return false;
    if (st === 'out_of_sync' && e.overall_status <= 1) return false;
    if (st === 'caddy_only'  && e.overall_status !== 3) return false;
    if (st === 'stale'       && e.overall_status !== 4) return false;
    if (st === 'cloudflare'  && !e.cloudflare_status?.configured) return false;
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
function cfBadge(cf) {
  if (!cf?.configured) return '<span class="service-badge missing" data-label="Cloudflare route"><strong>CF</strong><span>Not routed</span></span>';
  const label = cf.tunnel_name || 'CF';
  const details = [cf.service ? `Service: ${cf.service}` : '', cf.http_host_header ? `Host header: ${cf.http_host_header}` : ''].filter(Boolean).join(' · ');
  return `<span class="service-badge cf" data-label="Cloudflare route" title="${esc(details)}"><strong>CF</strong><span>${esc(label)}</span></span>`;
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
      ${pills.map(p => `<button class="nav-item service svc-pill ${p.on?'on':'off'}${S.serviceFilter===p.key?' pill-active':''}"
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
  const sf = S.statusFilter;
  function mcard(tone, status, label, val, sub) {
    const active = sf === status ? ' metric-active' : '';
    const act = status === 'all' ? 'data-action="filter-status" data-status="all"' : `data-action="filter-status" data-status="${status}"`;
    return `<article class="metric-card ${tone}${active}" ${act} role="button" tabindex="0" title="Filter: ${label}"><div><span>${label}</span><strong>${val}</strong><small>${sub}</small></div></article>`;
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

  const rows = entries.map(e => {
    const sel = e.hostname === S.selectedHostname;
    const ub  = e.unbound_status, ag = e.adguard_status, cf = e.cloudflare_status;
    return `<tr class="${sel?'selected-row':''}" data-action="select-row" data-hostname="${esc(e.hostname)}" tabindex="0" aria-selected="${sel ? 'true' : 'false'}">
      <td data-label="Hostname"><strong>${esc(e.hostname)}</strong><span class="subtle">${esc(e.data_source||'Caddy route')}</span></td>
      <td data-label="Status"><span class="status-chip ${statusCls(e.overall_status)}">${esc(e.status_label||'Unknown')}</span></td>
      <td data-label="Services">
        <div class="service-stack">
          <span class="service-badge ${svcTone(ub)}"><strong>UB</strong>${esc(svcText(ub))}</span>
          <span class="service-badge ${svcTone(ag)}"><strong>AG</strong>${esc(svcText(ag))}</span>
          ${cfBadge(cf)}
        </div>
      </td>
      <td data-label="Upstream"><span>${esc(e.caddy_upstream||'—')}</span><span class="subtle">${esc(e.caddy_ip||'')}</span></td>
      <td data-label="DNS"><span class="dns-result ${dnsCls(e.dns_resolved)}">${esc(e.dns_resolved||'FAIL')}</span></td>
      <td data-label="Actions">
        <div class="row-actions">
          <button class="row-preview" data-action="row-preview" data-hostname="${esc(e.hostname)}">Preview</button>
          <button class="row-sync" data-action="row-sync" data-hostname="${esc(e.hostname)}"${wc().mutationEnabled?'':' disabled'}>Sync</button>
        </div>
      </td>
    </tr>`;
  }).join('');

  return `<section id="entries-panel" class="panel entries-panel" data-table-scrolls="false">
    <table>
      <thead><tr><th>Hostname</th><th>Status</th><th>Services</th><th>Upstream</th><th>DNS</th><th>Actions</th></tr></thead>
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

function tInspector() {
  const e = selectedEntry();
  if (!e) return `<section id="host-inspector" class="panel inspector" aria-live="polite">
    <div class="panel-title"><strong>Selected host</strong><span>Click a row to inspect.</span></div>
    <div class="empty-state">${ICON.wifi_off} No hostname selected.</div>
  </section>`;

  const ub = e.unbound_status, ag = e.adguard_status, cf = e.cloudflare_status;
  return `<section id="host-inspector" class="panel inspector" aria-live="polite">
    <div class="host-title">
      <strong>${esc(e.hostname)}</strong>
      <div>
        <span class="status-chip ${statusCls(e.overall_status)}">${esc(e.status_label)}</span>
        <span class="dns-result ${dnsCls(e.dns_resolved)}">${esc(e.dns_resolved||'FAIL')}</span>
      </div>
    </div>
    <div class="inspector-grid">
      <div class="inspector-line"><span>Upstream</span><strong>${esc(e.caddy_upstream||'—')}</strong></div>
      <div class="inspector-line"><span>Source</span><strong>${esc(e.data_source||'—')}</strong></div>
      <div class="inspector-line ${svcTone(ub)}"><span>Unbound</span><strong>${esc(svcText(ub))}</strong></div>
      <div class="inspector-line ${svcTone(ag)}"><span>AdGuard</span><strong>${esc(svcText(ag))}</strong></div>
      <div class="inspector-line ${cf?.configured?'violet':''}"><span>Cloudflare</span><strong>${cf?.configured?esc(cf.service||'Routed'):'Not routed'}</strong></div>
    </div>
    <div class="inspector-actions">
      <button id="inspector-preview" data-action="inspector-preview">Preview</button>
      <button id="inspector-sync" data-action="inspector-sync"${wc().mutationEnabled?'':' disabled'}>Sync</button>
    </div>
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
        <aside class="right-rail">
          ${tSyncPanel()}
          ${tInspector()}
        </aside>
      </section>
    </main>
    ${tConfigModal()}
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
  if (a === 'filter-status')  { S.statusFilter = el.dataset.status || 'all'; render(); return; }
  if (a === 'cfg-tab')        { S.configTab = el.dataset.tab; render(); return; }

  if (a === 'select-row') {
    const tr = el.closest('tr');
    const h  = tr?.dataset.hostname ?? el.dataset.hostname;
    if (h) { S.selectedHostname = h; render(); }
    return;
  }
  if (a === 'row-preview') {
    ev.stopPropagation();
    S.selectedHostname = el.dataset.hostname;
    await fetchPlan(S.syncService, el.dataset.hostname);
    return;
  }
  if (a === 'row-sync') {
    ev.stopPropagation();
    S.selectedHostname = el.dataset.hostname;
    if (await fetchPlan(S.syncService, el.dataset.hostname)) await applySync(false);
    return;
  }
  if (a === 'preview-sync')       { await fetchPlan(S.syncService, S.selectedHostname); return; }
  if (a === 'dry-run')            { await applySync(true);  return; }
  if (a === 'sync-now')           { await applySync(false); return; }
  if (a === 'inspector-preview')  { await fetchPlan(S.syncService, S.selectedHostname); return; }
  if (a === 'inspector-sync')     { if (await fetchPlan(S.syncService, S.selectedHostname)) await applySync(false); return; }
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

document.addEventListener('change', ev => {
  if (ev.target.id === 'status-filter') { S.statusFilter = ev.target.value; render(); return; }
  if (ev.target.id === 'sync-service')  { S.syncService  = ev.target.value; return; }
  const { form, field } = ev.target.dataset;
  if (form && field && S.forms[form]) {
    S.forms[form] = { ...S.forms[form], [field]: ev.target.dataset.type === 'checkbox' ? ev.target.checked : ev.target.value };
  }
});

document.addEventListener('keydown', ev => {
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
