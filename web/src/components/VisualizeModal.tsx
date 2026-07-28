import '../styles/VisualizeModal.css';
import { AlertTriangle, Globe, Loader2, Network, Server, ShieldCheck, ShieldX, Smartphone, Wifi, X } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { api } from '../api/client';
import { detectAuthPattern } from '../lib/authMeta';
import { FlowArrow, FlowExplanation, FlowNode, FlowRow } from './FlowDiagram';
import type { Entry, HostAuth } from '../types';

export function VisualizeModal({ entry, onClose }: { entry: Entry; onClose: () => void }) {
  const cf = entry.cloudflare_status;
  const hasCF = cf?.configured;
  const hasForwardAuth = entry.has_forward_auth;
  const hasDNS = Boolean(entry.dns_resolved && entry.dns_resolved !== 'FAIL');
  const upstream = entry.caddy_upstream || 'unknown';

  // Fetch rich auth data (CF Access policies, Authentik providers)
  const [auth, setAuth] = useState<HostAuth | null>(null);
  const [authLoading, setAuthLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setAuthLoading(true);
    setAuth(null);
    api.authInventory()
      .then((res) => {
        if (cancelled) return;
        const match = res.hosts.find((h) => h.hostname === entry.hostname);
        setAuth(match ?? null);
      })
      .catch(() => { /* auth inventory may not be configured */ })
      .finally(() => { if (!cancelled) setAuthLoading(false); });
    return () => { cancelled = true; };
  }, [entry.hostname]);

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

  // ── Build WAN flow nodes dynamically based on auth pattern ──
  const wanNodes = useMemo(() => {
    if (!hasCF) return null;

    type Node = { variant: 'wan' | 'cf' | 'cf_access' | 'caddy' | 'authentik' | 'app_auth' | 'upstream'; label: string; sublabel?: string; active?: boolean };
    type Step = { node?: Node; arrowLabel?: string; arrowActive?: boolean };

    const steps: Step[] = [
      { node: { variant: 'wan', label: 'Internet' } },
      { node: { variant: 'cf', label: 'Cloudflare', sublabel: cf?.tunnel_name || undefined } },
    ];

    if (hasCFAccess) {
      // CF Access is a node in the flow — show whether it challenges or bypasses
      const accessLabel = hasBypass ? 'CF Access (bypass)' : 'CF Access (login)';
      steps.push({ arrowLabel: hasBypass ? 'bypass' : 'IdP login' });
      steps.push({ node: { variant: 'cf_access', label: accessLabel, sublabel: cfAppDomain || undefined } });
    }

    // After CF, traffic reaches Caddy
    steps.push({ node: { variant: 'caddy', label: 'Caddy', sublabel: entry.caddy_ip || undefined } });

    if (hasForwardAuth) {
      // Forward auth is a node — Authentik challenges before reaching the app
      const isDoubleLogin = hasCFAccess && !hasBypass;
      steps.push({ arrowLabel: 'forward_auth' });
      steps.push({
        node: {
          variant: 'authentik',
          label: 'Authentik',
          sublabel: authentikSlug || undefined,
          active: !isDoubleLogin,
        },
      });
      if (isDoubleLogin) {
        // Mark as double-login — the Authentik node shows a warning
        steps.push({ arrowLabel: 'login again!' });
      } else {
        steps.push({ arrowLabel: 'authorized' });
      }
    }

    // Final destination
    steps.push({ node: { variant: 'upstream', label: 'Service', sublabel: upstream } });

    return steps;
  }, [hasCF, hasCFAccess, hasBypass, hasForwardAuth, cf, cfAppDomain, authentikSlug, entry.caddy_ip, upstream]);

  // ── Build LAN flow nodes dynamically ──
  const lanNodes = useMemo(() => {
    type Node = { variant: 'app' | 'dns' | 'caddy' | 'authentik' | 'upstream'; label: string; sublabel?: string; active?: boolean };
    type Step = { node?: Node; arrowLabel?: string; arrowActive?: boolean };

    const steps: Step[] = [
      { node: { variant: 'app', label: 'LAN Client', sublabel: entry.dns_resolved || undefined, active: hasDNS } },
      { arrowLabel: 'DNS', arrowActive: hasDNS },
      { node: { variant: 'dns', label: 'Unbound', sublabel: entry.unbound_status?.ip || undefined, active: hasDNS } },
      { node: { variant: 'caddy', label: 'Caddy', sublabel: entry.caddy_ip || undefined } },
    ];

    if (hasForwardAuth) {
      steps.push({ arrowLabel: 'forward_auth' });
      steps.push({ node: { variant: 'authentik', label: 'Authentik', sublabel: authentikSlug || undefined } });
      steps.push({ arrowLabel: 'authorized' });
    }

    steps.push({ node: { variant: 'upstream', label: 'Service', sublabel: upstream } });

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

          {/* ── WAN path ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Globe size={13} /> WAN Path (Internet → Service)
            </div>
            <div className="auth-flow-diagram">
              {wanNodes ? (
                <FlowRow>
                  {wanNodes.map((step, i) => (
                    <span key={i}>
                      {step.arrowLabel !== undefined && (
                        <FlowArrow label={step.arrowLabel} active={step.arrowActive !== false} />
                      )}
                      {step.node && (
                        <FlowNode
                          variant={step.node.variant}
                          label={step.node.label}
                          sublabel={step.node.sublabel}
                          active={step.node.active !== false}
                        />
                      )}
                    </span>
                  ))}
                </FlowRow>
              ) : (
                <FlowRow>
                  <FlowNode variant="wan" label="Internet" active={false} />
                  <FlowArrow active={false} />
                  <FlowNode variant="cf" label="Cloudflare" active={false} />
                  <FlowNode variant="caddy" label="Caddy" active={false} />
                  <FlowNode variant="upstream" label="Service" active={false} />
                </FlowRow>
              )}
              <FlowExplanation>
                {wanNodes ? (
                  <p><strong>WAN traffic</strong> reaches the service through the chain above.{' '}
                    {hasCFAccess && !hasBypass && hasForwardAuth && (
                      <span className="visualize-warn-inline">Double-login: users authenticate at CF Access, then again at Authentik.</span>
                    )}
                    {hasCFAccess && hasBypass && hasForwardAuth && (
                      <span>CF Access bypasses so Authentik handles the single login.</span>
                    )}
                  </p>
                ) : (
                  <p><strong>WAN traffic</strong>: Not exposed — no Cloudflare tunnel configured. This host is only reachable on the LAN.</p>
                )}
              </FlowExplanation>
            </div>

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

          {/* ── LAN path ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Wifi size={13} /> LAN Path (Internal → Service)
            </div>
            <div className="auth-flow-diagram">
              <FlowRow>
                {lanNodes.map((step, i) => (
                  <span key={i}>
                    {step.arrowLabel !== undefined && (
                      <FlowArrow label={step.arrowLabel} active={step.arrowActive !== false} />
                    )}
                    {step.node && (
                      <FlowNode
                        variant={step.node.variant}
                        label={step.node.label}
                        sublabel={step.node.sublabel}
                        active={step.node.active !== false}
                      />
                    )}
                  </span>
                ))}
              </FlowRow>
              <FlowExplanation>
                <p>
                  <strong>LAN traffic</strong>: Client resolves <code>{entry.hostname}</code>
                  {hasDNS ? ` → ${entry.dns_resolved}` : ' (not in DNS)'}
                  {' → '}Caddy{hasForwardAuth ? ' → Authentik (forward_auth)' : ''}
                  {' → '}Service <code>{upstream}</code>
                </p>
              </FlowExplanation>
            </div>
          </div>

          {/* ── Auth configuration grid ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Network size={13} /> Auth Configuration
              {authLoading && <Loader2 size={12} className="visualize-loading-spin" />}
            </div>
            <div className="visualize-status-grid">
              <StatusTile label="WAN Auth" ok={auth?.wan_auth !== undefined && auth.wan_auth !== 'none'} detail={auth?.wan_auth ? auth.wan_auth.replace(/_/g, ' ') : (authLoading ? '…' : 'N/A')} />
              <StatusTile label="LAN Auth" ok={auth?.lan_auth !== undefined && auth.lan_auth !== 'none'} detail={auth?.lan_auth ? auth.lan_auth.replace(/_/g, ' ') : (authLoading ? '…' : 'N/A')} />
              <StatusTile label="API Auth" ok={auth?.api_auth !== undefined && auth.api_auth !== 'none'} detail={auth?.api_auth ? auth.api_auth.replace(/_/g, ' ') : (authLoading ? '…' : 'N/A')} />
              <StatusTile label="CF Access" ok={hasCFAccess} detail={hasCFAccess ? 'Configured' : 'None'} />
              <StatusTile label="Forward Auth" ok={hasForwardAuth} detail={hasForwardAuth ? 'Authentik' : 'None'} />
              <StatusTile label="WAN Exposed" ok={wanExposed} detail={wanExposed ? 'Yes' : (authLoading ? '…' : 'No')} />
            </div>

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

          {/* ── Service status grid ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Server size={13} /> Service Status
            </div>
            <div className="visualize-status-grid">
              <StatusTile label="DNS (Unbound)" ok={entry.unbound_status?.configured ?? false} detail={entry.unbound_status?.ip || 'Not configured'} />
              <StatusTile label="AdGuard" ok={entry.adguard_status?.configured ?? false} detail={entry.adguard_status?.ip || 'Not configured'} />
              <StatusTile label="DHCP" ok={entry.dhcp_status?.configured ?? false} detail={entry.dhcp_status?.ip || 'Not configured'} />
              <StatusTile label="Cloudflare" ok={cf?.configured ?? false} detail={cf?.configured ? cf.tunnel_name : 'Not configured'} />
            </div>
          </div>

          {/* ── Mobile note ── */}
          <div className="visualize-mobile-note">
            <Smartphone size={12} /> Diagrams wrap vertically on mobile.
          </div>
        </div>
      </div>
    </div>
  );
}

function StatusTile({ label, ok, detail }: { label: string; ok: boolean; detail: string }) {
  return (
    <div className={`visualize-status-tile ${ok ? 'ok' : 'missing'}`}>
      <span className="visualize-status-label">{label}</span>
      <span className="visualize-status-detail">{detail}</span>
    </div>
  );
}
