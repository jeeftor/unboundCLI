let entries = [];
let plannedActions = [];
let plannedService = '';
let enabledServices = {};

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
  el('summary').innerHTML = [
    `<span class="metric"><strong>${visible.length}</strong> entries</span>`,
    `<span class="metric"><strong>${out}</strong> out of sync</span>`
  ].join('');
}

function renderEntries() {
  const visible = filteredEntries();
  el('entries').innerHTML = visible.map((entry) => `
    <tr>
      <td>${escapeHTML(entry.hostname)}</td>
      <td>${escapeHTML(entry.data_source || '-')}</td>
      <td>${escapeHTML(entry.caddy_upstream || '-')}</td>
      <td>${escapeHTML(entry.dns_resolved || '-')}</td>
      <td>${escapeHTML(entry.unbound_status?.ip || '-')}</td>
      <td>${escapeHTML(entry.adguard_status?.ip || '-')}</td>
      <td>${entry.cloudflare_status?.configured ? 'Configured' : '-'}</td>
      <td><span class="status-chip ${statusClass(entry.status_label)}">${escapeHTML(entry.status_label || 'Unknown')}</span></td>
    </tr>
  `).join('');
  markResponsiveState();
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
  if (isLoading) setMessage('Loading service status...', 'info');
}

function setDryRunEnabled(enabled) {
  el('dry-run-sync').disabled = !enabled;
  el('app').setAttribute('data-dry-run-enabled', String(enabled));
}

function clearPlannedActions() {
  plannedActions = [];
  plannedService = '';
  setDryRunEnabled(false);
}

function renderActions(actions) {
  if (!actions.length) {
    el('sync-log').textContent = 'No actions needed.';
    setDryRunEnabled(false);
    return;
  }
  const lines = actions.map((action) => {
    const target = action.new_ip ? ` -> ${action.new_ip}` : '';
    return `${action.type.toUpperCase()} ${action.service} ${action.hostname}${target}`;
  });
  el('sync-log').textContent = lines.join('\n');
  setDryRunEnabled(true);
}

async function previewSync() {
  const service = el('sync-service').value;
  if (service === 'dhcp') {
    clearPlannedActions();
    el('sync-log').textContent = 'DHCP apply is not implemented; preview only.';
    return;
  }
  const data = await getJSON(`/api/sync/plan?service=${encodeURIComponent(service)}`);
  plannedActions = data.actions || [];
  plannedService = service;
  renderActions(plannedActions);
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
  if (plannedService !== el('sync-service').value) {
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
    const [name, value] = action.split(':');
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
    renderSummary();
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
