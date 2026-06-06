let entries = [];
let plannedActions = [];
let plannedService = '';
let plannedHostname = '';
let enabledServices = {};
let serviceReport = {};

const el = (id) => document.getElementById(id);

async function getJSON(path) {
  const response = await fetch(path);
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error || response.statusText);
  }
  return data;
}

function escapeHTML(value) {
  return String(value || '').replace(/[&<>"']/g, (char) => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;'
  }[char]));
}

function statusClass(label) {
  const normalized = String(label || '').toLowerCase();
  if (normalized.includes('stale') || normalized.includes('out')) return 'bad';
  if (normalized.includes('sync')) return 'ok';
  return 'warn';
}

function statusClassByCode(status) {
  if (status === 0) return 'ok';
  if (status === 2 || status === 4 || status === 5) return 'bad';
  return 'warn';
}

function matchesStatus(entry, filter) {
  if (filter === 'all') return true;
  const status = entry.overall_status;
  if (filter === 'out_of_sync') return status === 2;
  if (filter === 'caddy_only') return status === 3;
  if (filter === 'stale') return status === 4;
  return true;
}

function matchesService(entry, filter) {
  if (filter === 'all') return true;
  if (filter === 'unbound') return Boolean(entry.unbound_status?.configured);
  if (filter === 'adguard') return Boolean(entry.adguard_status?.configured);
  if (filter === 'cloudflare') return Boolean(entry.cloudflare_status?.configured);
  return true;
}

function filteredEntries() {
  const status = el('status-filter').value;
  const service = el('service-filter').value;
  const query = el('search').value.trim().toLowerCase();
  return entries.filter((entry) =>
    matchesStatus(entry, status) &&
    matchesService(entry, service) &&
    (!query || String(entry.hostname || '').toLowerCase().includes(query))
  );
}

function renderSummary() {
  const visible = filteredEntries();
  const out = visible.filter((entry) => entry.overall_status === 2).length;
  const caddyOnly = visible.filter((entry) => entry.overall_status === 3).length;
  const stale = visible.filter((entry) => entry.overall_status === 4).length;
  const cloudflare = visible.filter((entry) => entry.cloudflare_status?.configured).length;
  el('summary').innerHTML = [
    `<span class="metric neutral"><strong>${visible.length}</strong> entries</span>`,
    `<span class="metric bad"><strong>${out}</strong> out of sync</span>`,
    `<span class="metric warn"><strong>${caddyOnly}</strong> caddy only</span>`,
    `<span class="metric bad"><strong>${stale}</strong> stale</span>`,
    `<span class="metric cf"><strong>${cloudflare}</strong> cloudflare</span>`
  ].join('');
}

function renderEntries() {
  const visible = filteredEntries();
  el('entries').innerHTML = visible.map((entry) => `
    <tr data-hostname="${escapeHTML(entry.hostname)}">
      <td>
        <strong>${escapeHTML(entry.hostname)}</strong>
        <span class="subtle">${escapeHTML(entry.data_source || '-')}</span>
      </td>
      <td><span class="status-chip ${statusClassByCode(entry.overall_status)}">${escapeHTML(entry.status_label || 'Unknown')}</span></td>
      <td>${serviceBadges(entry)}</td>
      <td>
        <span>${escapeHTML(entry.caddy_upstream || '-')}</span>
        <span class="subtle">admin ${escapeHTML(entry.caddy_ip || '-')}</span>
      </td>
      <td><span class="dns-result ${dnsResultClass(entry.dns_resolved)}">${escapeHTML(entry.dns_resolved || 'FAIL')}</span></td>
      <td>${cloudflareDetails(entry.cloudflare_status)}</td>
      <td>
        <div class="row-actions">
          <select class="row-sync-service" aria-label="Sync target for ${escapeHTML(entry.hostname)}">
            <option value="all">Available DNS</option>
            <option value="unbound">Unbound</option>
            <option value="adguard">AdGuard</option>
            <option value="dhcp">DHCP preview</option>
          </select>
          <button class="row-preview" type="button" data-hostname="${escapeHTML(entry.hostname)}">Preview</button>
        </div>
      </td>
    </tr>
  `).join('');
  document.querySelectorAll('.row-preview').forEach((button) => {
    button.addEventListener('click', () => {
      const row = button.closest('tr');
      const service = row.querySelector('.row-sync-service').value;
      previewSync(service, button.dataset.hostname).catch((err) => {
        el('sync-log').textContent = err.message;
      });
    });
  });
  markResponsiveState();
}

function dnsResultClass(value) {
  const normalized = String(value || '').toLowerCase();
  return normalized && normalized !== 'fail' ? 'ok' : 'bad';
}

