import '../styles/VisualizeModal.css';
import { Globe, Network, Server, Smartphone, Wifi, X } from 'lucide-react';
import { FlowArrow, FlowExplanation, FlowNode, FlowRow } from './FlowDiagram';
import type { Entry } from '../types';

export function VisualizeModal({ entry, onClose }: { entry: Entry; onClose: () => void }) {
  const cf = entry.cloudflare_status;
  const hasCF = cf?.configured;
  const hasCFAccess = cf?.has_access_policy;
  const hasForwardAuth = entry.has_forward_auth;
  const hasDNS = Boolean(entry.dns_resolved && entry.dns_resolved !== 'FAIL');
  const upstream = entry.caddy_upstream || 'unknown';

  return (
    <div className="modal-overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="modal visualize-modal">
        <div className="modal-header">
          <h3><Network size={15} /> {entry.hostname}</h3>
          <button type="button" className="modal-close" onClick={onClose}><X size={16} /></button>
        </div>
        <div className="modal-body visualize-body">

          {/* ── WAN path ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Globe size={13} /> WAN Path (Internet → App)
            </div>
            <div className="auth-flow-diagram">
              <FlowRow>
                <FlowNode variant="wan" label="Internet" />
                <FlowArrow label={hasCFAccess ? 'CF Access' : undefined} active={hasCF} />
                <FlowNode variant="cf" label="Cloudflare" sublabel={cf?.tunnel_name || undefined} active={hasCF} />
                <FlowArrow label={hasForwardAuth ? 'Forward Auth' : undefined} active={hasCF} />
                <FlowNode variant="caddy" label="Caddy" sublabel={entry.caddy_ip || undefined} active={hasCF} />
                <FlowArrow active={hasCF} />
                <FlowNode variant="upstream" label="Upstream" sublabel={upstream} active={hasCF} />
              </FlowRow>
              <FlowExplanation>
                {hasCF ? (
                  <p>
                    <strong>WAN traffic</strong>: Internet → Cloudflare Tunnel
                    {hasCFAccess ? ' (CF Access login required)' : ' (no access policy — open)'}
                    {' → '}Caddy
                    {hasForwardAuth ? ' (forward_auth → Authentik)' : ''}
                    {' → '}Upstream <code>{upstream}</code>
                  </p>
                ) : (
                  <p><strong>WAN traffic</strong>: Not exposed — no Cloudflare tunnel configured. This host is only reachable on the LAN.</p>
                )}
              </FlowExplanation>
            </div>
          </div>

          {/* ── LAN path ── */}
          <div className="visualize-section">
            <div className="visualize-section-title">
              <Wifi size={13} /> LAN Path (Internal → App)
            </div>
            <div className="auth-flow-diagram">
              <FlowRow>
                <FlowNode variant="app" label="LAN Client" sublabel={entry.dns_resolved || undefined} active={hasDNS} />
                <FlowArrow label="DNS" active={hasDNS} />
                <FlowNode variant="dns" label="Unbound" sublabel={entry.unbound_status?.ip || undefined} active={hasDNS} />
                <FlowArrow active />
                <FlowNode variant="caddy" label="Caddy" sublabel={entry.caddy_ip || undefined} />
                <FlowArrow label={hasForwardAuth ? 'Forward Auth' : undefined} />
                <FlowNode variant="upstream" label="Upstream" sublabel={upstream} />
              </FlowRow>
              <FlowExplanation>
                <p>
                  <strong>LAN traffic</strong>: Client resolves <code>{entry.hostname}</code>
                  {hasDNS ? ` → ${entry.dns_resolved}` : ' (not in DNS)'}
                  {' → '}Caddy{hasForwardAuth ? ' (forward_auth → Authentik)' : ''}
                  {' → '}Upstream <code>{upstream}</code>
                </p>
              </FlowExplanation>
            </div>
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
              <StatusTile label="CF Access" ok={cf?.has_access_policy ?? false} detail={cf?.has_access_policy ? 'Protected' : 'No policy'} />
              <StatusTile label="Forward Auth" ok={hasForwardAuth} detail={hasForwardAuth ? 'Authentik' : 'None'} />
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
