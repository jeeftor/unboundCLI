import {
  CircleAlert,
} from 'lucide-react';
import { useCallback, useEffect, useRef, useState, type MouseEvent } from 'react';
import { api } from '../api/client';
import type { Entry } from '../types';

const DNS_RETRY_INTERVAL_MS = 10_000;

export function DNSProbe({ hostname }: { hostname: string }) {
  const [state, setState] = useState<'idle' | 'loading' | 'ok' | 'fail'>('idle');
  const [tip, setTip] = useState('');
  const [countdown, setCountdown] = useState(0);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => { mountedRef.current = false; };
  }, []);

  const probe = useCallback(async () => {
    setState('loading');
    try {
      const res = await api.dnsProbe(hostname);
      if (!mountedRef.current) return;
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
      if (!mountedRef.current) return;
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

export function CFRepairBanner({ entries, cfEnabled }: {
  entries: Entry[];
  cfEnabled: boolean;
}) {
  const missingCNAME = entries.filter(e =>
    e.cloudflare_status?.configured && !e.cloudflare_status?.has_dns_record
  );

  if (!cfEnabled || missingCNAME.length === 0) return null;

  return (
    <div className="cf-repair-banner">
      <CircleAlert size={14} />
      <span>
        <strong>{missingCNAME.length} CF tunnel {missingCNAME.length === 1 ? 'entry is' : 'entries are'} missing a DNS CNAME record</strong>
        {' '}&mdash; they won&apos;t resolve publicly until fixed.
      </span>
      <span className="cf-repair-guidance">Open each hostname and choose Sync to preview its ownership-checked repair.</span>
    </div>
  );
}