function serviceBadges(entry) {
  return [
    serviceBadge('Unbound', entry.unbound_status?.configured, entry.unbound_status?.in_sync, entry.unbound_status?.ip),
    serviceBadge('AdGuard', entry.adguard_status?.configured, entry.adguard_status?.in_sync, entry.adguard_status?.ip),
    serviceBadge('DHCP', entry.dhcp_status?.configured, entry.dhcp_status?.in_sync, entry.dhcp_status?.ip)
  ].join('');
}

function serviceBadge(name, configured, inSync, value) {
  let tone = 'missing';
  let label = 'Missing';
  if (configured && inSync) {
    tone = 'ok';
    label = value || 'In sync';
  } else if (configured) {
    tone = 'bad';
    label = value || 'Mismatch';
  }
  return `<span class="service-badge ${tone}"><strong>${name}</strong>${escapeHTML(label)}</span>`;
}

function cloudflareDetails(status) {
  if (!status?.configured) {
    return '<span class="cloudflare-detail missing">Not routed</span>';
  }
  const header = status.http_host_header ? `Host header ${escapeHTML(status.http_host_header)}` : 'Missing HTTPHostHeader';
  const tone = status.http_host_header ? 'ok' : 'bad';
  return `
    <span class="cloudflare-detail ${tone}">
      <strong>${escapeHTML(status.tunnel_name || 'Tunnel')}</strong>
      <span>${escapeHTML(status.service || '-')}</span>
      <span>${header}</span>
      <span>${status.has_access_policy ? 'Access policy' : 'No access policy'}</span>
    </span>
  `;
}

function renderServiceHealth() {
  const services = [
    {key: 'caddy', label: 'Caddy', enabled: enabledServices.caddy !== false},
    {key: 'unbound', label: 'Unbound', enabled: enabledServices.unbound !== false},
    {key: 'adguard', label: 'AdGuard', enabled: enabledServices.adguard !== false},
    {key: 'dhcp', label: 'DHCP', enabled: enabledServices.dhcp !== false},
    {key: 'cloudflare', label: 'Cloudflare', enabled: enabledServices.cloudflare !== false}
  ];
  el('service-health').innerHTML = services.map((service) => {
    const report = serviceReport[service.key] || {};
    const count = typeof report.count === 'number' ? `${report.count} loaded` : service.enabled ? 'configured' : 'not configured';
    const tone = service.enabled ? 'ok' : 'missing';
    return `<div class="service-card ${tone}"><span>${service.label}</span><strong>${count}</strong></div>`;
  }).join('');
}

function applyEnabledServices(config) {
  enabledServices = config.enabled || {};
  document.querySelectorAll('[data-service]').forEach((node) => {
    const service = node.getAttribute('data-service');
    const available = enabledServices[service] !== false;
    node.disabled = !available;
    node.classList.toggle('disabled-service', !available);
  });
  const allOption = el('sync-service-all');
  const unboundAvailable = enabledServices.unbound !== false;
  const adguardAvailable = enabledServices.adguard !== false;
  allOption.disabled = !unboundAvailable && !adguardAvailable;
  allOption.textContent = unboundAvailable && adguardAvailable ? 'Unbound + AdGuard' : 'Available DNS targets';
  el('app').setAttribute('data-unbound-enabled', String(unboundAvailable));
  el('app').setAttribute('data-adguard-enabled', String(adguardAvailable));
  renderServiceHealth();
  if (el('sync-service').selectedOptions[0]?.disabled) {
    el('sync-service').value = unboundAvailable ? 'unbound' : adguardAvailable ? 'adguard' : 'dhcp';
    clearPlannedActions();
  }
}

function setMessage(text, kind = 'info') {
  const node = el('message');
  node.textContent = text;
  node.className = `message ${kind}`;
}

function setLoading(isLoading) {
  el('refresh').disabled = isLoading;
  el('preview-sync').disabled = isLoading;
  el('app').setAttribute('data-loading', String(isLoading));
  el('top-progress').hidden = !isLoading;
  if (isLoading) setMessage('Loading service status...', 'info');
}

function setPreviewLoading(isLoading) {
  el('app').setAttribute('data-preview-loading', String(isLoading));
  el('preview-sync').disabled = isLoading;
  el('dry-run-sync').disabled = isLoading || !plannedActions.length;
  el('sync-progress').hidden = !isLoading;
}

function setDryRunEnabled(enabled) {
  el('dry-run-sync').disabled = !enabled;
  el('app').setAttribute('data-dry-run-enabled', String(enabled));
}

function clearPlannedActions() {
  plannedActions = [];
  plannedService = '';
  plannedHostname = '';
  setDryRunEnabled(false);
}

