import '../styles/Topbar.css';
import '../styles/Sidebar.css';
import {
  FileCode2,
  Gauge,
  RefreshCw,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Stethoscope,
  TerminalSquare,
  Zap
} from 'lucide-react';
import {
  serviceMeta,
  serviceOrder
} from '../lib/services';
import type { ConfigResponse, EntriesResponse, ServiceKey } from '../types';
import { LoadingSpinner } from './LoadingSpinner';
import { TabBar } from './TabBar';
import type { TabId } from './TabBar';

export function Topbar({ config, loading, syncLoading, view, setView, onRefresh, onOpenConfig }: { config: ConfigResponse | null; loading: boolean; syncLoading: boolean; view: TabId; setView: (v: TabId) => void; onRefresh: () => void; onOpenConfig: () => void }) {
  const busy = loading || syncLoading;
  const busyLabel = syncLoading ? 'Syncing...' : 'Loading...';

  return (
    <header className="topbar">
      <div className="brand-lockup">
        <div className="brand-mark"><Zap size={18} /></div>
        <div>
          <h1>Caddy DNS Sync</h1>
          <span>{config?.version ? <span className="brand-version">{config.version}</span> : 'Local dashboard'}</span>
        </div>
      </div>
      <TabBar active={view} onChange={setView} />
      {busy && (
        <div className="topbar-activity" role="status" aria-live="polite">
          <LoadingSpinner size={14} />
          <span>{busyLabel}</span>
        </div>
      )}
      <div className="runtime-card" id="runtime">
        <span>Caddy runtime</span>
        <strong>{config ? `${config.caddy.server_ip}:${config.caddy.server_port}` : 'Loading...'}</strong>
        <em className={config?.enabled?.caddy === false ? 'down' : ''}>{config?.enabled?.caddy === false ? 'Offline' : 'Running'}</em>
      </div>
      <div className="top-actions">
        <button id="refresh" type="button" onClick={onRefresh} disabled={loading}>
          {loading ? <LoadingSpinner size={16} /> : <RefreshCw size={16} />} Refresh
        </button>
        <button id="config-toggle" type="button" onClick={onOpenConfig}>
          <Settings size={16} /> Settings
        </button>
      </div>
    </header>
  );
}

export function Sidebar({ config, report, view, setView, onOpenConfig }: { config: ConfigResponse | null; report: EntriesResponse['report']; view: TabId; setView: (v: TabId) => void; onOpenConfig: () => void }) {
  return (
    <aside className="sidebar">
      <div className="brand-lockup">
        <div className="brand-mark"><Zap size={18} /></div>
        <div>
          <h1>Caddy DNS Sync</h1>
          <span>{config?.version ? <span className="brand-version">{config.version}</span> : 'Local dashboard'}</span>
        </div>
      </div>
      <nav className="nav-stack" aria-label="Primary">
        <span className="nav-section">Overview</span>
        <button type="button" className={`nav-item ${view === 'dashboard' ? 'active' : ''}`} onClick={() => setView('dashboard')}><Gauge size={15} /> Dashboard</button>
        <button type="button" className={`nav-item ${view === 'caddyfile' ? 'active' : ''}`} onClick={() => setView('caddyfile')}><FileCode2 size={15} /> Caddyfile</button>
        <button type="button" className={`nav-item ${view === 'auth' ? 'active' : ''}`} onClick={() => setView('auth')}><ShieldCheck size={15} /> Auth Flows</button>
        <button type="button" className={`nav-item ${view === 'diagnostics' ? 'active' : ''}`} onClick={() => setView('diagnostics')}><Stethoscope size={15} /> Diagnostics</button>
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
