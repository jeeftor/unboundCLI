import '../styles/VisualizeModal.css';
import { AlertTriangle, Globe, Loader2, Network, Server, ShieldCheck, ShieldX, Wifi, X } from 'lucide-react';
import { useMemo } from 'react';
import { buildLanRequestFlow, buildWanRequestFlow, detectAuthPattern } from '../lib/authMeta';
import { ReactFlowDiagram } from './ReactFlowDiagram';
import type { Step } from './ReactFlowDiagramNodes';
import { useStore } from '../store';
import type { Entry } from '../types';

export function VisualizeModal({ entry, onClose }: { entry: Entry; onClose: () => void }) {
  const cf = entry.cloudflare_status;
  const hasCF = cf?.configured;
  const hasForwardAuth = entry.has_forward_auth;
  const hasDNS = Boolean(entry.dns_resolved && entry.dns_resolved !== 'FAIL');
  const upstream = entry.caddy_upstream || 'unknown';

  // Auth data from store (cached, fetched once at startup)
  const auth = useStore((s) => s.authHosts.get(entry.hostname) ?? null);
  const authLoading = useStore((s) => s.authLoading);

  // Enriched auth info
  const hasCFAccess = auth?.cf_access_app_id !== undefined && auth.cf_access_app_id !== '';
  const hasBypass = auth?.cf_access_decisions?.includes('bypass') ?? false;
  const cfAppDomain = auth?.cf_access_app_domain;
  const cfDecisions = auth?.cf_access_decisions;
  const cfAppType = auth?.cf_access_app_type;
  const authentikSlug = auth?.authentik_app_slug;
  const authentikMode = auth?.authentik_provider_mode;
  const authNotes = auth?.notes;
  const wanExposed = auth?.wan_exposed ?? false;

  // Detect auth pattern
  const pattern = useMemo(() => {
    if (!auth) return null;
    return detectAuthPattern({
      wan_exposed: auth.wan_exposed,
      wan_auth: auth.wan_auth,
      lan_auth: auth.lan_auth,
      has_forward_auth: auth.has_forward_auth,
      cf_access_app_id: auth.cf_access_app_id,
      cf_access_decisions: auth.cf_access_decisions,
      notes: auth.notes,
    });
  }, [auth]);

  // ── Request flow step-by-step (what happens when a user hits the service) ──
  const wanFlow = useMemo(() => {
    if (!auth) return null;
    return buildWanRequestFlow({
      wan_exposed: auth.wan_exposed,
      wan_auth: auth.wan_auth,
      has_forward_auth: auth.has_forward_auth,
      cf_access_app_id: auth.cf_access_app_id,
      cf_access_decisions: auth.cf_access_decisions,
      authentik_provider_mode: auth.authentik_provider_mode,
      authentik_app_slug: auth.authentik_app_slug,
      hostname: entry.hostname,
      upstream,
    });
  }, [auth, entry.hostname, upstream]);

  const lanFlow = useMemo(() => {
    if (!auth) return null;
    return buildLanRequestFlow({
      has_forward_auth: auth.has_forward_auth,
      authentik_provider_mode: auth.authentik_provider_mode,
      authentik_app_slug: auth.authentik_app_slug,
      hostname: entry.hostname,
      upstream,
      dns_resolved: entry.dns_resolved || '',
      unbound_ip: entry.unbound_status?.ip,
    });
  }, [auth, entry.hostname, upstream, entry.dns_resolved, entry.unbound_status?.ip]);

  // ── Build WAN flow steps for React Flow ──
  const wanSteps = useMemo(() => {
    if (!hasCF) return null;

    const steps: Step[] = [
      { id: 'wan', nodeType: 'wan', label: 'Internet', arrowLabel: '' },
      { id: 'cf', nodeType: 'cf', label: 'Cloudflare', sublabel: cf?.tunnel_name || undefined },
    ];

    if (hasCFAccess) {
      steps.push({
        id: 'cf_access',
        nodeType: 'cf_access',
        label: hasBypass ? 'CF Access (bypass)' : 'CF Access (login)',
        sublabel: cfAppDomain || undefined,
        warn: hasBypass && !hasForwardAuth,
        arrowLabel: hasBypass ? 'bypass' : 'IdP login',
      });
    }

    steps.push({
      id: 'caddy',
      nodeType: 'caddy',
      label: 'Caddy',
      sublabel: entry.caddy_ip || undefined,
      arrowLabel: '',
    });

    if (hasForwardAuth) {
      const isDoubleLogin = hasCFAccess && !hasBypass;
      steps.push({
        id: 'authentik',
        nodeType: 'authentik',
        label: 'Authentik',
        sublabel: authentikSlug || undefined,
        inactive: isDoubleLogin,
        arrowLabel: 'forward_auth',
        arrowWarn: isDoubleLogin,
      });
      steps.push({
        id: 'upstream',
        nodeType: 'upstream',
        label: 'Service',
        sublabel: upstream,
        arrowLabel: isDoubleLogin ? 'login again!' : 'authorized',
        arrowWarn: isDoubleLogin,
      });
    } else {
      steps.push({
        id: 'upstream',
        nodeType: 'upstream',
        label: 'Service',
        sublabel: upstream,
        arrowLabel: '',
      });
    }

    return steps;
  }, [hasCF, hasCFAccess, hasBypass, hasForwardAuth, cf, cfAppDomain, authentikSlug, entry.caddy_ip, upstream]);

  // ── Build LAN flow steps for React Flow ──
  const lanSteps = useMemo(() => {
    const steps: Step[] = [
      { id: 'app', nodeType: 'app', label: 'LAN Client', sublabel: entry.dns_resolved || undefined, inactive: !hasDNS, arrowLabel: '' },
      { id: 'dns', nodeType: 'dns', label: 'Unbound', sublabel: entry.unbound_status?.ip || undefined, inactive: !hasDNS, arrowLabel: 'DNS' },
      { id: 'caddy-lan', nodeType: 'caddy', label: 'Caddy', sublabel: entry.caddy_ip || undefined, arrowLabel: '' },
    ];

    if (hasForwardAuth) {
      steps.push({
        id: 'authentik-lan',
        nodeType: 'authentik',
        label: 'Authentik',
        sublabel: authentikSlug || undefined,
        arrowLabel: 'forward_auth',
      });
      steps.push({
        id: 'upstream-lan',
        nodeType: 'upstream',
        label: 'Service',
        sublabel: upstream,
        arrowLabel: 'authorized',
      });
    } else {
      steps.push({
        id: 'upstream-lan',
        nodeType: 'upstream',
        label: 'Service',
        sublabel: upstream,
        arrowLabel: '',
      });
    }

    return steps;
  }, [hasDNS, hasForwardAuth, entry.dns_resolved, entry.unbound_status?.ip, entry.caddy_ip, upstream, authentikSlug]);

  return (
    <div className="modal-overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="modal visualize-modal">
        <div className="modal-header">
          <h3><Network size={15} /> {entry.hostname}</h3>
          <button type="button" className="modal-close" onClick={onClose}><X size={16} /></button>
        </div>
        <div className="modal-body visualize-body">

          {/* ── Auth pattern verdict ── */}
          {pattern && (
            <div className={`visualize-auth-verdict ${pattern.verdict}`}>
              <div className="visualize-auth-verdict-icon">
                {pattern.verdict === 'ok' && <ShieldCheck size={18} />}
                {pattern.verdict === 'warning' && <AlertTriangle size={18} />}
                {pattern.verdict === 'error' && <ShieldX size={18} />}
              </div>
              <div className="visualize-auth-verdict-content">
                <div className="visualize-auth-verdict-name">{pattern.name}</div>
                <div className="visualize-auth-verdict-summary">{pattern.summary}</div>
                <div className="visualize-auth-verdict-detail">{pattern.detail}</div>
              </div>
            </div>
          )}

          {/* ── WAN: diagram + request flow ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Globe size={13} /> WAN Path (Internet → Service)
            </div>
            {wanSteps ? (
              <>
                <ReactFlowDiagram steps={wanSteps} height={350} />
                {hasCFAccess && !hasBypass && hasForwardAuth && (
                  <div className="visualize-warn-inline" style={{ marginTop: 6 }}>
                    Double-login: users authenticate at CF Access, then again at Authentik.
                  </div>
                )}
                {hasCFAccess && hasBypass && hasForwardAuth && (
                  <div className="visualize-section-sub" style={{ marginTop: 6 }}>
                    CF Access bypasses so Authentik handles the single login.
                  </div>
                )}
              </>
            ) : (
              <div className="visualize-section-sub">
                Not exposed — no Cloudflare tunnel configured. This host is only reachable on the LAN.
              </div>
            )}

            {/* WAN request flow table (right below diagram) */}
            {wanFlow && <FlowTable steps={wanFlow} />}

            {/* CF Access details */}
            {hasCFAccess && (
              <div className="visualize-auth-detail">
                <div className="visualize-auth-detail-title">Cloudflare Access Policy</div>
                <dl>
                  {cfAppDomain && <><dt>App Domain</dt><dd>{cfAppDomain}</dd></>}
                  {cfAppType && <><dt>App Type</dt><dd>{cfAppType}</dd></>}
                  {cfDecisions && cfDecisions.length > 0 && <><dt>Policy Decisions</dt><dd>{cfDecisions.join(', ')}</dd></>}
                </dl>
              </div>
            )}

            {/* Authentik details */}
            {hasForwardAuth && (
              <div className="visualize-auth-detail">
                <div className="visualize-auth-detail-title">Authentik Forward Auth</div>
                <dl>
                  {authentikSlug && <><dt>App Slug</dt><dd>{authentikSlug}</dd></>}
                  {authentikMode && <><dt>Provider Mode</dt><dd>{authentikMode}</dd></>}
                  {!authentikSlug && !authentikMode && <><dt>Status</dt><dd>Configured (details loading…)</dd></>}
                </dl>
              </div>
            )}
          </div>

          {/* ── LAN: diagram + request flow ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Wifi size={13} /> LAN Path (Internal → Service)
            </div>
            <ReactFlowDiagram steps={lanSteps} height={350} />
            <div className="visualize-section-sub" style={{ marginTop: 6 }}>
              Client resolves <code>{entry.hostname}</code>
              {hasDNS ? ` → ${entry.dns_resolved}` : ' (not in DNS)'}
              {' → '}Caddy{hasForwardAuth ? ' → Authentik (forward_auth)' : ''}
              {' → '}Service <code>{upstream}</code>
            </div>

            {/* LAN request flow table (right below diagram) */}
            {lanFlow && <FlowTable steps={lanFlow} />}
          </div>

          {/* ── Auth configuration table ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Network size={13} /> Auth Configuration
              {authLoading && <Loader2 size={12} className="visualize-loading-spin" />}
            </div>
            <table className="visualize-config-table">
              <thead>
                <tr>
                  <th>Layer</th>
                  <th>Method</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                <ConfigRow label="WAN" value={auth?.wan_auth ? auth.wan_auth.replace(/_/g, ' ') : (authLoading ? '…' : 'none')} ok={auth?.wan_auth !== undefined && auth.wan_auth !== 'none'} />
                <ConfigRow label="LAN" value={auth?.lan_auth ? auth.lan_auth.replace(/_/g, ' ') : (authLoading ? '…' : 'none')} ok={auth?.lan_auth !== undefined && auth.lan_auth !== 'none'} />
                <ConfigRow label="API" value={auth?.api_auth ? auth.api_auth.replace(/_/g, ' ') : (authLoading ? '…' : 'none')} ok={auth?.api_auth !== undefined && auth.api_auth !== 'none'} />
                <ConfigRow label="CF Access" value={hasCFAccess ? 'configured' : 'none'} ok={hasCFAccess} />
                <ConfigRow label="Forward Auth" value={hasForwardAuth ? 'authentik' : 'none'} ok={hasForwardAuth} />
                <ConfigRow label="WAN Exposed" value={wanExposed ? 'yes' : (authLoading ? '…' : 'no')} ok={wanExposed} />
              </tbody>
            </table>

            {/* Auth notes */}
            {authNotes && authNotes.length > 0 && (
              <div className="visualize-auth-notes">
                <strong>Notes</strong>
                <ul>
                  {authNotes.map((n, i) => <li key={i}>{n}</li>)}
                </ul>
              </div>
            )}
          </div>

          {/* ── Service status table ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Server size={13} /> Service Status
            </div>
            <table className="visualize-config-table">
              <thead>
                <tr>
                  <th>Service</th>
                  <th>Detail</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                <ConfigRow label="DNS (Unbound)" value={entry.unbound_status?.ip || 'not configured'} ok={entry.unbound_status?.configured ?? false} />
                <ConfigRow label="AdGuard" value={entry.adguard_status?.ip || 'not configured'} ok={entry.adguard_status?.configured ?? false} />
                <ConfigRow label="DHCP" value={entry.dhcp_status?.ip || 'not configured'} ok={entry.dhcp_status?.configured ?? false} />
                <ConfigRow label="Cloudflare" value={cf?.configured ? cf.tunnel_name : 'not configured'} ok={cf?.configured ?? false} />
              </tbody>
            </table>
          </div>

        </div>
      </div>
    </div>
  );
}

function ConfigRow({ label, value, ok }: { label: string; value: string; ok: boolean }) {
  return (
    <tr>
      <td className="config-label">{label}</td>
      <td className="config-value">{value}</td>
      <td className={`config-status ${ok ? 'ok' : 'missing'}`}>
        {ok ? '✓' : '—'}
      </td>
    </tr>
  );
}

function FlowTable({ steps }: { steps: Array<{ step: number; actor: string; action: string; result: string; warn?: boolean }> }) {
  return (
    <table className="flow-table">
      <thead>
        <tr>
          <th>#</th>
          <th>Actor</th>
          <th>Action</th>
          <th>Result</th>
        </tr>
      </thead>
      <tbody>
        {steps.map((s) => (
          <tr key={s.step} className={s.warn ? 'flow-warn' : undefined}>
            <td className="flow-step-num">{s.step}</td>
            <td className="flow-actor">{s.actor}</td>
            <td className="flow-action">{s.action}</td>
            <td className="flow-result">{s.result}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
