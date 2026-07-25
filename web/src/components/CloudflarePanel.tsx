import {
  CheckCircle2,
  CircleAlert,
  Loader2,
  Trash2
} from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import type { MouseEvent } from 'react';
import { api } from '../api/client';
import type { Entry } from '../types';

const DNS_RETRY_INTERVAL_MS = 10_000;

export function DNSProbe({ hostname }: { hostname: string }) {
  const [state, setState] = useState<'idle' | 'loading' | 'ok' | 'fail'>('idle');
  const [tip, setTip] = useState('');
  const [countdown, setCountdown] = useState(0);

  const probe = useCallback(async () => {
    setState('loading');
    try {
      const res = await api.dnsProbe(hostname);
      if (res.resolved) {
        const detail = res.cname ? `-> ${res.cname}` : (res.addresses?.[0] ?? '');
        setTip(`Resolves via 1.1.1.1: ${detail}`);
        setState('ok');
      } else {
        setTip(res.error ? `Not resolving: ${res.error}` : 'Not resolving yet');
        setState('fail');
        setCountdown(DNS_RETRY_INTERVAL_MS / 1000);
      }
    } catch {
      setTip('Probe failed');
      setState('fail');
      setCountdown(DNS_RETRY_INTERVAL_MS / 1000);
    }
  }, [hostname]);

  const handleClick = useCallback((e: MouseEvent) => {
    e.stopPropagation();
    void probe();
  }, [probe]);

  // Auto-retry countdown when failing
  useEffect(() => {
    if (state !== 'fail') return;
    if (countdown <= 0) {
      void probe();
      return;
    }
    const id = setTimeout(() => setCountdown(c => c - 1), 1000);
    return () => clearTimeout(id);
  }, [state, countdown, probe]);

  if (state === 'idle') {
    return <button type="button" className="dns-probe-btn" onClick={handleClick} title="Check if hostname resolves via 1.1.1.1">DNS?</button>;
  }
  if (state === 'loading') {
    return <span className="dns-probe-badge loading" title="Probing...">...</span>;
  }
  if (state === 'ok') {
    return <span className="dns-probe-badge ok" title={tip}>DNS</span>;
  }
  // fail -- show countdown until next retry, click to retry immediately
  return (
    <button type="button" className="dns-probe-badge fail" title={tip} onClick={handleClick}>
      DNS {countdown > 0 ? `(${countdown}s)` : ''}
    </button>
  );
}

export function CloudflareDetails({ status, hostname }: { status: Entry['cloudflare_status']; hostname: string }) {
  if (!status?.configured) return <span className="cloudflare-detail missing"><strong>Not routed</strong><span>No tunnel rule</span></span>;
  return (
    <span className={`cloudflare-detail ${status.http_host_header ? 'ok' : 'bad'}`}>
      <strong>{status.tunnel_name || 'Tunnel'}</strong>
      <span>{status.service || '-'}</span>
      <span>{status.http_host_header ? `Host header ${status.http_host_header}` : 'Missing HTTPHostHeader'}</span>
      <span>{status.has_access_policy ? 'Access policy' : 'No access policy'}</span>
      <DNSProbe hostname={hostname} />
    </span>
  );
}