function renderActions(actions, title = 'Planned actions') {
  if (!actions.length) {
    el('sync-log').textContent = 'No actions needed.';
    setDryRunEnabled(false);
    return;
  }
  const lines = actions.map((action) => {
    const target = action.new_ip ? ` -> ${action.new_ip}` : '';
    return `${action.type.toUpperCase()} ${action.service} ${action.hostname}${target}`;
  });
  el('sync-log').textContent = `${title}\n${lines.join('\n')}`;
  setDryRunEnabled(true);
}

async function previewSync(service = el('sync-service').value, hostname = '') {
  if (service === 'dhcp') {
    clearPlannedActions();
    el('sync-log').textContent = 'DHCP apply is not implemented; preview only.';
    return;
  }
  setPreviewLoading(true);
  el('sync-log').textContent = hostname ? `Planning ${service} sync for ${hostname}...` : `Planning ${service} sync...`;
  try {
    const data = await getJSON(`/api/sync/plan?service=${encodeURIComponent(service)}`);
    plannedActions = (data.actions || []).filter((action) => !hostname || action.hostname === hostname);
    plannedService = service;
    plannedHostname = hostname;
    renderActions(plannedActions, hostname ? `Planned actions for ${hostname}` : 'Planned actions');
  } finally {
    setPreviewLoading(false);
  }
}

async function dryRunSync() {
  if (el('sync-service').value === 'dhcp') {
    el('sync-log').textContent = 'DHCP apply is not implemented.';
    return;
  }
  if (!plannedActions.length) {
    el('sync-log').textContent = 'Preview sync before running a dry run.';
    return;
  }
  if (!plannedHostname && plannedService !== el('sync-service').value) {
    clearPlannedActions();
    el('sync-log').textContent = 'Preview sync again for the selected target.';
    return;
  }
  const response = await fetch('/api/sync/apply', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({dry_run: true, actions: plannedActions})
  });
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error || response.statusText);
  }
  const result = data.result;
  el('sync-log').textContent = `${result.message}\nadded=${result.items_added} updated=${result.items_updated} deleted=${result.items_deleted}`;
}

function markResponsiveState() {
  const app = el('app');
  app.setAttribute('data-mobile', window.innerWidth <= 760 ? 'true' : 'false');
  const panel = el('entries-panel');
  app.setAttribute('data-table-scrolls', panel.scrollWidth > panel.clientWidth ? 'true' : 'false');
}

async function runE2EActions() {
  const params = new URLSearchParams(window.location.search);
  const script = params.get('e2e');
  if (!script || window.UNBOUNDCLI_TEST_HOOKS !== true) return;
  for (const action of script.split(',')) {
    const [name, ...parts] = action.split(':');
    const value = parts.join(':');
    if (name === 'filter') {
      el('status-filter').value = value;
      el('status-filter').dispatchEvent(new Event('input', {bubbles: true}));
    }
    if (name === 'search') {
      el('search').value = value;
      el('search').dispatchEvent(new Event('input', {bubbles: true}));
    }
    if (name === 'preview') {
      el('sync-service').value = value;
      await previewSync();
    }
    if (name === 'rowpreview') {
      const [hostname, service = 'unbound'] = parts;
      await previewSync(service, hostname);
    }
    if (name === 'dryrun') {
      await dryRunSync();
    }
  }
  el('app').setAttribute('data-e2e', 'done');
}

async function refreshEntries() {
  setLoading(true);
  try {
    const config = await getJSON('/api/config');
    el('runtime').textContent = `Caddy ${config.caddy.server_ip}:${config.caddy.server_port}`;
    applyEnabledServices(config);
    const data = await getJSON('/api/entries');
    entries = data.entries || [];
    serviceReport = data.report?.services || {};
    renderSummary();
    renderServiceHealth();
    renderEntries();
    setMessage(entries.length ? 'Loaded service status.' : 'No entries found.', 'info');
    await runE2EActions();
  } catch (err) {
    el('runtime').textContent = err.message;
    setMessage(err.message, 'error');
  } finally {
    setLoading(false);
  }
}

window.addEventListener('DOMContentLoaded', () => {
  el('refresh').addEventListener('click', () => refreshEntries());
  el('preview-sync').addEventListener('click', () => previewSync().catch((err) => {
    el('sync-log').textContent = err.message;
  }));
  el('dry-run-sync').addEventListener('click', () => dryRunSync().catch((err) => {
    el('sync-log').textContent = err.message;
  }));
  ['status-filter', 'service-filter', 'search'].forEach((id) => {
    el(id).addEventListener('input', () => {
      renderSummary();
      renderEntries();
    });
  });
  el('sync-service').addEventListener('input', () => {
    clearPlannedActions();
    el('sync-log').textContent = 'No actions planned.';
  });
  window.addEventListener('resize', markResponsiveState);
  setDryRunEnabled(false);
  refreshEntries();
});
