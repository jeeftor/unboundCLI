import { describe, it, expect } from 'vitest';
import { getHostnameDecision } from '../src/lib/hostnameDecision';
import type { Entry } from '../src/types';

const baseEntry: Entry = {
  hostname: 'adsb.vookie.net',
  caddy_upstream: '192.168.1.81:1099',
  caddy_ip: '192.168.1.81',
  caddy_port: '1099',
  unbound_status: { configured: true, ip: '192.168.1.15', in_sync: true },
  adguard_status: { configured: true, ip: '192.168.1.15', in_sync: true },
  dhcp_status: {
    configured: true,
    type: 'static',
    ip: '192.168.1.82',
    mac: '00:11:22:33:44:55',
    hostname: 'adsb',
    in_sync: false
  },
  dns_resolved: '192.168.1.15',
  cloudflare_status: {
    configured: false,
    tunnel_name: '',
    tunnel_id: '',
    service: '',
    path: '',
    is_default_tunnel: false,
    http_host_header: '',
    origin_server_name: '',
    no_tls_verify: false,
    http2_origin: false,
    has_access_policy: false,
    has_dns_record: false
  },
  overall_status: 1,
  status_label: 'Synced',
  data_source: 'Caddy',
  has_forward_auth: false,
  has_auth_bypass_risk: false
};

describe('getHostnameDecision', () => {
  it('flags a Caddy-backed hostname that collides with a DHCP machine name', () => {
    const decision = getHostnameDecision(baseEntry, '192.168.1.15');

    expect(decision.kind).toBe('collision');
    expect(decision.severity).toBe('warning');
    expect(decision.title).toBe('Direct access risk');
    expect(decision.summary).toMatch(/ssh adsb\.vookie\.net/i);
    expect(decision.facts).toEqual([
      'DNS resolves to Caddy at 192.168.1.15',
      'DHCP has adsb at 192.168.1.82',
      'Caddy proxies this name to 192.168.1.81:1099'
    ]);
    expect(decision.actions.map(action => action.label)).toEqual([
      'Keep web, add SSH alias',
      'Keep SSH, add web alias'
    ]);
  });

  it('treats a Caddy service without a matching DHCP hostname as informational', () => {
    const decision = getHostnameDecision(
      {
        ...baseEntry,
        hostname: 'ai-pages.vookie.net',
        dhcp_status: { ...baseEntry.dhcp_status, configured: false, ip: '', hostname: '' }
      },
      '192.168.1.15'
    );

    expect(decision.kind).toBe('service');
    expect(decision.severity).toBe('info');
    expect(decision.title).toBe('Caddy service');
  });
});
