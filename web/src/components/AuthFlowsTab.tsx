import {
  AlertTriangle,
  ArrowLeftRight,
  CheckCircle2,
  Cloud,
  Fingerprint,
  Globe,
  HelpCircle,
  KeyRound,
  Loader2,
  LockKeyhole,
  Monitor,
  Network,
  RefreshCw,
  Route,
  Server,
  ShieldAlert,
  ShieldCheck,
  ShieldQuestion,
  ShieldX,
  Smartphone,
  Terminal,
  Ticket,
  Unlock,
  UnlockKeyhole,
  Wifi,
} from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import type { ComponentType } from 'react';
import type {
  AuthStatus,
  HostAuth,
} from '../types';

// ─── Auth type metadata ─────────────────────────────────────────────────────
// Each auth type has a UNIQUE icon — no two types share the same icon.
// This makes every badge instantly recognizable in both the legend and table.

type AuthMeta = { label: string; desc: string; icon: ComponentType<{ size?: number }>; tone: string };

const WAN_AUTH_INFO: Record<string, AuthMeta> = {
  none: {
    label: 'None',
    desc: 'No WAN authentication. If WAN-exposed, this is a security risk — the host is reachable from the internet without any auth barrier.',
    icon: Unlock,
    tone: 'danger',
  },
  cf_access: {
    label: 'CF Access',
    desc: 'Cloudflare Access sits at the edge and requires IdP login before traffic reaches the tunnel. This is the standard edge-auth pattern.',
    icon: Cloud,
    tone: 'orange',
  },
  forward_auth: {
    label: 'Forward Auth',
    desc: "Caddy's forward_auth directive delegates authentication to Authentik. CF Access (if present) must have a bypass policy to avoid double-login.",
    icon: ArrowLeftRight,
    tone: 'blue',
  },
  app_native: {
    label: 'App-Native',
    desc: 'The application handles its own authentication (e.g., Jellyfin, Audiobookshelf have built-in login). No external auth layer is enforced.',
    icon: LockKeyhole,
    tone: 'green',
  },
};

const LAN_AUTH_INFO: Record<string, AuthMeta> = {
  none: {
    label: 'None',
    desc: 'No LAN authentication. The app is directly accessible on the LAN. This is normal for apps with their own login (app-native).',
    icon: Unlock,
    tone: 'neutral',
  },
  forward_auth: {
    label: 'Forward Auth',
    desc: "Caddy's forward_auth delegates to Authentik even on LAN requests. Users must authenticate via Authentik before reaching the app.",
    icon: Route,
    tone: 'blue',
  },
  app_native: {
    label: 'App-Native',
    desc: 'The application handles its own authentication on LAN traffic.',
    icon: Monitor,
    tone: 'green',
  },
};

const API_AUTH_INFO: Record<string, AuthMeta> = {
  none: {
    label: 'None',
    desc: 'No API-specific authentication. API calls use the same auth as browser traffic (or none if WAN/LAN is none).',
    icon: UnlockKeyhole,
    tone: 'neutral',
  },
  cf_service_token: {
    label: 'CF Service Token',
    desc: 'Cloudflare Access service token. Machine-to-machine calls send CF-Access-Client-Id and CF-Access-Client-Secret headers. Browsers are unaffected.',
    icon: Ticket,
    tone: 'orange',
  },
  authentik_bearer: {
    label: 'Authentik Bearer',
    desc: 'Authentik bearer token. API calls send Authorization: Bearer <token>. The Authentik proxy provider validates the token.',
    icon: Fingerprint,
    tone: 'blue',
  },
  app_native_key: {
    label: 'App-Native Key',
    desc: 'The application has its own API key mechanism (e.g., Jellyfin API keys). No external auth layer for API access.',
    icon: KeyRound,
    tone: 'green',
  },
};

const STATUS_INFO: Record<string, AuthMeta> = {
  ok: {
    label: 'OK',
    desc: 'Auth is properly configured for this host.',
    icon: ShieldCheck,
    tone: 'green',
  },
  warning: {
    label: 'Warning',
    desc: 'Auth works but has a non-ideal configuration (e.g., split WAN/LAN modes, forward_auth without CF bypass).',
    icon: AlertTriangle,
    tone: 'yellow',
  },
  error: {
    label: 'Error',
    desc: 'Auth is missing or broken (e.g., WAN-exposed host with no auth, double-login risk from CF Access + forward_auth without bypass).',
    icon: ShieldX,
    tone: 'red',
  },
  unknown: {
    label: 'Unknown',
    desc: "Auth state couldn't be determined (e.g., Authentik/CF Access API unavailable).",
    icon: ShieldQuestion,
    tone: 'gray',
  },
};

