import '../styles/Topbar.css';
import {
  RefreshCw,
  Settings,
  Zap
} from 'lucide-react';
import type { ConfigResponse } from '../types';
import { LoadingSpinner } from './LoadingSpinner';
import { TabBar, type TabId } from './TabBar';

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


