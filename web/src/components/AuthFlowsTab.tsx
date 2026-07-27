import { AlertTriangle, CheckCircle2, Loader2, RefreshCw, ShieldAlert, ShieldCheck, ShieldX } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type { AuthInventoryResponse, AuthStatus, HostAuth } from '../types';

const WAN_AUTH_LABELS: Record<string, string> = {
  none: 'None',
  cf_access: 'CF Access',
  forward_auth: 'Forward Auth',
  app_native: 'App-Native',
};

const LAN_AUTH_LABELS: Record<string, string> = {
  none: 'None',
  forward_auth: 'Forward Auth',
  app_native: 'App-Native',
};

const API_AUTH_LABELS: Record<string, string> = {
  none: 'None',
  cf_service_token: 'CF Service Token',
  authentik_bearer: 'Authentik Bearer',
  app_native_key: 'App-Native Key',
};

function StatusIcon({ status }: { status: AuthStatus }) {
  switch (status) {
    case 'ok':
      return <ShieldCheck size={16} className="auth-status-icon ok" />;
    case 'warning':
      return <AlertTriangle size={16} className="auth-status-icon warning" />;
    case 'error':
      return <ShieldX size={16} className="auth-status-icon error" />;
    default:
      return <ShieldAlert size={16} className="auth-status-icon unknown" />;
  }
}

function AuthBadge({ value, labels }: { value: string; labels: Record<string, string> }) {
  const text = labels[value] ?? value;
  const cls = value === 'none' ? 'auth-badge none' : `auth-badge ${value.replace(/_/g, '-')}`;
  return <span className={cls}>{text}</span>;
}

function HostRow({ host }: { host: HostAuth }) {
  const [expanded, setExpanded] = useState(false);
  const hasDetails = Boolean(
    host.cf_access_app_id ||
    host.authentik_provider_pk ||
    (host.notes && host.notes.length > 0)
  );

  return (
    <>
      <tr className="auth-row" onClick={() => hasDetails && setExpanded(e => !e)}>
        <td className="auth-hostname">
          {hasDetails && <span className="auth-expand">{expanded ? '▾' : '▸'}</span>}
          {host.hostname}
        </td>
        <td className="auth-cell">
          {host.wan_exposed ? <AuthBadge value={host.wan_auth} labels={WAN_AUTH_LABELS} /> : <span className="auth-na">N/A</span>}
        </td>
        <td className="auth-cell">
          <AuthBadge value={host.lan_auth} labels={LAN_AUTH_LABELS} />
        </td>
        <td className="auth-cell">
          <AuthBadge value={host.api_auth} labels={API_AUTH_LABELS} />
        </td>
        <td className="auth-cell">
          <span className={`auth-indicator ${host.status}`}>
            <StatusIcon status={host.status} />
          </span>
        </td>
      </tr>
      {expanded && hasDetails && (
        <tr className="auth-detail-row">
          <td colSpan={5}>
            <div className="auth-detail">
              {host.cf_access_app_id && (
                <div className="auth-detail-section">
                  <strong>Cloudflare Access</strong>
                  <dl>
                    <dt>App ID</dt><dd>{host.cf_access_app_id}</dd>
                    <dt>Domain</dt><dd>{host.cf_access_app_domain}</dd>
                    {host.cf_access_app_type && <><dt>Type</dt><dd>{host.cf_access_app_type}</dd></>}
                    {host.cf_access_decisions && host.cf_access_decisions.length > 0 && (
                      <><dt>Policy Decisions</dt><dd>{host.cf_access_decisions.join(', ')}</dd></>
                    )}
                  </dl>
                </div>
              )}
              {host.authentik_provider_pk ? (
                <div className="auth-detail-section">
                  <strong>Authentik</strong>
                  <dl>
                    <dt>Provider PK</dt><dd>{host.authentik_provider_pk}</dd>
                    {host.authentik_app_slug && <><dt>App Slug</dt><dd>{host.authentik_app_slug}</dd></>}
                    {host.authentik_provider_mode && <><dt>Mode</dt><dd>{host.authentik_provider_mode}</dd></>}
                    {host.authentik_outpost_uuid && <><dt>Outpost</dt><dd>{host.authentik_outpost_uuid}</dd></>}
                  </dl>
                </div>
              ) : null}
              {host.notes && host.notes.length > 0 && (
                <div className="auth-detail-section">
                  <strong>Notes</strong>
                  <ul>
                    {host.notes.map((n, i) => <li key={i}>{n}</li>)}
                  </ul>
                </div>
              )}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

export function AuthFlowsTab() {
  const [data, setData] = useState<AuthInventoryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await api.authInventory();
      setData(result);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const hosts = data?.hosts ?? [];
  const okCount = hosts.filter(h => h.status === 'ok').length;
  const warnCount = hosts.filter(h => h.status === 'warning').length;
  const errCount = hosts.filter(h => h.status === 'error').length;
  const wanExposedCount = hosts.filter(h => h.wan_exposed).length;

  return (
    <main className="dashboard-shell auth-shell">
      <div className="auth-header">
        <div className="auth-header-title">
          <ShieldCheck size={20} />
          <h2>Auth Flows</h2>
        </div>
        <button type="button" className="btn-sm" onClick={() => void load()} disabled={loading}>
          {loading ? <Loader2 size={14} className="spin" /> : <RefreshCw size={14} />} Refresh
        </button>
      </div>

      {data && (
        <div className="auth-sources">
          <span className={`auth-source-badge ${data.sources.cloudflare_access ? 'ok' : 'off'}`}>
            CF Access: {data.sources.cloudflare_access ? 'Connected' : 'Not configured'}
          </span>
          <span className={`auth-source-badge ${data.sources.authentik ? 'ok' : 'off'}`}>
            Authentik: {data.sources.authentik ? 'Connected' : 'Not configured'}
          </span>
        </div>
      )}

      <div className="auth-metrics">
        <div className="auth-metric">
          <CheckCircle2 size={16} className="ok" />
          <span>{okCount} OK</span>
        </div>
        <div className="auth-metric">
          <AlertTriangle size={16} className="warning" />
          <span>{warnCount} Warnings</span>
        </div>
        <div className="auth-metric">
          <ShieldX size={16} className="error" />
          <span>{errCount} Errors</span>
        </div>
        <div className="auth-metric">
          <span className="auth-metric-label">WAN-exposed:</span>
          <span>{wanExposedCount} hosts</span>
        </div>
      </div>

      {error && (
        <div className="auth-error">
          <ShieldX size={16} />
          <span>Failed to load auth inventory: {error}</span>
        </div>
      )}

      {loading && !data ? (
        <div className="auth-loading">
          <Loader2 size={24} className="spin" />
          <span>Loading auth inventory...</span>
        </div>
      ) : (
        <div className="auth-table-wrap" id="entries-panel">
          <table className="auth-table">
            <thead>
              <tr>
                <th>Hostname</th>
                <th>WAN Auth</th>
                <th>LAN Auth</th>
                <th>API Auth</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {hosts.length === 0 ? (
                <tr>
                  <td colSpan={5} className="auth-empty">No hosts discovered</td>
                </tr>
              ) : (
                hosts.map(host => <HostRow key={host.hostname} host={host} />)
              )}
            </tbody>
          </table>
        </div>
      )}
    </main>
  );
}