// Section metadata — color-coded by traffic type
const SECTION_META = {
  wan: { label: 'WAN Auth', subtitle: 'internet-facing traffic', icon: Globe, color: 'blue' },
  lan: { label: 'LAN Auth', subtitle: 'internal traffic', icon: Wifi, color: 'green' },
  api: { label: 'API Auth', subtitle: 'machine-to-machine', icon: Terminal, color: 'purple' },
  status: { label: 'Status', subtitle: 'overall health', icon: ShieldCheck, color: 'gray' },
} as const;

// ─── Badge component with tooltip ────────────────────────────────────────────

function AuthBadge({ value, info, showIcon = true }: { value: string; info: Record<string, AuthMeta>; showIcon?: boolean }) {
  const meta = info[value] ?? { label: value, desc: '', icon: HelpCircle, tone: 'gray' };
  const cls = value === 'none' ? 'auth-badge none' : `auth-badge ${value.replace(/_/g, '-')}`;
  const Icon = meta.icon;
  return (
    <span className={cls} title={meta.desc}>
      {showIcon && <Icon size={12} className="auth-badge-icon" />}
      {meta.label}
      <HelpCircle size={11} className="auth-badge-help" />
    </span>
  );
}

function StatusIcon({ status }: { status: AuthStatus }) {
  const meta = STATUS_INFO[status] ?? STATUS_INFO.unknown;
  const Icon = meta.icon;
  return <Icon size={16} className={`auth-status-icon ${status}`} title={meta.desc} />;
}

// ─── Legend section card ────────────────────────────────────────────────────

