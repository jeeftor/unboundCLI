import '../styles/AuthFlowsTab.css';
import {
  AlertTriangle,
  ArrowLeftRight,
  CheckCircle2,
  ChevronDown,
  Cloud,
  Fingerprint,
  Globe,
  HelpCircle,
  KeyRound,
  Loader2,
  LockKeyhole,
  Monitor,
  Network,
  Pencil,
  RefreshCw,
  Route,
  Search,
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
  X,
} from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import type { ComponentType } from 'react';
import type {
  APIAuthMode,
  AuthStatus,
  HostAuth,
  LANAuthMode,
  WANAuthMode,
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
  api: { label: 'API Auth', subtitle: 'scripts & automation — how non-browser clients authenticate', icon: Terminal, color: 'purple' },
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

// ─── Pending changes tracking ────────────────────────────────────────────────

type AuthField = 'wan_auth' | 'lan_auth' | 'api_auth';

type PendingChange = {
  hostname: string;
  field: AuthField;
  oldValue: string;
  newValue: string;
};

function changesKey(hostname: string, field: AuthField) {
  return `${hostname}::${field}`;
}

// ─── Editable auth badge with inline dropdown ────────────────────────────────

function EditableAuthBadge({
  hostname,
  field,
  value,
  info,
  disabled,
  pendingValue,
  onChange,
}: {
  hostname: string;
  field: AuthField;
  value: string;
  info: Record<string, AuthMeta>;
  disabled?: boolean;
  pendingValue?: string;
  onChange: (hostname: string, field: AuthField, newValue: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLSpanElement>(null);
  const displayValue = pendingValue ?? value;
  const isPending = pendingValue !== undefined && pendingValue !== value;
  const meta = info[displayValue] ?? { label: displayValue, desc: '', icon: HelpCircle, tone: 'gray' };
  const Icon = meta.icon;
  const cls = displayValue === 'none' ? 'auth-badge none' : `auth-badge ${displayValue.replace(/_/g, '-')}`;

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  if (disabled) {
    return <span className="auth-na" title="This host is not exposed to the internet (no Cloudflare tunnel). WAN auth doesn't apply.">Not exposed</span>;
  }

  return (
    <span className="auth-badge-editable" ref={ref}>
      <button
        type="button"
        className={`${cls} ${isPending ? 'pending' : ''}`}
        onClick={(e) => {
          e.stopPropagation();
          setOpen(o => !o);
        }}
        title={meta.desc + (isPending ? ' (pending change)' : '')}
      >
        <Icon size={12} className="auth-badge-icon" />
        {meta.label}
        {isPending && <span className="auth-badge-pending-dot" />}
        <ChevronDown size={10} className="auth-badge-chevron" />
      </button>
      {open && (
        <div className="auth-dropdown" onClick={e => e.stopPropagation()}>
          {Object.entries(info).map(([key, m]) => {
            const ItemIcon = m.icon;
            const isSelected = key === displayValue;
            return (
              <button
                key={key}
                type="button"
                className={`auth-dropdown-item ${isSelected ? 'selected' : ''}`}
                onClick={() => {
                  onChange(hostname, field, key);
                  setOpen(false);
                }}
                title={m.desc}
              >
                <ItemIcon size={13} className="auth-dropdown-item-icon" />
                <span className="auth-dropdown-item-label">{m.label}</span>
                {isSelected && <CheckCircle2 size={12} className="auth-dropdown-item-check" />}
              </button>
            );
          })}
        </div>
      )}
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

function HostRow({
  host,
  pendingChanges,
  onBadgeChange,
  onEditHost,
}: {
  host: HostAuth;
  pendingChanges: Map<string, PendingChange>;
  onBadgeChange: (hostname: string, field: AuthField, newValue: string) => void;
  onEditHost: (hostname: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const hasDetails = Boolean(
    host.cf_access_app_id ||
    host.authentik_provider_pk ||
    (host.notes && host.notes.length > 0)
  );

  const isStale = host._stale;
  const wanPending = pendingChanges.get(changesKey(host.hostname, 'wan_auth'))?.newValue;
  const lanPending = pendingChanges.get(changesKey(host.hostname, 'lan_auth'))?.newValue;
  const apiPending = pendingChanges.get(changesKey(host.hostname, 'api_auth'))?.newValue;
  const hasPendingChanges = pendingChanges.size > 0 && Array.from(pendingChanges.values()).some(c => c.hostname === host.hostname);

  return (
    <>
      <tr className={`auth-row ${isStale ? 'stale' : ''} ${hasPendingChanges ? 'has-pending' : ''}`}>
        <td className="auth-hostname" onClick={() => hasDetails && setExpanded(e => !e)}>
          <span className="auth-expand">{hasDetails ? (expanded ? '▾' : '▸') : ''}</span>
          <Network size={13} className="auth-hostname-icon" />
          {host.hostname}
          {host.wan_exposed && <Globe size={12} className="auth-wan-icon" title="WAN-exposed" />}
          {isStale && <span className="auth-stale-tag">enriching…</span>}
          {hasPendingChanges && <span className="auth-pending-tag" title="Has pending changes">{pendingChanges.size > 0 ? Array.from(pendingChanges.values()).filter(c => c.hostname === host.hostname).length : 0} pending</span>}
          <button type="button" className="auth-edit-btn" onClick={(e) => { e.stopPropagation(); onEditHost(host.hostname); }} title="Edit auth configuration">
            <Pencil size={12} />
          </button>
        </td>
        <td className="auth-cell">
          <EditableAuthBadge
            hostname={host.hostname}
            field="wan_auth"
            value={host.wan_auth}
            info={WAN_AUTH_INFO}
            disabled={!host.wan_exposed}
            pendingValue={wanPending}
            onChange={onBadgeChange}
          />
        </td>
        <td className="auth-cell">
          <EditableAuthBadge
            hostname={host.hostname}
            field="lan_auth"
            value={host.lan_auth}
            info={LAN_AUTH_INFO}
            pendingValue={lanPending}
            onChange={onBadgeChange}
          />
        </td>
        <td className="auth-cell">
          <EditableAuthBadge
            hostname={host.hostname}
            field="api_auth"
            value={host.api_auth}
            info={API_AUTH_INFO}
            pendingValue={apiPending}
            onChange={onBadgeChange}
          />
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
  const [search, setSearch] = useState('');
  const [pendingChanges, setPendingChanges] = useState<Map<string, PendingChange>>(new Map());
  const [editModalHost, setEditModalHost] = useState<string | null>(null);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [confirmDeploy, setConfirmDeploy] = useState(false);
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

  // ─── Pending changes handlers ──────────────────────────────────────────────

  const handleBadgeChange = useCallback((hostname: string, field: AuthField, newValue: string) => {
    setPendingChanges(prev => {
      const next = new Map(prev);
      const host = hosts.get(hostname);
      if (!host) return prev;
      const oldValue = host[field] as string;
      const key = changesKey(hostname, field);
      if (newValue === oldValue) {
        next.delete(key);
      } else {
        next.set(key, { hostname, field, oldValue, newValue });
      }
      return next;
    });
  }, [hosts]);

  const handleRevertChange = useCallback((hostname: string, field: AuthField) => {
    setPendingChanges(prev => {
      const next = new Map(prev);
      next.delete(changesKey(hostname, field));
      return next;
    });
  }, []);

  const handleRevertAll = useCallback(() => {
    setPendingChanges(new Map());
  }, []);

  const handleRevertHost = useCallback((hostname: string) => {
    setPendingChanges(prev => {
      const next = new Map(prev);
      for (const [key, change] of next) {
        if (change.hostname === hostname) next.delete(key);
      }
      return next;
    });
  }, []);

  // "Deploy" opens a confirmation modal — UI-only, no actual commands issued.
  const handleDeploy = useCallback(() => {
    setConfirmDeploy(true);
  }, []);

  const handleConfirmDeploy = useCallback(() => {
    // In the future, this would POST pending changes to a backend endpoint.
    // For now, just clear them.
    setPendingChanges(new Map());
    setConfirmDeploy(false);
    setReviewOpen(false);
  }, []);

  const handleCancelDeploy = useCallback(() => {
    setConfirmDeploy(false);
  }, []);

  const pendingChangesList = Array.from(pendingChanges.values());
  const pendingHosts = new Set(pendingChangesList.map(c => c.hostname));

  // Sort hosts by hostname for display, filtered by search query.
  const sortedHosts = Array.from(hosts.values())
    .sort((a, b) => a.hostname.localeCompare(b.hostname));
  const filteredHosts = search.trim()
    ? sortedHosts.filter(h => h.hostname.toLowerCase().includes(search.trim().toLowerCase()))
    : sortedHosts;

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
        <>
          {sortedHosts.length > 0 && (
            <div className="auth-search-bar">
              <Search size={14} className="auth-search-icon" />
              <input
                type="text"
                className="auth-search-input"
                placeholder="Filter hosts…"
                value={search}
                onChange={e => setSearch(e.target.value)}
                aria-label="Filter hosts by name"
              />
              {search && (
                <button type="button" className="auth-search-clear" onClick={() => setSearch('')} aria-label="Clear filter">
                  <X size={13} />
                </button>
              )}
              <span className="auth-search-count">
                {filteredHosts.length}/{sortedHosts.length}
              </span>
            </div>
          )}
          <div className="auth-table-wrap" id="entries-panel">
            <table className="auth-table">
              <thead>
                <tr>
                  <th><Network size={13} /> Hostname <HelpCircle size={11} className="th-info" title="The hostname as configured in Caddy. Click rows with ▸ to see detailed auth config." /></th>
                  <th><Globe size={13} /> WAN Auth <HelpCircle size={11} className="th-info" title="How browser traffic from the internet is authenticated. 'Not exposed' means the host has no Cloudflare tunnel and isn't reachable from the internet." /></th>
                  <th><Wifi size={13} /> LAN Auth <HelpCircle size={11} className="th-info" title="How browser traffic from your local network is authenticated. 'None' means the app is directly accessible (it may have its own login). 'Forward Auth' means Caddy delegates to Authentik." /></th>
                  <th><Terminal size={13} /> API Auth <HelpCircle size={11} className="th-info" title="How scripts, automation, and other non-browser clients authenticate. Uses tokens/keys instead of interactive login. 'None' means API calls use the same auth as browser traffic." /></th>
                  <th><ShieldCheck size={13} /> Status <HelpCircle size={11} className="th-info" title="Overall auth health. OK = properly configured. Warning = works but non-ideal. Error = missing or broken auth (e.g., WAN-exposed with no auth, double-login risk)." /></th>
                </tr>
              </thead>
              <tbody>
                {sortedHosts.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="auth-empty">No hosts discovered</td>
                  </tr>
                ) : filteredHosts.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="auth-empty">No hosts match "{search}"</td>
                  </tr>
                ) : (
                  filteredHosts.map(host => (
                    <HostRow
                      key={host.hostname}
                      host={host}
                      pendingChanges={pendingChanges}
                      onBadgeChange={handleBadgeChange}
                      onEditHost={setEditModalHost}
                    />
                  ))
                )}
              </tbody>
            </table>
          </div>
        </>
      )}

      {/* Pending changes bar */}
      {pendingChangesList.length > 0 && (
        <div className="auth-pending-bar">
          <div className="auth-pending-bar-info">
            <AlertTriangle size={16} />
            <span>
              <strong>{pendingChangesList.length}</strong> pending change{pendingChangesList.length !== 1 ? 's' : ''} across{' '}
              <strong>{pendingHosts.size}</strong> host{pendingHosts.size !== 1 ? 's' : ''}
            </span>
          </div>
          <div className="auth-pending-bar-actions">
            <button type="button" className="btn-sm" onClick={() => setReviewOpen(true)}>
              Review Changes
            </button>
            <button type="button" className="btn-sm btn-warn" onClick={handleRevertAll}>
              Discard All
            </button>
          </div>
        </div>
      )}

      {/* Review changes modal */}
      {reviewOpen && pendingChangesList.length > 0 && (
        <div className="auth-modal-overlay" onClick={() => setReviewOpen(false)}>
          <div className="auth-modal" onClick={e => e.stopPropagation()}>
            <div className="auth-modal-header">
              <h3>Review Pending Changes</h3>
              <button type="button" className="auth-modal-close" onClick={() => setReviewOpen(false)}>
                <X size={18} />
              </button>
            </div>
            <div className="auth-modal-body">
              <table className="auth-review-table">
                <thead>
                  <tr>
                    <th>Hostname</th>
                    <th>Field</th>
                    <th>Current</th>
                    <th />
                    <th>New</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {pendingChangesList.map(change => {
                    const fieldInfo = change.field === 'wan_auth' ? WAN_AUTH_INFO
                      : change.field === 'lan_auth' ? LAN_AUTH_INFO
                      : API_AUTH_INFO;
                    const OldIcon = fieldInfo[change.oldValue]?.icon ?? HelpCircle;
                    const NewIcon = fieldInfo[change.newValue]?.icon ?? HelpCircle;
                    return (
                      <tr key={changesKey(change.hostname, change.field)}>
                        <td className="auth-review-host">{change.hostname}</td>
                        <td className="auth-review-field">{change.field.replace('_', ' ')}</td>
                        <td className="auth-review-old">
                          <OldIcon size={12} /> {fieldInfo[change.oldValue]?.label ?? change.oldValue}
                        </td>
                        <td className="auth-review-arrow">→</td>
                        <td className="auth-review-new">
                          <NewIcon size={12} /> {fieldInfo[change.newValue]?.label ?? change.newValue}
                        </td>
                        <td className="auth-review-revert">
                          <button type="button" className="btn-sm" onClick={() => handleRevertChange(change.hostname, change.field)}>
                            <X size={12} /> Revert
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
              <div className="auth-modal-note">
                <HelpCircle size={14} />
                <span>These changes are UI-only — no actual commands will be issued. Deploy is a placeholder that clears the pending list.</span>
              </div>
            </div>
            <div className="auth-modal-footer">
              <button type="button" className="btn-sm" onClick={handleRevertAll}>Discard All</button>
              <button type="button" className="btn-sm btn-primary" onClick={handleDeploy}>
                <CheckCircle2 size={14} /> Deploy (Preview)
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Deploy confirmation modal */}
      {confirmDeploy && pendingChangesList.length > 0 && (
        <div className="auth-modal-overlay" onClick={handleCancelDeploy}>
          <div className="auth-modal auth-modal-confirm" onClick={e => e.stopPropagation()}>
            <div className="auth-modal-header">
              <h3>Confirm Deploy</h3>
              <button type="button" className="auth-modal-close" onClick={handleCancelDeploy}>
                <X size={18} />
              </button>
            </div>
            <div className="auth-modal-body">
              <div className="auth-confirm-warning">
                <AlertTriangle size={28} />
                <div>
                  <p className="auth-confirm-title">
                    Apply <strong>{pendingChangesList.length}</strong> auth change{pendingChangesList.length !== 1 ? 's' : ''} to{' '}
                    <strong>{pendingHosts.size}</strong> host{pendingHosts.size !== 1 ? 's' : ''}?
                  </p>
                  <p className="auth-confirm-subtitle">
                    This will modify Caddy and/or Cloudflare Access configurations.
                  </p>
                </div>
              </div>
              <div className="auth-confirm-changes">
                {pendingChangesList.map(change => {
                  const fieldInfo = change.field === 'wan_auth' ? WAN_AUTH_INFO
                    : change.field === 'lan_auth' ? LAN_AUTH_INFO
                    : API_AUTH_INFO;
                  const OldIcon = fieldInfo[change.oldValue]?.icon ?? HelpCircle;
                  const NewIcon = fieldInfo[change.newValue]?.icon ?? HelpCircle;
                  return (
                    <div key={changesKey(change.hostname, change.field)} className="auth-confirm-change-row">
                      <span className="auth-confirm-host">{change.hostname}</span>
                      <span className="auth-confirm-field">{change.field.replace('_', ' ')}</span>
                      <span className="auth-confirm-old"><OldIcon size={12} /> {fieldInfo[change.oldValue]?.label ?? change.oldValue}</span>
                      <span className="auth-confirm-arrow">→</span>
                      <span className="auth-confirm-new"><NewIcon size={12} /> {fieldInfo[change.newValue]?.label ?? change.newValue}</span>
                    </div>
                  );
                })}
              </div>
              <div className="auth-modal-note">
                <HelpCircle size={14} />
                <span>Preview mode — no actual commands will be issued. This is a UI demonstration only.</span>
              </div>
            </div>
            <div className="auth-modal-footer">
              <button type="button" className="btn-sm" onClick={handleCancelDeploy}>Cancel</button>
              <button type="button" className="btn-sm btn-primary" onClick={handleConfirmDeploy}>
                <CheckCircle2 size={14} /> Confirm & Deploy
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit modal per host */}
      {editModalHost && hosts.get(editModalHost) && (
        <EditHostModal
          host={hosts.get(editModalHost)!}
          pendingChanges={pendingChanges}
          onChange={handleBadgeChange}
          onRevert={handleRevertHost}
          onClose={() => setEditModalHost(null)}
        />
      )}
    </main>
  );
}

// ─── Edit host modal ─────────────────────────────────────────────────────────

function EditHostModal({
  host,
  pendingChanges,
  onChange,
  onRevert,
  onClose,
}: {
  host: HostAuth;
  pendingChanges: Map<string, PendingChange>;
  onChange: (hostname: string, field: AuthField, newValue: string) => void;
  onRevert: (hostname: string) => void;
  onClose: () => void;
}) {
  const wanPending = pendingChanges.get(changesKey(host.hostname, 'wan_auth'))?.newValue;
  const lanPending = pendingChanges.get(changesKey(host.hostname, 'lan_auth'))?.newValue;
  const apiPending = pendingChanges.get(changesKey(host.hostname, 'api_auth'))?.newValue;
  const hostPendingCount = Array.from(pendingChanges.values()).filter(c => c.hostname === host.hostname).length;

  function FieldRow({
    label,
    field,
    info,
    value,
    pendingValue,
    disabled,
  }: {
    label: string;
    field: AuthField;
    info: Record<string, AuthMeta>;
    value: string;
    pendingValue?: string;
    disabled?: boolean;
  }) {
    const isPending = pendingValue !== undefined && pendingValue !== value;
    const displayValue = pendingValue ?? value;
    return (
      <div className={`auth-edit-field ${disabled ? 'disabled' : ''} ${isPending ? 'pending' : ''}`}>
        <label>{label}</label>
        <div className="auth-edit-field-control">
          {disabled ? (
            <span className="auth-na">Not exposed</span>
          ) : (
            <select
              value={displayValue}
              onChange={e => onChange(host.hostname, field, e.target.value)}
              className="auth-edit-select"
            >
              {Object.entries(info).map(([key, m]) => (
                <option key={key} value={key}>{m.label}</option>
              ))}
            </select>
          )}
          {isPending && (
            <button type="button" className="btn-sm" onClick={() => onChange(host.hostname, field, value)}>
              <X size={12} /> Revert
            </button>
          )}
        </div>
        {info[displayValue] && (
          <p className="auth-edit-field-desc">{info[displayValue].desc}</p>
        )}
      </div>
    );
  }

  return (
    <div className="auth-modal-overlay" onClick={onClose}>
      <div className="auth-modal auth-modal-edit" onClick={e => e.stopPropagation()}>
        <div className="auth-modal-header">
          <h3>Edit Auth: {host.hostname}</h3>
          <button type="button" className="auth-modal-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>
        <div className="auth-modal-body">
          <div className="auth-edit-host-info">
            <span className={`auth-source-badge ${host.wan_exposed ? 'ok' : 'off'}`}>
              <Globe size={12} /> {host.wan_exposed ? 'WAN-exposed' : 'LAN-only'}
            </span>
            {host.has_forward_auth && (
              <span className="auth-source-badge ok"><Route size={12} /> Forward Auth</span>
            )}
            {host.cf_access_app_id && (
              <span className="auth-source-badge ok"><Cloud size={12} /> CF Access</span>
            )}
            {host.authentik_provider_pk && (
              <span className="auth-source-badge ok"><Fingerprint size={12} /> Authentik</span>
            )}
          </div>

          <FieldRow
            label="WAN Auth"
            field="wan_auth"
            info={WAN_AUTH_INFO}
            value={host.wan_auth}
            pendingValue={wanPending}
            disabled={!host.wan_exposed}
          />
          <FieldRow
            label="LAN Auth"
            field="lan_auth"
            info={LAN_AUTH_INFO}
            value={host.lan_auth}
            pendingValue={lanPending}
          />
          <FieldRow
            label="API Auth"
            field="api_auth"
            info={API_AUTH_INFO}
            value={host.api_auth}
            pendingValue={apiPending}
          />

          {/* Current config details */}
          {(host.cf_access_app_id || host.authentik_provider_pk) && (
            <div className="auth-edit-current-config">
              <h4>Current Configuration</h4>
              {host.cf_access_app_id && (
                <div className="auth-edit-config-row">
                  <Cloud size={13} />
                  <span>CF Access: {host.cf_access_app_domain} ({host.cf_access_app_type})</span>
                </div>
              )}
              {host.authentik_provider_pk && (
                <div className="auth-edit-config-row">
                  <Fingerprint size={13} />
                  <span>Authentik: {host.authentik_app_slug} ({host.authentik_provider_mode})</span>
                </div>
              )}
              {host.cf_access_decisions && host.cf_access_decisions.length > 0 && (
                <div className="auth-edit-config-row">
                  <ShieldCheck size={13} />
                  <span>Policies: {host.cf_access_decisions.join(', ')}</span>
                </div>
              )}
            </div>
          )}

          {host.notes && host.notes.length > 0 && (
            <div className="auth-edit-notes">
              <AlertTriangle size={14} />
              <ul>
                {host.notes.map((n, i) => <li key={i}>{n}</li>)}
              </ul>
            </div>
          )}
        </div>
        <div className="auth-modal-footer">
          {hostPendingCount > 0 && (
            <button type="button" className="btn-sm btn-warn" onClick={() => onRevert(host.hostname)}>
              Revert {hostPendingCount} change{hostPendingCount !== 1 ? 's' : ''}
            </button>
          )}
          <button type="button" className="btn-sm btn-primary" onClick={onClose}>
            Done
          </button>
        </div>
      </div>
    </div>
  );
}
