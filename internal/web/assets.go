package web

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Caddy DNS Sync</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #0f1218;
      --panel: #171b24;
      --panel-2: #1f2530;
      --text: #e7edf6;
      --muted: #8f9bad;
      --border: #303848;
      --accent: #7aa2f7;
      --ok: #9ece6a;
      --warn: #e0af68;
      --bad: #f7768e;
      --radius: 8px;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font: 14px/1.45 ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 18px 24px;
      border-bottom: 1px solid var(--border);
      background: #121620;
    }
    h1 { margin: 0; font-size: 18px; font-weight: 650; letter-spacing: 0; }
    main { padding: 20px 24px 28px; display: grid; gap: 16px; }
    .toolbar, .summary {
      display: flex;
      gap: 10px;
      align-items: center;
      flex-wrap: wrap;
    }
    .panel {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      overflow: hidden;
    }
    button, select {
      border: 1px solid var(--border);
      background: var(--panel-2);
      color: var(--text);
      border-radius: 6px;
      padding: 7px 10px;
      font: inherit;
    }
    button.primary { background: var(--accent); color: #0b1020; border-color: var(--accent); font-weight: 650; }
    button:disabled { opacity: .55; }
    .metric {
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 8px 10px;
      background: var(--panel);
      color: var(--muted);
    }
    .metric strong { color: var(--text); }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 10px 12px; border-bottom: 1px solid var(--border); text-align: left; white-space: nowrap; }
    th { color: var(--muted); font-size: 12px; text-transform: uppercase; letter-spacing: .04em; background: #151923; }
    tbody tr:hover { background: #1b2130; }
    .status { font-weight: 650; }
    .ok { color: var(--ok); }
    .warn { color: var(--warn); }
    .bad { color: var(--bad); }
    .muted { color: var(--muted); }
    .log {
      min-height: 42px;
      padding: 12px;
      color: var(--muted);
      border-top: 1px solid var(--border);
      background: #11151d;
    }
    @media (max-width: 760px) {
      header { align-items: flex-start; flex-direction: column; gap: 10px; }
      main { padding: 14px; }
      .panel { overflow-x: auto; }
    }
  </style>
</head>
<body>
  <header>
    <h1>Caddy DNS Sync</h1>
    <div class="muted" id="config">Loading runtime...</div>
  </header>
  <main>
    <section class="toolbar">
      <button class="primary" id="refresh">Refresh</button>
      <select id="service">
        <option value="all">All targets</option>
        <option value="unbound">Unbound</option>
        <option value="adguard">AdGuard</option>
        <option value="dhcp">DHCP</option>
      </select>
      <button id="plan">Preview sync</button>
      <button id="dryrun">Dry-run apply</button>
      <span class="muted" id="state">Idle</span>
    </section>
    <section class="summary" id="summary"></section>
    <section class="panel">
      <table>
        <thead>
          <tr><th>Hostname</th><th>Source</th><th>Caddy</th><th>DNS</th><th>Unbound</th><th>AdGuard</th><th>Status</th></tr>
        </thead>
        <tbody id="entries"></tbody>
      </table>
      <div class="log" id="log">No actions planned.</div>
    </section>
  </main>
  <script>
    let currentActions = [];
    const $ = id => document.getElementById(id);

    async function getJSON(url) {
      const res = await fetch(url);
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || res.statusText);
      return data;
    }

    function statusClass(status) {
      if (status === 0) return 'ok';
      if (status === 1 || status === 3 || status === 5) return 'warn';
      return 'bad';
    }

    function statusText(status) {
      return ['Synced','Partial','Out of Sync','Caddy Only','Stale','DHCP Mismatch'][status] || 'Unknown';
    }

    function esc(value) {
      return String(value || '').replace(/[&<>"']/g, c => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
      }[c]));
    }

    function renderEntries(entries) {
      $('entries').innerHTML = entries.map(e => '<tr>' +
        '<td>' + esc(e.hostname) + '</td>' +
        '<td class="muted">' + esc(e.data_source || '-') + '</td>' +
        '<td>' + esc(e.caddy_upstream || '-') + '</td>' +
        '<td>' + esc(e.dns_resolved || '-') + '</td>' +
        '<td>' + esc(e.unbound_status && e.unbound_status.ip || '-') + '</td>' +
        '<td>' + esc(e.adguard_status && e.adguard_status.ip || '-') + '</td>' +
        '<td class="status ' + statusClass(e.overall_status) + '">' + esc(e.status_label || statusText(e.overall_status)) + '</td>' +
      '</tr>').join('');
      $('summary').innerHTML = '<span class="metric"><strong>' + entries.length + '</strong> entries</span>';
    }

    async function refresh() {
      $('state').textContent = 'Loading entries...';
      const config = await getJSON('/api/config');
      $('config').textContent = 'Caddy ' + config.caddy.server_ip + ':' + config.caddy.server_port;
      const data = await getJSON('/api/entries');
      renderEntries(data.entries || []);
      $('state').textContent = 'Loaded';
    }

    async function plan() {
      $('state').textContent = 'Planning sync...';
      const service = $('service').value;
      const data = await getJSON('/api/sync/plan?service=' + encodeURIComponent(service));
      currentActions = data.actions || [];
      $('log').textContent = currentActions.length
        ? currentActions.map(a => a.type + ' ' + a.service + ' ' + a.hostname + ' -> ' + (a.new_ip || '')).join('\n')
        : 'No actions needed.';
      $('state').textContent = 'Plan ready';
    }

    async function dryrun() {
      $('state').textContent = 'Applying dry run...';
      const res = await fetch('/api/sync/apply', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({dry_run: true, actions: currentActions})
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || res.statusText);
      $('log').textContent = data.result.message + ' added=' + data.result.items_added + ' updated=' + data.result.items_updated + ' deleted=' + data.result.items_deleted;
      $('state').textContent = 'Dry run complete';
    }

    $('refresh').onclick = () => refresh().catch(err => $('state').textContent = err.message);
    $('plan').onclick = () => plan().catch(err => $('state').textContent = err.message);
    $('dryrun').onclick = () => dryrun().catch(err => $('state').textContent = err.message);
    refresh().catch(err => $('state').textContent = err.message);
  </script>
</body>
</html>`