function LegendSection({ sectionKey, info }: { sectionKey: keyof typeof SECTION_META; info: Record<string, AuthMeta> }) {
  const meta = SECTION_META[sectionKey];
  const SectionIcon = meta.icon;
  return (
    <div className={`auth-legend-card color-${meta.color}`}>
      <div className="auth-legend-card-header">
        <span className="auth-legend-card-icon"><SectionIcon size={18} /></span>
        <div>
          <h4>{meta.label}</h4>
          <span className="auth-legend-subtitle">{meta.subtitle}</span>
        </div>
      </div>
      <div className="auth-legend-items">
        {Object.entries(info).map(([key, authMeta]) => {
          const Icon = authMeta.icon;
          return (
            <div key={key} className={`auth-legend-item tone-${authMeta.tone}`}>
              <span className="auth-legend-item-icon"><Icon size={14} /></span>
              <div className="auth-legend-item-content">
                <AuthBadge value={key} info={info} />
                <span className="auth-legend-desc">{authMeta.desc}</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ─── Visual flow diagram ────────────────────────────────────────────────────

function FlowDiagram() {
  return (
    <div className="auth-flow-diagram">
      <div className="auth-flow-row">
        <div className="auth-flow-node wan">
          <Globe size={20} />
          <span>Internet</span>
        </div>
        <div className="auth-flow-arrow">
          <span className="auth-flow-label">CF Access</span>
          <span className="auth-flow-line">→</span>
        </div>
        <div className="auth-flow-node cf">
          <Cloud size={20} />
          <span>Cloudflare Tunnel</span>
        </div>
        <div className="auth-flow-arrow">
          <span className="auth-flow-label">Forward Auth</span>
          <span className="auth-flow-line">→</span>
        </div>
        <div className="auth-flow-node caddy">
          <Server size={20} />
          <span>Caddy</span>
        </div>
        <div className="auth-flow-arrow">
          <span className="auth-flow-label">App-Native</span>
          <span className="auth-flow-line">→</span>
        </div>
        <div className="auth-flow-node app">
          <Monitor size={20} />
          <span>Application</span>
        </div>
      </div>
      <div className="auth-flow-explanation">
        <p><strong>WAN traffic</strong> flows: Internet → Cloudflare (CF Access login) → Tunnel → Caddy (forward_auth / Authentik) → App (own login)</p>
        <p><strong>LAN traffic</strong> flows: LAN device → Caddy (forward_auth / Authentik) → App (own login). No Cloudflare in the path.</p>
        <p><strong>API traffic</strong> uses service tokens (CF) or bearer tokens (Authentik) instead of browser login flows.</p>
      </div>
    </div>
  );
}

// ─── Legend / help panel ────────────────────────────────────────────────────

function AuthLegend({ open, onToggle }: { open: boolean; onToggle: () => void }) {
  return (
    <div className="auth-legend">
      <button type="button" className="auth-legend-toggle" onClick={onToggle}>
        <HelpCircle size={14} />
        <span>Auth Type Reference</span>
        <span className="auth-legend-chevron">{open ? '▾' : '▸'}</span>
      </button>
      {open && (
        <div className="auth-legend-content">
          <FlowDiagram />
          <div className="auth-legend-grid">
            <LegendSection sectionKey="wan" info={WAN_AUTH_INFO} />
            <LegendSection sectionKey="lan" info={LAN_AUTH_INFO} />
            <LegendSection sectionKey="api" info={API_AUTH_INFO} />
            <LegendSection sectionKey="status" info={STATUS_INFO} />
          </div>
        </div>
      )}
    </div>
  );
}

// ─── Host row ────────────────────────────────────────────────────────────────

function HostRow({ host }: { host: HostAuth }) {
  const [expanded, setExpanded] = useState(false);
  const hasDetails = Boolean(
    host.cf_access_app_id ||
    host.authentik_provider_pk ||
    (host.notes && host.notes.length > 0)
  );

  const isStale = host._stale;

  return (
    <>
      <tr className={`auth-row ${isStale ? 'stale' : ''}`} onClick={() => hasDetails && setExpanded(e => !e)}>
        <td className="auth-hostname">
          {hasDetails && <span className="auth-expand">{expanded ? '▾' : '▸'}</span>}
          <Network size={13} className="auth-hostname-icon" />
          {host.hostname}
          {host.wan_exposed && <Globe size={12} className="auth-wan-icon" title="WAN-exposed" />}
          {isStale && <span className="auth-stale-tag">enriching…</span>}
        </td>
        <td className="auth-cell">
          {host.wan_exposed ? <AuthBadge value={host.wan_auth} info={WAN_AUTH_INFO} /> : <span className="auth-na">N/A</span>}
        </td>
        <td className="auth-cell">
          <AuthBadge value={host.lan_auth} info={LAN_AUTH_INFO} />
        </td>
        <td className="auth-cell">
          <AuthBadge value={host.api_auth} info={API_AUTH_INFO} />
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

// ─── Main component ──────────────────────────────────────────────────────────

type StreamState = 'idle' | 'loading-base' | 'enriching' | 'done' | 'error';

export function AuthFlowsTab() {
  const [hosts, setHosts] = useState<Map<string, HostAuth>>(new Map());
  const [streamState, setStreamState] = useState<StreamState>('idle');
  const [sources, setSources] = useState<{ cloudflare_access: boolean; authentik: boolean } | null>(null);
  const [errors, setErrors] = useState<string[]>([]);
  const [legendOpen, setLegendOpen] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);

  const load = useCallback(() => {
    // Close any existing connection.
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    setHosts(new Map());
    setErrors([]);
    setSources(null);
    setStreamState('loading-base');

    const es = new EventSource('/api/auth/inventory/stream');
    eventSourceRef.current = es;

    es.addEventListener('base', (e: MessageEvent) => {
      const data = JSON.parse(e.data);
      const newMap = new Map<string, HostAuth>();
      for (const h of data.hosts ?? []) {
        // Mark as stale — will be updated by enrich events.
        h._stale = true;
        newMap.set(h.hostname, h);
      }
      setHosts(newMap);
      setStreamState('enriching');
    });

    es.addEventListener('enrich', (e: MessageEvent) => {
      const data = JSON.parse(e.data);
      setHosts(prev => {
        const next = new Map(prev);
        for (const h of data.hosts ?? []) {
          h._stale = false;
          next.set(h.hostname, h);
        }
        return next;
      });
    });

    es.addEventListener('error', (e: MessageEvent) => {
      // SSE 'error' events from the server have data; native ES errors don't.
      if (e.data) {
        try {
          const data = JSON.parse(e.data);
          setErrors(prev => [...prev, `${data.source ?? 'unknown'}: ${data.error ?? 'unknown error'}`]);
        } catch {
          setErrors(prev => [...prev, 'Unknown stream error']);
        }
      }
    });

    es.addEventListener('done', (e: MessageEvent) => {
      const data = JSON.parse(e.data);
      setSources(data.sources ?? null);
      // Mark all remaining stale hosts as non-stale.
      setHosts(prev => {
        const next = new Map(prev);
        for (const [key, h] of next) {
          h._stale = false;
        }
        return next;
      });
      setStreamState('done');
      es.close();
      eventSourceRef.current = null;
    });

    // Native error handler (connection failed, not server-sent error event).
    es.onerror = () => {
      if (streamState !== 'done') {
        setStreamState(prev => prev === 'done' ? prev : 'error');
        es.close();
        eventSourceRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    void load();
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, [load]);

  // Sort hosts by hostname for display.
  const sortedHosts = Array.from(hosts.values()).sort((a, b) => a.hostname.localeCompare(b.hostname));

  const okCount = sortedHosts.filter(h => h.status === 'ok').length;
  const warnCount = sortedHosts.filter(h => h.status === 'warning').length;
  const errCount = sortedHosts.filter(h => h.status === 'error').length;
  const wanExposedCount = sortedHosts.filter(h => h.wan_exposed).length;
  const staleCount = sortedHosts.filter(h => h._stale).length;

  const isLoading = streamState === 'idle' || streamState === 'loading-base';
  const isEnriching = streamState === 'enriching';

  return (
    <main className="dashboard-shell auth-shell">
      <div className="auth-header">
        <div className="auth-header-title">
          <ShieldCheck size={20} />
          <h2>Auth Flows</h2>
          {isLoading && <span className="auth-phase-tag"><Loader2 size={12} className="spin" /> Loading hosts…</span>}
          {isEnriching && <span className="auth-phase-tag enriching"><Loader2 size={12} className="spin" /> Enriching ({staleCount} pending)…</span>}
          {streamState === 'done' && <span className="auth-phase-tag done"><CheckCircle2 size={12} /> Complete</span>}
        </div>
        <button type="button" className="btn-sm" onClick={() => void load()} disabled={isLoading}>
          {isLoading ? <Loader2 size={14} className="spin" /> : <RefreshCw size={14} />} Refresh
        </button>
      </div>

      <AuthLegend open={legendOpen} onToggle={() => setLegendOpen(o => !o)} />

      {sources && (
        <div className="auth-sources">
          <span className={`auth-source-badge ${sources.cloudflare_access ? 'ok' : 'off'}`}>
            CF Access: {sources.cloudflare_access ? 'Connected' : 'Not configured'}
          </span>
          <span className={`auth-source-badge ${sources.authentik ? 'ok' : 'off'}`}>
            Authentik: {sources.authentik ? 'Connected' : 'Not configured'}
          </span>
        </div>
      )}

      {sortedHosts.length > 0 && (
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
      )}

      {errors.length > 0 && (
        <div className="auth-error">
          <ShieldX size={16} />
          <div>
            <strong>Some auth sources failed:</strong>
            <ul>
              {errors.map((err, i) => <li key={i}>{err}</li>)}
            </ul>
          </div>
        </div>
      )}

      {isLoading && sortedHosts.length === 0 ? (
        <div className="auth-loading">
          <Loader2 size={24} className="spin" />
          <span>Loading auth inventory...</span>
        </div>
      ) : (
        <div className="auth-table-wrap" id="entries-panel">
          <table className="auth-table">
            <thead>
              <tr>
                <th><Network size={13} /> Hostname</th>
                <th><Globe size={13} /> WAN Auth</th>
                <th><Wifi size={13} /> LAN Auth</th>
                <th><Terminal size={13} /> API Auth</th>
                <th><ShieldCheck size={13} /> Status</th>
              </tr>
            </thead>
            <tbody>
              {sortedHosts.length === 0 ? (
                <tr>
                  <td colSpan={5} className="auth-empty">No hosts discovered</td>
                </tr>
              ) : (
                sortedHosts.map(host => <HostRow key={host.hostname} host={host} />)
              )}
            </tbody>
          </table>
        </div>
      )}
    </main>
  );
}
