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
  // Step numbers correspond to the WAN request flow table rows
  const wanSteps = useMemo(() => {
    if (!hasCF) return null;

    let n = 1;
    const steps: Step[] = [
      { id: 'wan', nodeType: 'wan', label: 'Internet', stepNum: n++, arrowLabel: '' },
      { id: 'cf', nodeType: 'cf', label: 'Cloudflare', sublabel: cf?.tunnel_name || undefined, stepNum: n++ },
    ];

    if (hasCFAccess) {
      if (hasBypass) {
        // Bypass: CF Access step 3, then skip IdP steps, tunnel is step 4
        steps.push({
          id: 'cf_access',
          nodeType: 'cf_access',
          label: 'CF Access (bypass)',
          sublabel: cfAppDomain || undefined,
          warn: hasBypass && !hasForwardAuth,
          stepNum: n++,
          arrowLabel: 'bypass',
        });
        // Skip IdP (4) and CF Access exchange (5) — those don't happen on bypass
        n += 2; // skip the IdP + exchange steps in the flow table
      } else {
        // Login: CF Access step 3, IdP step 4, CF Access step 5, tunnel step 6
        steps.push({
          id: 'cf_access',
          nodeType: 'cf_access',
          label: 'CF Access (login)',
          sublabel: cfAppDomain || undefined,
          stepNum: n++,
          // No arrow label here — the IdP login happens AT cf_access
        });
        n++; // skip IdP step (4) — it's between CF Access and the exchange
        n++; // skip CF Access exchange step (5)
      }
    }

    // Tunnel step
    n++; // tunnel step number

    steps.push({
      id: 'caddy',
      nodeType: 'caddy',
      label: 'Caddy',
      sublabel: entry.caddy_ip || undefined,
      stepNum: n++,
      arrowLabel: hasCFAccess && !hasBypass ? 'JWT verified' : '',
    });

    if (hasForwardAuth) {
      const isDoubleLogin = hasCFAccess && !hasBypass;
      steps.push({
        id: 'authentik',
        nodeType: 'authentik',
        label: 'Authentik',
        sublabel: authentikSlug || undefined,
        inactive: isDoubleLogin,
        stepNum: n++,
        arrowLabel: 'forward_auth',
        arrowWarn: isDoubleLogin,
      });
      // Skip the Authentik redirect + login again steps
      if (isDoubleLogin) n += 2;
      steps.push({
        id: 'upstream',
        nodeType: 'upstream',
        label: 'Service',
        sublabel: upstream,
        stepNum: n++,
        arrowLabel: isDoubleLogin ? 'login again!' : 'authorized',
        arrowWarn: isDoubleLogin,
      });
    } else {
      // Skip the "no forward_auth" step
      n++;
      steps.push({
        id: 'upstream',
        nodeType: 'upstream',
        label: 'Service',
        sublabel: upstream,
        stepNum: n++,
        arrowLabel: '',
      });
    }

    return steps;
  }, [hasCF, hasCFAccess, hasBypass, hasForwardAuth, cf, cfAppDomain, authentikSlug, entry.caddy_ip, upstream]);

  // ── Build LAN flow steps for React Flow ──
  const lanSteps = useMemo(() => {
    let n = 1;
    const steps: Step[] = [
      { id: 'app', nodeType: 'app', label: 'LAN Client', sublabel: entry.dns_resolved || undefined, inactive: !hasDNS, stepNum: n++, arrowLabel: '' },
      { id: 'dns', nodeType: 'dns', label: 'Unbound', sublabel: entry.unbound_status?.ip || undefined, inactive: !hasDNS, stepNum: n++, arrowLabel: 'DNS' },
      { id: 'caddy-lan', nodeType: 'caddy', label: 'Caddy', sublabel: entry.caddy_ip || undefined, stepNum: n++, arrowLabel: '' },
    ];

    if (hasForwardAuth) {
      // Skip the "forward_auth subrequest" step
      n++;
      steps.push({
        id: 'authentik-lan',
        nodeType: 'authentik',
        label: 'Authentik',
        sublabel: authentikSlug || undefined,
        stepNum: n++,
        arrowLabel: 'forward_auth',
      });
      // Skip the "validate session" step
      n++;
      // Skip the "forward_auth returned 200" step
      n++;
      steps.push({
        id: 'upstream-lan',
        nodeType: 'upstream',
        label: 'Service',
        sublabel: upstream,
        stepNum: n++,
        arrowLabel: 'authorized',
      });
    } else {
      // Skip the "no forward_auth" step
      n++;
      steps.push({
        id: 'upstream-lan',
        nodeType: 'upstream',
        label: 'Service',
        sublabel: upstream,
        stepNum: n++,
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

          {/* ── WAN: diagram + request flow side by side ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Globe size={13} /> WAN Path (Internet → Service)
            </div>
            {wanSteps ? (
              <div className="visualize-diagram-flow">
                <div className="visualize-diagram-side">
                  <ReactFlowDiagram steps={wanSteps} height={380} />
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
                </div>
                {wanFlow && (
                  <div className="visualize-flow-side">
                    <FlowTable steps={wanFlow} />
                  </div>
                )}
              </div>
            ) : (
              <div className="visualize-section-sub">
                Not exposed — no Cloudflare tunnel configured. This host is only reachable on the LAN.
              </div>
            )}

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

          {/* ── LAN: diagram + request flow side by side ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Wifi size={13} /> LAN Path (Internal → Service)
            </div>
            <div className="visualize-diagram-flow">
              <div className="visualize-diagram-side">
                <ReactFlowDiagram steps={lanSteps} height={380} />
                <div className="visualize-section-sub" style={{ marginTop: 6 }}>
                  Client resolves <code>{entry.hostname}</code>
                  {hasDNS ? ` → ${entry.dns_resolved}` : ' (not in DNS)'}
                  {' → '}Caddy{hasForwardAuth ? ' → Authentik (forward_auth)' : ''}
                  {' → '}Service <code>{upstream}</code>
                </div>
              </div>
              {lanFlow && (
                <div className="visualize-flow-side">
                  <FlowTable steps={lanFlow} />
                </div>
              )}
            </div>
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
    <div className="flow-steps">
      {steps.map((s) => (
        <div key={s.step} className={`flow-step ${s.warn ? 'flow-warn' : ''}`}>
          <div className="flow-step-num">{s.step}</div>
          <div className="flow-step-body">
            <div className="flow-step-actor">{s.actor}</div>
            <div className="flow-step-action">{s.action}</div>
            <div className="flow-step-result">{s.result}</div>
          </div>
        </div>
      ))}
    </div>
  );
}