export function CloudflareRoutePanel({ entry, caddyServerIP, mutationEnabled, onRefresh }: {
  entry: Entry;
  caddyServerIP: string;
  mutationEnabled: boolean;
  onRefresh: () => void;
}) {
  const cf = entry.cloudflare_status;
  const [saving, setSaving] = useState(false);
  const [removed, setRemoved] = useState(false);
  const [error, setError] = useState('');
  const [dnsWarning, setDnsWarning] = useState('');

  const viaService = cf.service ?? '';
  const isViaCaddy = !!(caddyServerIP && viaService.includes(caddyServerIP));

  // Derive scheme and host from caddy_upstream for direct mode
  const upstreamRaw = entry.caddy_upstream ?? '';
  const upstreamHasScheme = upstreamRaw.startsWith('http://') || upstreamRaw.startsWith('https://');
  const upstreamBase = upstreamHasScheme ? upstreamRaw.replace(/^https?:\/\//, '') : upstreamRaw;

  const [directScheme, setDirectScheme] = useState<'http' | 'https'>(
    upstreamRaw.startsWith('https://') ? 'https' : 'http'
  );
  const [noTLSVerify, setNoTLSVerify] = useState(cf.no_tls_verify ?? false);

  const directService = upstreamBase ? `${directScheme}://${upstreamBase}` : '';
  const caddyService = caddyServerIP ? `https://${caddyServerIP}` : '';

  const setRoute = async (mode: 'caddy' | 'direct') => {
    setSaving(true);
    setError('');
    setDnsWarning('');
    try {
      const service = mode === 'caddy' ? caddyService : directService;
      const httpHostHeader = mode === 'caddy' ? entry.hostname : '';
      const res = await api.cfSetRoute({ hostname: entry.hostname, service, http_host_header: httpHostHeader, no_tls_verify: mode === 'direct' ? noTLSVerify : false });
      if (res.dns_warning) setDnsWarning(res.dns_warning);
      onRefresh();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed');
    } finally {
      setSaving(false);
    }
  };

  const removeRoute = async () => {
    setSaving(true);
    setError('');
    try {
      await api.cfRemoveRoute(entry.hostname);
      setRemoved(true);
      onRefresh();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed');
    } finally {
      setSaving(false);
    }
  };

  const configured = cf.configured && !removed;

  return (
    <div className="cf-route-panel">
      <div className="cf-route-header">
        <span className="cf-route-title">Cloudflare Tunnel</span>
        {configured
          ? <span className={`service-sync-badge ok`}>{isViaCaddy ? 'Via Caddy' : 'Direct'}</span>
          : <span className="service-sync-badge missing">{removed ? 'Removed' : 'Not in tunnel'}</span>
        }
      </div>

      {configured && (
        <div className="cf-route-service muted" title={viaService}>{viaService}</div>
      )}
      {configured && cf.has_access_policy && (
        <div className="cf-route-tag">Access policy</div>
      )}

      {mutationEnabled && !removed && upstreamBase && !isViaCaddy && (
        <div className="cf-route-direct-opts">
          <span className="cf-route-label">Direct scheme:</span>
          <button type="button" className={`btn-sm${directScheme === 'http' ? ' btn-primary' : ''}`} onClick={() => setDirectScheme('http')}>http</button>
          <button type="button" className={`btn-sm${directScheme === 'https' ? ' btn-primary' : ''}`} onClick={() => setDirectScheme('https')}>https</button>
          {directScheme === 'https' && (
            <label className="cf-route-tls-label">
              <input type="checkbox" checked={noTLSVerify} onChange={e => setNoTLSVerify(e.target.checked)} />
              Skip TLS verify
            </label>
          )}
        </div>
      )}

      {mutationEnabled && !removed && (
        <div className="cf-route-actions">
          {configured ? (
            <>
              {isViaCaddy ? (
                <button type="button" className="btn-sm" onClick={() => void setRoute('direct')} disabled={saving || !directService} title={directService}>
                  {saving ? <Loader2 size={11} className="spin" /> : null} Switch to Direct
                </button>
              ) : (
                <button type="button" className="btn-sm" onClick={() => void setRoute('caddy')} disabled={saving || !caddyService} title={caddyService}>
                  {saving ? <Loader2 size={11} className="spin" /> : null} Switch to Caddy
                </button>
              )}
              <button type="button" className="btn-sm btn-danger-sm" onClick={() => void removeRoute()} disabled={saving}>
                {saving ? <Loader2 size={11} className="spin" /> : <Trash2 size={12} />} Remove
              </button>
            </>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6, alignItems: 'flex-start' }}>
              {caddyService && (
                <button type="button" className="btn-sm" onClick={() => void setRoute('caddy')} disabled={saving}>
                  {saving ? <Loader2 size={11} className="spin" /> : null} Route via Caddy ({caddyServerIP})
                </button>
              )}
              {directService && (
                <button type="button" className="btn-sm" onClick={() => void setRoute('direct')} disabled={saving}>
                  {saving ? <Loader2 size={11} className="spin" /> : null} Route direct ({directService})
                </button>
              )}
            </div>
          )}
          {saving && <span className="cf-route-saving"><Loader2 size={12} className="spin" /> Updating tunnel + DNS...</span>}
        </div>
      )}

      {error && <div className="cf-route-error">{error}</div>}
      {dnsWarning && <div className="cf-route-dns-warning">{dnsWarning}</div>}
    </div>
  );
}

export function CFRepairBanner({ entries, mutationEnabled, cfEnabled, onRepaired }: {
  entries: Entry[];
  mutationEnabled: boolean;
  cfEnabled: boolean;
  onRepaired: () => void;
}) {
  const [repairing, setRepairing] = useState(false);
  const [result, setResult] = useState<{ fixed: string[]; failed: string[] } | null>(null);

  const missingCNAME = entries.filter(e =>
    e.cloudflare_status?.configured && !e.cloudflare_status?.has_dns_record
  );

  if (!cfEnabled || missingCNAME.length === 0) return null;

  const repair = async () => {
    setRepairing(true);
    setResult(null);
    try {
      const r = await api.cfRepairDNS();
      setResult(r);
      onRepaired();
    } catch { /* ignore */ } finally {
      setRepairing(false);
    }
  };

  return (
    <div className="cf-repair-banner">
      <CircleAlert size={14} />
      <span>
        <strong>{missingCNAME.length} CF tunnel {missingCNAME.length === 1 ? 'entry is' : 'entries are'} missing a DNS CNAME record</strong>
        {' '}&mdash; they won&apos;t resolve publicly until fixed.
      </span>
      {mutationEnabled && (
        <button type="button" className="btn-sm btn-primary" onClick={() => void repair()} disabled={repairing}>
          {repairing ? <Loader2 size={11} className="spin" /> : null}
          {repairing ? ' Repairing...' : 'Repair all missing CNAMEs'}
        </button>
      )}
      {result && result.fixed.length > 0 && (
        <span className="cf-repair-ok">
          <CheckCircle2 size={12} /> Fixed: {result.fixed.join(', ')}
        </span>
      )}
      {result && result.failed.length > 0 && (
        <span className="cf-repair-fail">Failed: {result.failed.join(', ')}</span>
      )}
    </div>
  );
}
